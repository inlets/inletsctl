package provision

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/hetznercloud/hcloud-go/hcloud"
)

// HetznerProvisioner provision a VM on hetzner.com
type HetznerProvisioner struct {
	client *hcloud.Client
}

// NewHetznerProvisioner with an accessKey
func NewHetznerProvisioner(accessKey string) (*HetznerProvisioner, error) {
	client := hcloud.NewClient(hcloud.WithToken(accessKey))
	return &HetznerProvisioner{
		client: client,
	}, nil
}

func (p *HetznerProvisioner) Status(id string) (*ProvisionedHost, error) {
	server, err := p.getServer(id)
	if err != nil {
		return nil, err
	}
	ip := server.PublicNet.IPv4.IP.String()

	status := string(server.Status)
	if status == "running" {
		status = ActiveStatus
	}

	return &ProvisionedHost{
		ID:     id,
		Status: status,
		IP:     ip,
	}, nil
}

func (p *HetznerProvisioner) Delete(id string) error {
	server, err := p.getServer(id)
	if err != nil {
		return err
	}
	_, err = p.client.Server.Delete(context.Background(), server)
	return err
}

func (p *HetznerProvisioner) Provision(host BasicHost) (*ProvisionedHost, error) {
	log.Printf("Provisioning host with Hetzner\n")

	if host.Region == "" {
		host.Region = "fsn1"
	}

	serverType, _, err := p.client.ServerType.GetByName(context.Background(), host.Plan)
	if err != nil {
		return nil, err
	}
	if serverType == nil {
		return nil, fmt.Errorf("Server type %s does not exist!", host.Plan)
	}

	location, _, err := p.client.Location.GetByName(context.Background(), host.Region)
	if err != nil {
		return nil, err
	}
	if location == nil {
		return nil, fmt.Errorf("Location %s does not exist!", host.Region)
	}

	image, _, err := p.client.Image.GetByName(context.Background(), host.OS)
	if err != nil {
		return nil, err
	}
	if image == nil {
		return nil, fmt.Errorf("Image %s does not exist!", host.OS)
	}

	createOpts := hcloud.ServerCreateOpts{
		Name:       host.Name,
		Location:   location,
		ServerType: serverType,
		Image:      image,
		UserData:   host.UserData,
	}
	server, _, err := p.client.Server.Create(context.Background(), createOpts)
	if err != nil {
		return nil, err
	}

	return &ProvisionedHost{
		ID: fmt.Sprintf("%d", server.Server.ID),
	}, nil
}

func (p *HetznerProvisioner) getServer(id string) (*hcloud.Server, error) {
	sid, _ := strconv.Atoi(id)

	server, _, err := p.client.Server.GetByID(context.Background(), sid)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, fmt.Errorf("hetzner client returned nil server for %d", sid)
	}

	return server, nil
}
