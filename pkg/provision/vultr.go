package provision

import (
	"context"
	"fmt"
	"github.com/vultr/govultr"
	"strconv"
	"strings"
)

const vultrHostRunning = "ok"
const exiteNodeTag = "inlets-exit-node"

type VultrProvisioner struct {
	client *govultr.Client
}

func NewVultrProvisioner(accessKey string) (*VultrProvisioner, error) {
	return &VultrProvisioner{
		client: govultr.NewClient(nil, accessKey),
	}, nil
}

func (v *VultrProvisioner) Provision(host BasicHost) (*ProvisionedHost, error) {

	script, err := v.client.StartupScript.Create(context.Background(), host.Name, host.UserData, "boot")
	if err != nil {
		return nil, err
	}

	region, err := v.lookupRegion(host.Region)
	if err != nil {
		return nil, err
	}

	plan, err := strconv.Atoi(host.Plan)
	if err != nil {
		return nil, err
	}

	os, err := strconv.Atoi(host.OS)
	if err != nil {
		return nil, err
	}

	opts := &govultr.ServerOptions{
		ScriptID: script.ScriptID,
		Hostname: host.Name,
		Label:    host.Name,
		Tag:      exiteNodeTag,
	}

	result, err := v.client.Server.Create(context.Background(), *region, plan, os, opts)
	if err != nil {
		return nil, err
	}

	return &ProvisionedHost{
		IP:     result.MainIP,
		ID:     result.InstanceID,
		Status: result.ServerState,
	}, nil
}

func (v *VultrProvisioner) Status(id string) (*ProvisionedHost, error) {
	server, err := v.client.Server.GetServer(context.Background(), id)
	if err != nil {
		return nil, err
	}

	status := server.ServerState
	if status == "ok" {
		status = ActiveStatus
	}

	return &ProvisionedHost{
		IP:     server.MainIP,
		ID:     server.InstanceID,
		Status: status,
	}, nil
}

func (v *VultrProvisioner) Delete(request HostDeleteRequest) error {
	var id string
	var err error
	if len(request.ID) > 0 {
		id = request.ID
	} else {
		id, err = v.lookupID(request)
		if err != nil {
			return err
		}
	}

	server, err := v.client.Server.GetServer(context.Background(), id)
	if err != nil {
		return err
	}

	err = v.client.Server.Delete(context.Background(), id)
	if err != nil {
		return err
	}

	scripts, err := v.client.StartupScript.List(context.Background())
	for _, s := range scripts {
		if s.Name == server.Label {
			_ = v.client.StartupScript.Delete(context.Background(), s.ScriptID)
			break
		}
	}

	return nil
}

// List returns a list of exit nodes
func (v *VultrProvisioner) List(filter ListFilter) ([]*ProvisionedHost, error) {
	servers, err := v.client.Server.ListByTag(context.Background(), filter.Filter)
	if err != nil {
		return nil, err
	}

	var inlets []*ProvisionedHost
	for _, server := range servers {
		host := &ProvisionedHost{
			IP:     server.MainIP,
			ID:     server.InstanceID,
			Status: vultrToInletsStatus(server.Status),
		}
		inlets = append(inlets, host)
	}

	return inlets, nil
}

func (v *VultrProvisioner) lookupID(request HostDeleteRequest) (string, error) {

	inlets, err := v.List(ListFilter{Filter: exiteNodeTag, ProjectID: request.ProjectID})
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

func (v *VultrProvisioner) lookupRegion(id string) (*int, error) {
	result, err := strconv.Atoi(id)
	if err == nil {
		return &result, nil
	}

	regions, err := v.client.Region.List(context.Background())
	if err != nil {
		return nil, err
	}

	for _, region := range regions {
		if strings.EqualFold(id, region.RegionCode) || strings.EqualFold(id, region.Name) {
			regionId, _ := strconv.Atoi(region.RegionID)
			return &regionId, nil
		}
	}

	return nil, fmt.Errorf("region '%s' not available", id)
}

func vultrToInletsStatus(vultr string) string {
	status := vultr
	if status == vultrHostRunning {
		status = ActiveStatus
	}
	return status
}
