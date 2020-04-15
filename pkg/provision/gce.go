package provision

import (
	"context"
	"fmt"
	"log"
	"net/http"
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

	err := p.createInletsFirewallRule(host.Additional["projectid"], host.Additional["firewall-name"], host.Additional["firewall-port"], host.Additional["pro"])
	if err != nil {
		return nil, err
	}

	// instance auto restart on failure
	autoRestart := true

	if host.Additional["pro"] == "true" && host.Additional["tmp"] == "true" {
		host.Plan = "e2-micro"
	}

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
					DiskSizeGb:  15,
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

	op, err := p.gceProvisioner.Instances.Insert(
		host.Additional["projectid"],
		host.Additional["zone"],
		instance).Do()

	if err != nil {
		return nil, fmt.Errorf("could not provision GCE instance: %s", err)
	}

	if op.HTTPStatusCode == http.StatusConflict {
		log.Println("Host already exists, status: conflict.")
	}

	return &ProvisionedHost{
		ID:     toGCEID(host.Name, host.Additional["zone"], host.Additional["projectid"]),
		Status: "provisioning",
	}, nil
}

// gceFirewallExists checks if the inlets firewall rule exists or not
func (p *GCEProvisioner) gceFirewallExists(projectID string, firewallRuleName string) (bool, error) {
	op, err := p.gceProvisioner.Firewalls.Get(projectID, firewallRuleName).Do()
	if err != nil {
		return false, fmt.Errorf("could not get inlets firewall rule: %v", err)
	}
	if op.Name == firewallRuleName {
		return true, nil
	}
	return false, nil
}

// createInletsFirewallRule creates a firewall rule opening up the control port for inlets
func (p *GCEProvisioner) createInletsFirewallRule(projectID string, firewallRuleName string, controlPort string, pro string) error {
	var firewallRule *compute.Firewall
	if pro == "true" {
		firewallRule = &compute.Firewall{
			Name:        firewallRuleName,
			Description: "Firewall rule created by inlets-operator",
			Network:     fmt.Sprintf("projects/%s/global/networks/default", projectID),
			Allowed: []*compute.FirewallAllowed{
				{
					IPProtocol: "tcp",
				},
			},
			SourceRanges: []string{"0.0.0.0/0"},
			Direction:    "INGRESS",
			TargetTags:   []string{"inlets"},
		}
	} else {
		firewallRule = &compute.Firewall{
			Name:        firewallRuleName,
			Description: "Firewall rule created by inlets-operator",
			Network:     fmt.Sprintf("projects/%s/global/networks/default", projectID),
			Allowed: []*compute.FirewallAllowed{
				{
					IPProtocol: "tcp",
					Ports:      []string{controlPort, "80", "443"},
				},
			},
			SourceRanges: []string{"0.0.0.0/0"},
			Direction:    "INGRESS",
			TargetTags:   []string{"inlets"},
		}
	}

	exists, _ := p.gceFirewallExists(projectID, firewallRuleName)
	if exists {
		log.Printf("Creating firewall exists, updating: %s\n", firewallRuleName)

		_, err := p.gceProvisioner.Firewalls.Update(projectID, firewallRuleName, firewallRule).Do()
		if err != nil {
			return fmt.Errorf("could not update inlets firewall rule %s, error: %s", firewallRuleName, err)
		}
		return nil
	}

	_, err := p.gceProvisioner.Firewalls.Insert(projectID, firewallRule).Do()
	log.Printf("Creating firewall rule: %s\n", firewallRuleName)
	if err != nil {
		return fmt.Errorf("could not create inlets firewall rule: %v", err)
	}
	return nil
}

// Delete deletes the GCE exit node
func (p *GCEProvisioner) Delete(request HostDeleteRequest) error {
	var instanceName, projectID, zone string
	var err error
	if len(request.ID) > 0 {
		instanceName, zone, projectID, err = getGCEFieldsFromID(request.ID)
		if err != nil {
			return err
		}
	} else {
		inletsID, err := p.lookupID(request)
		if err != nil {
			return err
		}
		instanceName, zone, projectID, err = getGCEFieldsFromID(inletsID)
		if err != nil {
			return err
		}
	}

	if len(request.ProjectID) > 0 {
		projectID = request.ProjectID
	}

	if len(request.Zone) > 0 {
		zone = request.Zone
	}

	log.Printf("Deleting GCE host: %s, %s, %s\n", projectID, zone, instanceName)

	_, err = p.gceProvisioner.Instances.Delete(projectID, zone, instanceName).Do()
	if err != nil {
		return fmt.Errorf("could not delete the GCE instance: %v", err)
	}
	return err
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

	var ip string
	if len(op.NetworkInterfaces) > 0 {
		ip = op.NetworkInterfaces[0].AccessConfigs[0].NatIP
	}

	status := gceToInletsStatus(op.Status)

	return &ProvisionedHost{
		IP:     ip,
		ID:     toGCEID(instanceName, zone, projectID),
		Status: status,
	}, nil
}

func gceToInletsStatus(gce string) string {
	status := gce
	if status == gceHostRunning {
		status = ActiveStatus
	}
	return status
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
