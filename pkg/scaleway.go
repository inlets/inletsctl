package pkg

import (
	"log"
	"strings"
	"time"

	"github.com/inlets/inlets-operator/pkg/provision"
	instance "github.com/scaleway/scaleway-sdk-go/api/instance/v1"
	"github.com/scaleway/scaleway-sdk-go/scw"
)

// ScalewayProvisioner provision a VM on scaleway.com
type ScalewayProvisioner struct {
	instanceAPI *instance.API
}

// NewScalewayProvisioner with an accessKey and secretKey
func NewScalewayProvisioner(accessKey, secretKey, organizationID string) (*ScalewayProvisioner, error) {
	client, err := scw.NewClient(
		scw.WithAuth(accessKey, secretKey),
		scw.WithDefaultOrganizationID(organizationID),
		scw.WithDefaultZone(scw.ZoneFrPar1),
	)
	if err != nil {
		return nil, err
	}

	return &ScalewayProvisioner{
		instanceAPI: instance.NewAPI(client),
	}, nil
}

// Provision creates a new scaleway instance from the BasicHost type
// To provision the instance we first create the server, then set its user-data and power it on
func (s *ScalewayProvisioner) Provision(host provision.BasicHost) (*provision.ProvisionedHost, error) {
	log.Printf("Provisioning host with Scaleway\n")

	if host.OS == "" {
		host.OS = "ubuntu-bionic"
	}

	if host.Plan == "" {
		host.Plan = "DEV1-S"
	}

	res, err := s.instanceAPI.CreateServer(&instance.CreateServerRequest{
		Name:              host.Name,
		CommercialType:    host.Plan,
		Image:             host.OS,
		DynamicIPRequired: scw.BoolPtr(true),
	})

	if err != nil {
		return nil, err
	}

	server := res.Server

	err = s.instanceAPI.SetServerUserData(&instance.SetServerUserDataRequest{
		ServerID: server.ID,
		Key:      "cloud-init",
		Content:  strings.NewReader(host.UserData),
	})

	if err != nil {
		return nil, err
	}

	err = s.instanceAPI.ServerActionAndWait(&instance.ServerActionAndWaitRequest{
		ServerID: server.ID,
		Action:   instance.ServerActionPoweron,
	})

	if err != nil {
		return nil, err
	}

	return serverToProvisionedHost(server), nil

}

// Status returns the status of the scaleway instance
func (s *ScalewayProvisioner) Status(id string) (*provision.ProvisionedHost, error) {
	res, err := s.instanceAPI.GetServer(&instance.GetServerRequest{
		ServerID: id,
	})

	if err != nil {
		return nil, err
	}

	return serverToProvisionedHost(res.Server), nil
}

// Delete deletes the provisionned instance by ID
// We should first poweroff the instance,
// otherwise we'll have: http error 400 Bad Request: instance should be powered off.
// Then we have to delete the server and attached volumes
func (s *ScalewayProvisioner) Delete(id string) error {
	server, err := s.instanceAPI.GetServer(&instance.GetServerRequest{
		ServerID: id,
	})

	err = s.instanceAPI.ServerActionAndWait(&instance.ServerActionAndWaitRequest{
		ServerID: id,
		Action:   instance.ServerActionPoweroff,
		Timeout:  5 * time.Minute,
	})

	if err != nil {
		return err
	}

	err = s.instanceAPI.DeleteServer(&instance.DeleteServerRequest{
		ServerID: id,
	})

	if err != nil {
		return err
	}

	for _, volume := range server.Server.Volumes {
		err := s.instanceAPI.DeleteVolume(&instance.DeleteVolumeRequest{
			VolumeID: volume.ID,
		})

		if err != nil {
			return err
		}
	}

	return nil
}

func serverToProvisionedHost(server *instance.Server) *provision.ProvisionedHost {
	var ip = ""
	if server.PublicIP != nil {
		ip = server.PublicIP.Address.String()
	}
	state := server.State.String()
	if state == "running" {
		state = "active"
	}
	return &provision.ProvisionedHost{
		ID:     server.ID,
		IP:     ip,
		Status: state,
	}
}
