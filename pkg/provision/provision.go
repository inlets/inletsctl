package provision

import (
	"fmt"
	"github.com/sethvargo/go-password/password"
)

type Provisioner interface {
	Provision(BasicHost) (*ProvisionedHost, error)
	Status(id string) (*ProvisionedHost, error)
	Delete(id string) error
}

const ActiveStatus = "active"

const DigitaloceanProvider = "digitalocean"
const ScalewayProvider = "scaleway"
const GCEProvider = "gce"
const PacketProvider = "packet"
const CivoProvider = "civo"
const EC2Provider = "ec2"

const ControlPort = "8080"

type ProvisionerDefaults struct {
	OS     string
	Plan   string
	Region string
	Zone   string
}

var Defaults = map[string]ProvisionerDefaults{
	DigitaloceanProvider: {
		OS:   "ubuntu-16-04-x64",
		Plan: "512mb",
		Region: "lon1",
	},
	PacketProvider: {
		OS:     "ubuntu_16_04",
		Plan:   "t1.small.x86",
		Region: "ams1",
	},
	ScalewayProvider: {
		OS:     "ubuntu-bionic",
		Plan:   "DEV1-S",
		Region: "fr-par-1",
	},
	CivoProvider: {
		OS:   "811a8dfb-8202-49ad-b1ef-1e6320b20497",
		Plan: "g2.small",
		Region: "lon1",
	},
	GCEProvider: {
		OS:   "projects/debian-cloud/global/images/debian-9-stretch-v20191121",
		Plan: "f1-micro",
		Zone: "us-central1-a",
	},
	EC2Provider: {
		OS:     "ubuntu/images/hvm-ssd/ubuntu-xenial-16.04-amd64-server-20191114",
		Plan:   "t3.nano",
		Region: "eu-west-1",
	},
}

type ProvisionerRequest struct {
	Provider       string
	AccessToken    string
	SecretKey      string
	OrganisationID string
	Region         string
}

func NewProvisioner(request ProvisionerRequest) (Provisioner, error) {
	if request.Provider == DigitaloceanProvider {
		return NewDigitalOceanProvisioner(request.AccessToken)
	} else if request.Provider == PacketProvider {
		return NewPacketProvisioner(request.AccessToken)
	} else if request.Provider == CivoProvider {
		return NewCivoProvisioner(request.AccessToken)
	} else if request.Provider == ScalewayProvider {
		return NewScalewayProvisioner(request.AccessToken, request.SecretKey, request.OrganisationID, request.Region)
	} else if request.Provider == GCEProvider {
		return NewGCEProvisioner(request.AccessToken)
	} else if request.Provider == EC2Provider {
		return NewEC2Provisioner(request.AccessToken, request.SecretKey, request.Region)
	}
	return nil, fmt.Errorf("no provisioner for provider: %s", request.Provider)
}

type ProvisionedHost struct {
	IP     string
	ID     string
	Status string
}

type BasicHost struct {
	Region    string
	Plan      string
	OS        string
	Name      string
	UserData  string
	ProjectID string
	Zone      string
}

func NewBasicHost(provider, name, region, projectID, zone, userData string) (*BasicHost, error) {
	if _, ok := Defaults[provider]; !ok {
		return nil, fmt.Errorf("no provisioner for provider: %q", provider)
	}
	host := &BasicHost{
		Name:      name,
		OS:        Defaults[provider].OS,
		Plan:      Defaults[provider].Plan,
		UserData:  userData,
		ProjectID: projectID,
	}
	if region == "" && len(Defaults[provider].Region) != 0 {
		host.Region = Defaults[provider].Region
	} else {
		host.Region = region
	}
	if zone == "" && len(Defaults[provider].Zone) != 0 {
		host.Zone = Defaults[provider].Zone
	} else {
		host.Zone = zone
	}
	return host, nil
}

func GenerateAuth() (string, error) {
	pwdRes, pwdErr := password.Generate(64, 10, 0, false, true)
	return pwdRes, pwdErr
}

type UserDataRequest struct {
	AuthToken string
	InletsControlPort string
	RemoteTCP string
}

func MakeUserdata(request UserDataRequest) string {
	if len(request.RemoteTCP) == 0 {
		return `#!/bin/bash
export AUTHTOKEN="` + request.AuthToken + `"
export CONTROLPORT="` + request.InletsControlPort + `"
curl -sLS https://get.inlets.dev | sh

curl -sLO https://raw.githubusercontent.com/inlets/inlets/master/hack/inlets-operator.service  && \
	mv inlets-operator.service /etc/systemd/system/inlets.service && \
	echo "AUTHTOKEN=$AUTHTOKEN" > /etc/default/inlets && \
	echo "CONTROLPORT=$CONTROLPORT" >> /etc/default/inlets && \
	systemctl start inlets && \
	systemctl enable inlets`
	}

	return `#!/bin/bash
	export AUTHTOKEN="` + request.AuthToken + `"
	export REMOTETCP="` + request.RemoteTCP + `"
	export IP=$(curl -sfSL https://ifconfig.co)

	curl -SLsf https://github.com/inlets/inlets-pro-pkg/releases/download/0.4.0/inlets-pro-linux > inlets-pro-linux && \
	chmod +x ./inlets-pro-linux  && \
	mv ./inlets-pro-linux /usr/local/bin/inlets-pro

	curl -sLO https://raw.githubusercontent.com/inlets/inlets/master/hack/inlets-pro.service  && \
		mv inlets-pro.service /etc/systemd/system/inlets-pro.service && \
		echo "AUTHTOKEN=$AUTHTOKEN" >> /etc/default/inlets-pro && \
		echo "REMOTETCP=$REMOTETCP" >> /etc/default/inlets-pro && \
		echo "IP=$IP" >> /etc/default/inlets-pro && \
		systemctl start inlets-pro && \
		systemctl enable inlets-pro`
}