package provision

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws/credentials"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/google/uuid"
)

// EC2Provisioner contains the EC2 client
type EC2Provisioner struct {
	ec2Provisioner *ec2.EC2
}

// NewEC2Provisioner creates an EC2Provisioner and initialises an EC2 client
func NewEC2Provisioner(region, accessKey, secretKey string) (*EC2Provisioner, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(region),
		Credentials: credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	svc := ec2.New(sess)
	return &EC2Provisioner{ec2Provisioner: svc}, err
}

// Provision deploys an exit node into AWS EC2
func (p *EC2Provisioner) Provision(host BasicHost) (*ProvisionedHost, error) {
	image, err := p.lookupAMI(host.OS)
	if err != nil {
		return nil, err
	}

	port, err := strconv.Atoi(host.Additional["inlets-port"])
	if err != nil {
		return nil, err
	}
	pro := host.Additional["pro"]

	groupID, name, err := p.createEC2SecurityGroup(port, pro)
	if err != nil {
		return nil, err
	}

	runResult, err := p.ec2Provisioner.RunInstances(&ec2.RunInstancesInput{
		ImageId:      image,
		InstanceType: aws.String(host.Plan),
		MinCount:     aws.Int64(1),
		MaxCount:     aws.Int64(1),
		UserData:     &host.UserData,
		NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
			{
				DeviceIndex:              aws.Int64(int64(0)),
				AssociatePublicIpAddress: aws.Bool(true),
				DeleteOnTermination:      aws.Bool(true),
				Groups:                   []*string{groupID},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(runResult.Instances) == 0 {
		return nil, fmt.Errorf("could not create host: %s", runResult.String())
	}

	_, err = p.ec2Provisioner.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{runResult.Instances[0].InstanceId},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(*name),
			},
			{
				Key:   aws.String("inlets"),
				Value: aws.String("exit-node"),
			},
		},
	})

	return &ProvisionedHost{
		ID:     *runResult.Instances[0].InstanceId,
		Status: "creating",
	}, nil
}

// Status returns the ID, Status and IP of the exit node
func (p *EC2Provisioner) Status(id string) (*ProvisionedHost, error) {
	var status string
	s, err := p.ec2Provisioner.DescribeInstanceStatus(&ec2.DescribeInstanceStatusInput{
		InstanceIds: []*string{aws.String(id)},
	})
	if err != nil {
		return nil, err
	}
	if len(s.InstanceStatuses) > 0 {
		if *s.InstanceStatuses[0].InstanceStatus.Status == "ok" {
			status = ActiveStatus
		} else {
			status = "initialising"
		}
	} else {
		status = "creating"
	}

	d, err := p.ec2Provisioner.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(id)},
	})
	if err != nil {
		return nil, err
	}
	if len(d.Reservations) == 0 {
		return nil, fmt.Errorf("cannot describe host: %s", id)
	}

	return &ProvisionedHost{
		ID:     id,
		Status: status,
		IP:     aws.StringValue(d.Reservations[0].Instances[0].PublicIpAddress),
	}, nil
}

// Delete removes the exit node
func (p *EC2Provisioner) Delete(request HostDeleteRequest) error {
	var id string
	var err error
	if len(request.ID) > 0 {
		id = request.ID
	} else {
		id, err = p.lookupID(request)
		if err != nil {
			return err
		}
	}

	i, err := p.ec2Provisioner.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(id)},
	})
	if err != nil {
		return err
	}
	groups := i.Reservations[0].Instances[0].SecurityGroups

	_, err = p.ec2Provisioner.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{aws.String(id)},
	})
	if err != nil {
		return err
	}

	// Instance has to be terminated before we can remove the security group
	err = p.ec2Provisioner.WaitUntilInstanceTerminated(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(id)},
	})
	if err != nil {
		return err
	}

	for _, group := range groups {
		_, err := p.ec2Provisioner.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
			GroupId: group.GroupId,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

// List returns a list of exit nodes
func (p *EC2Provisioner) List(filter ListFilter) ([]*ProvisionedHost, error) {
	var inlets []*ProvisionedHost
	var nextToken *string
	filterValues := strings.Split(filter.Filter, ",")
	for {
		instances, err := p.ec2Provisioner.DescribeInstances(&ec2.DescribeInstancesInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String(filterValues[0]),
					Values: []*string{aws.String(filterValues[1])},
				},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return nil, err
		}
		for _, r := range instances.Reservations {
			for _, i := range r.Instances {
				if *i.State.Name != ec2.InstanceStateNameTerminated {
					host := &ProvisionedHost{
						ID: *i.InstanceId,
					}
					if i.PublicIpAddress != nil {
						host.IP = *i.PublicIpAddress
					}
					inlets = append(inlets, host)
				}
			}
		}
		nextToken = instances.NextToken
		if nextToken == nil {
			break
		}
	}
	return inlets, nil
}

func (p *EC2Provisioner) lookupID(request HostDeleteRequest) (string, error) {
	inlets, err := p.List(ListFilter{
		Filter:    "tag:inlets,exit-node",
		ProjectID: request.ProjectID,
		Zone:      request.Zone,
	})
	if err != nil {
		return "", err
	}

	for _, inlet := range inlets {
		if inlet.IP == request.IP {
			return inlet.ID, nil
		}
	}
	return "", fmt.Errorf("no host with ip: %s", request.IP)
}

// creteEC2SecurityGroup creates a security group for the exit-node
func (p *EC2Provisioner) createEC2SecurityGroup(controlPort int, pro string) (*string, *string, error) {
	ports := []int{80, 443, controlPort}
	proPorts := []int{1024, 65535}
	groupName := "inlets-" + uuid.New().String()
	group, err := p.ec2Provisioner.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		Description: aws.String("inlets security group"),
		GroupName:   aws.String(groupName),
	})
	if err != nil {
		return nil, nil, err
	}

	for _, port := range ports {
		err = p.createEC2SecurityGroupRule(*group.GroupId, port, port)
		if err != nil {
			return group.GroupId, &groupName, err
		}
	}
	if pro == "true" {
		err = p.createEC2SecurityGroupRule(*group.GroupId, proPorts[0], proPorts[1])
		if err != nil {
			return group.GroupId, &groupName, err
		}
	}

	return group.GroupId, &groupName, nil
}

func (p *EC2Provisioner) createEC2SecurityGroupRule(groupID string, fromPort, toPort int) error {
	_, err := p.ec2Provisioner.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		CidrIp:     aws.String("0.0.0.0/0"),
		FromPort:   aws.Int64(int64(fromPort)),
		IpProtocol: aws.String("tcp"),
		ToPort:     aws.Int64(int64(toPort)),
		GroupId:    aws.String(groupID),
	})
	if err != nil {
		return err
	}
	return nil
}

// lookupAMI gets the AMI ID that the exit node will use
func (p *EC2Provisioner) lookupAMI(name string) (*string, error) {
	images, err := p.ec2Provisioner.DescribeImages(&ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("name"),
				Values: []*string{
					aws.String(name),
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}

	if len(images.Images) == 0 {
		return nil, fmt.Errorf("image not found")
	}
	return images.Images[0].ImageId, nil
}
