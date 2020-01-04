package provision

import (
	"fmt"
	"net/http"

	"github.com/packethost/packngo"
)

// PacketProvisioner provision a host on packet.com
type PacketProvisioner struct {
	client *packngo.Client
}

// NewPacketProvisioner with an accessKey
func NewPacketProvisioner(accessKey string) (*PacketProvisioner, error) {
	return &PacketProvisioner{
		client: packngo.NewClientWithAuth("", accessKey, http.DefaultClient),
	}, nil
}

// Status returns the IP, ID and Status of the exit node
func (p *PacketProvisioner) Status(id string) (*ProvisionedHost, error) {
	device, _, err := p.client.Devices.Get(id, nil)

	if err != nil {
		return nil, err
	}

	state := device.State

	ip := ""
	for _, network := range device.Network {
		if network.Public {
			ip = network.IpAddressCommon.Address
			break
		}
	}

	return &ProvisionedHost{
		ID:     device.ID,
		Status: state,
		IP:     ip,
	}, nil
}

// Delete terminates the exit node
func (p *PacketProvisioner) Delete(request HostDeleteRequest) error {
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
	_, err = p.client.Devices.Delete(id)
	return err
}

// Provision deploys an exit node into packet.com
func (p *PacketProvisioner) Provision(host BasicHost) (*ProvisionedHost, error) {
	if host.Region == "" {
		host.Region = "ams1"
	}

	createReq := &packngo.DeviceCreateRequest{
		Plan:         host.Plan,
		Facility:     []string{host.Region},
		Hostname:     host.Name,
		ProjectID:    host.Additional["project_id"],
		SpotInstance: false,
		OS:           host.OS,
		BillingCycle: "hourly",
		UserData:     host.UserData,
		Tags:         []string{"inlets"},
	}

	device, _, err := p.client.Devices.Create(createReq)

	if err != nil {
		return nil, err
	}

	return &ProvisionedHost{
		ID: device.ID,
	}, nil
}

// List returns a list of exit nodes
func (p *PacketProvisioner) List(filter ListFilter) ([]*ProvisionedHost, error) {
	var inlets []*ProvisionedHost
	devices, _, err := p.client.Devices.List(filter.ProjectID, nil)
	if err != nil {
		return nil, err
	}
	for _, device := range devices {
		for _, tag := range device.Tags {
			if tag == filter.Filter {
				net := device.GetNetworkInfo()
				host := &ProvisionedHost{
					IP: net.PublicIPv4,
					ID: device.ID,
				}
				inlets = append(inlets, host)
			}
		}
	}
	return inlets, nil
}

func (p *PacketProvisioner) lookupID(request HostDeleteRequest) (string, error) {
	inlets, err := p.List(ListFilter{Filter: "inlets", ProjectID: request.ProjectID})
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
