// Copyright (c) Inlets Author(s) 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package cmd

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"time"

	names "github.com/inlets/inletsctl/pkg/names"
	provision "github.com/inlets/inletsctl/pkg/provision"

	"github.com/pkg/errors"
	password "github.com/sethvargo/go-password/password"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {
	inletsCmd.AddCommand(createCmd)
	createCmd.Flags().StringP("provider", "p", "digitalocean", "The cloud provider - digitalocean, gce, ec2, packet, scaleway, or civo")
	createCmd.Flags().StringP("region", "r", "lon1", "The region for your cloud provider")
	createCmd.Flags().StringP("zone", "z", "us-central1-a", "The zone for the exit node (Google Compute Engine)")

	createCmd.Flags().StringP("inlets-token", "t", "", "The inlets auth token for your exit node")
	createCmd.Flags().StringP("access-token", "a", "", "The access token for your cloud")
	createCmd.Flags().StringP("access-token-file", "f", "", "Read this file for the access token for your cloud")

	createCmd.Flags().String("secret-key", "", "The access token for your cloud (Scaleway, EC2)")
	createCmd.Flags().String("secret-key-file", "", "Read this file for the access token for your cloud (Scaleway, EC2)")
	createCmd.Flags().String("organisation-id", "", "Organisation ID (Scaleway)")
	createCmd.Flags().String("project-id", "", "Project ID (Packet.com, Google Compute Engine)")

	createCmd.Flags().StringP("remote-tcp", "c", "", `Remote host for inlets-pro to use for forwarding TCP connections`)

	createCmd.Flags().DurationP("poll", "n", time.Second*2, "poll every N seconds")
}

// clientCmd represents the client sub command.
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an exit node",
	Long: `Create an exit node on your preferred cloud

  Example: inletsctl create --provider digitalocean`,
	RunE:          runCreate,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func runCreate(cmd *cobra.Command, _ []string) error {

	provider, err := cmd.Flags().GetString("provider")
	if err != nil {
		return errors.Wrap(err, "failed to get 'provider' value.")
	}

	fmt.Printf("Using provider: %s\n", provider)

	inletsToken, err := cmd.Flags().GetString("inlets-token")
	if err != nil {
		return errors.Wrap(err, "failed to get 'inlets-token' value.")
	}
	if len(inletsToken) == 0 {
		var passwordErr error
		inletsToken, passwordErr = generateAuth()

		if passwordErr != nil {
			return passwordErr
		}
	}

	var poll time.Duration
	pollOverride, pollOverrideErr := cmd.Flags().GetDuration("poll")
	if pollOverrideErr == nil {
		poll = pollOverride
	}

	accessToken, err := getFileOrString(cmd.Flags(), "access-token-file", "access-token", true)
	if err != nil {
		return err
	}

	var region string
	if cmd.Flags().Changed("region") {
		if regionVal, err := cmd.Flags().GetString("region"); len(regionVal) > 0 {
			if err != nil {
				return errors.Wrap(err, "failed to get 'region' value.")
			}
			region = regionVal
		}

	} else if provider == "scaleway" {
		region = "fr-par-1"
	} else if provider == "packet" {
		region = "ams1"
	} else if provider == "ec2" {
		region = "eu-west-1"
	}

	var zone string
	if provider == "gce" {
		zone, err = cmd.Flags().GetString("zone")
	}

	var secretKey string
	var organisationID string
	if provider == "scaleway" || provider == "ec2" {

		var secretKeyErr error
		secretKey, secretKeyErr = getFileOrString(cmd.Flags(), "secret-key-file", "secret-key", true)
		if secretKeyErr != nil {
			return secretKeyErr
		}

		if provider == "scaleway" {
			organisationID, _ = cmd.Flags().GetString("organisation-id")
			if len(organisationID) == 0 {
				return fmt.Errorf("--organisation-id cannot be empty")
			}
		}
	}

	provisioner, err := getProvisioner(provider, accessToken, secretKey, organisationID, region)

	if err != nil {
		return err
	}

	remoteTCP, _ := cmd.Flags().GetString("remote-tcp")

	name := strings.Replace(names.GetRandomName(10), "_", "-", -1)

	inletsControlPort := 8080

	userData := makeUserdata(inletsToken, inletsControlPort, remoteTCP)

	projectID, _ := cmd.Flags().GetString("project-id")

	hostReq, err := createHost(provider, name, region, zone, projectID, userData, strconv.Itoa(inletsControlPort))
	if err != nil {
		return err
	}

	if provider == "gce" {
		fmt.Printf("Requesting host: %s in %s, from %s\n", name, zone, provider)
	} else {
		fmt.Printf("Requesting host: %s in %s, from %s\n", name, region, provider)
	}

	hostRes, err := provisioner.Provision(*hostReq)
	if err != nil {
		return err
	}

	fmt.Printf("Host: %s, status: %s\n", hostRes.ID, hostRes.Status)

	max := 500
	for i := 0; i < max; i++ {
		time.Sleep(poll)

		hostStatus, err := provisioner.Status(hostRes.ID)
		if err != nil {
			return err
		}

		fmt.Printf("[%d/%d] Host: %s, status: %s\n",
			i+1, max, hostStatus.ID, hostStatus.Status)

		if hostStatus.Status == "active" {
			if len(remoteTCP) == 0 {
				fmt.Printf(`Inlets OSS exit-node summary:
  IP: %s
  Auth-token: %s

Command:
  export UPSTREAM=http://127.0.0.1:8000
  inlets client --remote "ws://%s:%d" \
	--token "%s" \
	--upstream $UPSTREAM

To Delete:
  inletsctl delete --provider %s --id "%s"
`,
					hostStatus.IP, inletsToken, hostStatus.IP, inletsControlPort, inletsToken, provider, hostStatus.ID)
				return nil
			}

			proPort := 8123
			fmt.Printf(`inlets-pro exit-node summary:
  IP: %s
  Auth-token: %s

Command:
  export TCP_PORTS="8000"
  export LICENSE=""
  inlets-pro client --connect "wss://%s:%d/connect" \
	--token "%s" \
	--license "$LICENSE" \
	--tcp-ports 8000
`,
				hostStatus.IP, inletsToken, hostStatus.IP, proPort, inletsToken)

			return nil
		}
	}

	return err
}

func getProvisioner(provider, accessToken, secretKey, organisationID, region string) (provision.Provisioner, error) {
	if provider == "digitalocean" {
		return provision.NewDigitalOceanProvisioner(accessToken)
	} else if provider == "packet" {
		return provision.NewPacketProvisioner(accessToken)
	} else if provider == "civo" {
		return provision.NewCivoProvisioner(accessToken)
	} else if provider == "scaleway" {
		return provision.NewScalewayProvisioner(accessToken, secretKey, organisationID, region)
	} else if provider == "gce" {
		return provision.NewGCEProvisioner(accessToken)
	} else if provider == "ec2" {
		return provision.NewEC2Provisioner(region, accessToken, secretKey)
	}
	return nil, fmt.Errorf("no provisioner for provider: %s", provider)
}

func generateAuth() (string, error) {
	pwdRes, pwdErr := password.Generate(64, 10, 0, false, true)
	return pwdRes, pwdErr
}

func createHost(provider, name, region, zone, projectID, userData, inletsPort string) (*provision.BasicHost, error) {
	if provider == "digitalocean" {
		return &provision.BasicHost{
			Name:       name,
			OS:         "ubuntu-16-04-x64",
			Plan:       "512mb",
			Region:     region,
			UserData:   userData,
			Additional: map[string]string{},
		}, nil
	} else if provider == "packet" {
		return &provision.BasicHost{
			Name:     name,
			OS:       "ubuntu_16_04",
			Plan:     "t1.small.x86",
			Region:   region,
			UserData: userData,
			Additional: map[string]string{
				"project_id": projectID,
			},
		}, nil
	} else if provider == "scaleway" {
		return &provision.BasicHost{
			Name:       name,
			OS:         "ubuntu-bionic",
			Plan:       "DEV1-S",
			Region:     region,
			UserData:   userData,
			Additional: map[string]string{},
		}, nil
	} else if provider == "civo" {
		return &provision.BasicHost{
			Name:       name,
			OS:         "811a8dfb-8202-49ad-b1ef-1e6320b20497",
			Plan:       "g2.small",
			Region:     region,
			UserData:   userData,
			Additional: map[string]string{},
		}, nil
	} else if provider == "gce" {
		return &provision.BasicHost{
			Name:     name,
			OS:       "projects/debian-cloud/global/images/debian-9-stretch-v20191121",
			Plan:     "f1-micro",
			Region:   "",
			UserData: userData,
			Additional: map[string]string{
				"projectid":     projectID,
				"zone":          zone,
				"firewall-name": "inlets",
				"firewall-port": inletsPort,
			},
		}, nil
	} else if provider == "ec2" {
		// Ubuntu images can be found here https://cloud-images.ubuntu.com/locator/ec2/
		// Name is used in the OS field so the ami can be lookup up in the region specified
		return &provision.BasicHost{
			Name:     name,
			OS:       "ubuntu/images/hvm-ssd/ubuntu-xenial-16.04-amd64-server-20191114",
			Plan:     "t3.nano",
			Region:   region,
			UserData: base64.StdEncoding.EncodeToString([]byte(userData)),
			Additional: map[string]string{
				"inlets-port": inletsPort,
			},
		}, nil
	}

	return nil, fmt.Errorf("no provisioner for provider: %q", provider)
}

func makeUserdata(authToken string, inletsControlPort int, remoteTCP string) string {

	controlPort := fmt.Sprintf("%d", inletsControlPort)

	if len(remoteTCP) == 0 {
		return `#!/bin/bash
export AUTHTOKEN="` + authToken + `"
export CONTROLPORT="` + controlPort + `"
curl -sLS https://get.inlets.dev | sh

curl -sLO https://raw.githubusercontent.com/inlets/inlets/master/hack/inlets-operator.service  && \
	mv inlets-operator.service /etc/systemd/system/inlets.service && \
	echo "AUTHTOKEN=$AUTHTOKEN" > /etc/default/inlets && \
	echo "CONTROLPORT=$CONTROLPORT" >> /etc/default/inlets && \
	systemctl start inlets && \
	systemctl enable inlets`
	}

	return `#!/bin/bash
	export AUTHTOKEN="` + authToken + `"
	export REMOTETCP="` + remoteTCP + `"
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

func getFileOrString(flags *pflag.FlagSet, file, value string, required bool) (string, error) {
	var val string
	fileVal, _ := flags.GetString(file)
	if len(fileVal) > 0 {
		res, err := ioutil.ReadFile(fileVal)
		if err != nil {
			return "", err
		}
		val = strings.TrimSpace(string(res))
	} else {

		flagVal, err := flags.GetString(value)
		if err != nil {
			return "", errors.Wrap(err, "failed to get '"+value+"' value.")
		}
		val = flagVal
	}

	if required && len(val) == 0 {
		return "", fmt.Errorf("give a value for --%s or --%s", file, value)
	}

	return val, nil
}
