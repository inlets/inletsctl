package provision

import (
	"context"
	"fmt"
	"log"
	"strings"

	"google.golang.org/api/compute/v1"
	"google.golang.org/api/option"
)

const gceHostRunning = "RUNNING"

// GCEProvisioner holds reference to the compute service to provision compute resources
type GCEProvisioner struct {
	gceProvisioner *compute.Service
}

// NewGCEProvisioner returns a new GCEProvisioner
func NewGCEProvisioner(accessKey string) (*GCEProvisioner, error) {
	gceService, err := compute.NewService(context.Background(), option.WithCredentialsJSON([]byte(accessKey)))
	return &GCEProvisioner{
		gceProvisioner: gceService,
	}, err
}

// Provision provisions a new GCE instance as an exit node
func (p *GCEProvisioner) Provision(host BasicHost) (*ProvisionedHost, error) {
	// instance auto restart on failure
	autoRestart := true
	instance := &compute.Instance{
		Name:         host.Name,
		Description:  "Exit node created by inlets-operator",
		MachineType:  fmt.Sprintf("zones/%s/machineTypes/%s", host.Additional["zone"], host.Plan),
		CanIpForward: true,
		Zone:         fmt.Sprintf("projects/%s/zones/%s", host.Additional["projectid"], host.Additional["zone"]),
		Disks: []*compute.AttachedDisk{
			{
				AutoDelete: true,
				Boot:       true,
				DeviceName: host.Name,
				Mode:       "READ_WRITE",
				Type:       "PERSISTENT",
				InitializeParams: &compute.AttachedDiskInitializeParams{
					Description: "Boot Disk for the exit-node created by inlets-operator",
					DiskName:    host.Name,
					DiskSizeGb:  10,
					SourceImage: host.OS,
				},
			},
		},
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{
				{
					Key:   "startup-script",
					Value: &host.UserData,
				},
			},
		},
		Labels: map[string]string{
			"inlets": "exit-node",
		},
		Tags: &compute.Tags{
			Items: []string{
				"http-server",
				"https-server",
				"inlets"},
		},
		Scheduling: &compute.Scheduling{
			AutomaticRestart:  &autoRestart,
			OnHostMaintenance: "MIGRATE",
			Preemptible:       false,
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				AccessConfigs: []*compute.AccessConfig{
					{
						Type: "ONE_TO_ONE_NAT",
						Name: "External NAT",
					},
				},
				Network: "global/networks/default",
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email: "default",
				Scopes: []string{
					compute.ComputeScope,
				},
			},
		},
	}

	exists, _ := p.gceFirewallExists(host.Additional["projectid"], host.Additional["firewall-name"], host.Additional["firewall-port"])

	if !exists {
		err := p.createInletsFirewallRule(host.Additional["projectid"], host.Additional["firewall-name"], host.Additional["firewall-port"])
		log.Println("inlets firewallRule does not exist")
		if err != nil {
			return nil, fmt.Errorf("could not create inlets firewall rule: %v", err)
		}
		log.Printf("Creating inlets firewallRule opening port: %s\n", host.Additional["firewall-port"])
	} else {
		log.Println("inlets firewallRule exists")
	}

	op, err := p.gceProvisioner.Instances.Insert(host.Additional["projectid"], host.Additional["zone"], instance).Do()
	if err != nil {
		return nil, fmt.Errorf("could not provision GCE instance: %v", err)
	}

	status := ""

	if op.Status == gceHostRunning {
		status = ActiveStatus
	}

	return &ProvisionedHost{
		ID:     toGCEID(host.Name, host.Additional["zone"], host.Additional["projectid"]),
		Status: status,
	}, nil
}

// gceFirewallExists checks if the inlets firewall rule exists or not
func (p *GCEProvisioner) gceFirewallExists(projectID string, firewallRuleName string, controlPort string) (bool, error) {
	op, err := p.gceProvisioner.Firewalls.Get(projectID, firewallRuleName).Do()
	if err != nil {
		return false, fmt.Errorf("could not get inlets firewall rule: %v", err)
	}
	if op.Name == firewallRuleName {
		for _, firewallRule := range op.Allowed {
			for _, port := range firewallRule.Ports {
				if port == controlPort {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

// createInletsFirewallRule creates a firewall rule opening up the control port for inlets
func (p *GCEProvisioner) createInletsFirewallRule(projectID string, firewallRuleName string, controlPort string) error {
	firewallRule := &compute.Firewall{
		Name:        firewallRuleName,
		Description: "Firewall rule created by inlets-operator",
		Network:     fmt.Sprintf("projects/%s/global/networks/default", projectID),
		Allowed: []*compute.FirewallAllowed{
			{
				IPProtocol: "tcp",
				Ports:      []string{controlPort},
			},
		},
		SourceRanges: []string{"0.0.0.0/0"},
		Direction:    "INGRESS",
		TargetTags:   []string{"inlets"},
	}

	_, err := p.gceProvisioner.Firewalls.Insert(projectID, firewallRule).Do()
	if err != nil {
		return fmt.Errorf("could not create firewall rule: %v", err)
	}

	return nil
}

// Delete deletes the GCE exit node
func (p *GCEProvisioner) Delete(request HostDeleteRequest) error {
	var instanceName, projectID string
	var err error
	if len(request.ID) > 0 {
		instanceName, _, projectID, err = getGCEFieldsFromID(request.ID)
		if err != nil {
			return err
		}
	} else {
		inletID, err := p.lookupID(request)
		if err != nil {
			return err
		}
		instanceName, _, projectID, err = getGCEFieldsFromID(inletID)
		if err != nil {
			return err
		}
	}
	if len(request.ProjectID) > 0 {
		projectID = request.ProjectID
	}
	_, err = p.gceProvisioner.Instances.Delete(projectID, request.Zone, instanceName).Do()
	if err != nil {
		return fmt.Errorf("could not delete the GCE instance: %v", err)
	}
	return err
}

// List returns a list of exit nodes
func (p *GCEProvisioner) List(filter ListFilter) ([]*ProvisionedHost, error) {
	var inlets []*ProvisionedHost
	var pageToken string
	for {
		call := p.gceProvisioner.Instances.List(filter.ProjectID, filter.Zone).Filter(filter.Filter)
		if len(pageToken) > 0 {
			call = call.PageToken(pageToken)
		}

		instances, err := call.Do()
		if err != nil {
			return inlets, fmt.Errorf("could not list instances: %v", err)
		}
		for _, instance := range instances.Items {
			var status string
			if instance.Status == gceHostRunning {
				status = ActiveStatus
			}
			host := &ProvisionedHost{
				IP:     instance.NetworkInterfaces[0].AccessConfigs[0].NatIP,
				ID:     toGCEID(instance.Name, filter.Zone, filter.ProjectID),
				Status: status,
			}
			inlets = append(inlets, host)
		}
		if len(instances.NextPageToken) == 0 {
			break
		}
	}
	return inlets, nil
}

func (p *GCEProvisioner) lookupID(request HostDeleteRequest) (string, error) {
	inletHosts, err := p.List(ListFilter{
		Filter:    "labels.inlets=exit-node",
		ProjectID: request.ProjectID,
		Zone:      request.Zone,
	})
	if err != nil {
		return "", err
	}

	for _, host := range inletHosts {
		if host.IP == request.IP {
			return host.ID, nil
		}
	}

	return "", fmt.Errorf("no host found with IP: %s", request.IP)
}

// Status checks the status of the provisioning GCE exit node
func (p *GCEProvisioner) Status(id string) (*ProvisionedHost, error) {
	instanceName, zone, projectID, err := getGCEFieldsFromID(id)
	if err != nil {
		return nil, fmt.Errorf("could not get custom GCE fields: %v", err)
	}

	op, err := p.gceProvisioner.Instances.Get(projectID, zone, instanceName).Do()
	if err != nil {
		return nil, fmt.Errorf("could not get instance: %v", err)
	}

	status := ""

	if op.Status == gceHostRunning {
		status = ActiveStatus
	}

	return &ProvisionedHost{
		IP:     op.NetworkInterfaces[0].AccessConfigs[0].NatIP,
		ID:     toGCEID(instanceName, zone, projectID),
		Status: status,
	}, nil
}

// toGCEID creates an ID for GCE based upon the instance ID,
// zone, and projectID fields
func toGCEID(instanceName, zone, projectID string) (id string) {
	return fmt.Sprintf("%s|%s|%s", instanceName, zone, projectID)
}

// get some required fields from the custom GCE instance ID
func getGCEFieldsFromID(id string) (instanceName, zone, projectID string, err error) {
	fields := strings.Split(id, "|")
	err = nil
	if len(fields) == 3 {
		instanceName = fields[0]
		zone = fields[1]
		projectID = fields[2]
	} else {
		err = fmt.Errorf("could not get fields from custom ID: fields: %v", fields)
		return "", "", "", err
	}
	return instanceName, zone, projectID, nil
}
