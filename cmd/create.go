// Copyright (c) Inlets Author(s) 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package cmd

import (
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/inlets/inletsctl/pkg/env"

	"github.com/inlets/inletsctl/pkg/names"
	"github.com/inlets/inletsctl/pkg/provision"

	"github.com/pkg/errors"
	"github.com/sethvargo/go-password/password"
	"github.com/spf13/cobra"
)

const inletsOSSVersion = "2.7.4"
const inletsPROVersion = "0.7.0"

const inletsOSSControlPort = 8080
const inletsProControlPort = 8123

func init() {

	inletsCmd.AddCommand(createCmd)

	createCmd.Flags().StringP("provider", "p", "digitalocean", "The cloud provider - digitalocean, gce, ec2, azure, packet, scaleway, linode, civo, hetzner or vultr")
	createCmd.Flags().StringP("region", "r", "lon1", "The region for your cloud provider")
	createCmd.Flags().StringP("zone", "z", "us-central1-a", "The zone for the exit-server (Google Compute Engine)")

	createCmd.Flags().StringP("inlets-token", "t", "", "The auth token for the inlets server on your new exit-server, leave blank to auto-generate")
	createCmd.Flags().StringP("access-token", "a", "", "The access token for your cloud")
	createCmd.Flags().StringP("access-token-file", "f", "", "Read this file for the access token for your cloud")

	createCmd.Flags().String("vpc-id", "", "The VPC ID to create the exit-server in (EC2)")
	createCmd.Flags().String("subnet-id", "", "The Subnet ID where the exit-server should be placed (EC2)")
	createCmd.Flags().String("secret-key", "", "The access token for your cloud (Scaleway, EC2)")
	createCmd.Flags().String("secret-key-file", "", "Read this file for the access token for your cloud (Scaleway, EC2)")
	createCmd.Flags().String("organisation-id", "", "Organisation ID (Scaleway)")
	createCmd.Flags().String("project-id", "", "Project ID (Packet.com, Google Compute Engine)")
	createCmd.Flags().String("subscription-id", "", "Subscription ID (Azure)")

	createCmd.Flags().Bool("pro", false, `Provision an exit-server for use with inlets PRO`)

	createCmd.Flags().DurationP("poll", "n", time.Second*2, "poll every N seconds, use a higher value if you encounter rate-limiting")
}

// clientCmd represents the client sub command.
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an exit-server on cloud infrastructure",
	Long: `Create an exit-server on cloud infrastructure with inlets or inlets PRO 
preloaded as a systemd service. The estimated cost of each VM along with 
what OS version and spec will be used is explained in the README.
`,
	Example: `  inletsctl create  \
	--provider [digitalocean|packet|ec2|scaleway|civo|gce|azure|linode|hetzner] \
	--access-token-file $HOME/access-token \
	--region lon1

  # For inlets-pro, give the --pro flag:
  inletsctl create --pro
`,
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

	accessToken, err := env.GetRequiredFileOrString(cmd.Flags(),
		"access-token-file",
		"access-token",
		"INLETS_ACCESS_TOKEN",
	)
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

	} else if provider == "digitalocean" {
		region = "lon1"
	} else if provider == "scaleway" {
		region = "fr-par-1"
	} else if provider == "packet" {
		region = "ams1"
	} else if provider == "ec2" {
		region = "eu-west-1"
	} else if provider == "hetzner" {
		region = "hel1"
	} else if provider == "vultr" {
		region = "LHR" // London
	}

	var zone string
	if provider == "gce" {
		zone, err = cmd.Flags().GetString("zone")
	}

	var secretKey string
	var organisationID string
	var projectID string
	var vpcID string
	var subnetID string
	if provider == "scaleway" || provider == "ec2" {

		var secretKeyErr error
		secretKey, secretKeyErr = getFileOrString(cmd.Flags(), "secret-key-file", "secret-key", true)
		if secretKeyErr != nil {
			return secretKeyErr
		}

		if provider == "scaleway" {
			organisationID, _ = cmd.Flags().GetString("organisation-id")
			if len(organisationID) == 0 {
				return fmt.Errorf("--organisation-id flag must be set")
			}
		}

		if provider == "ec2" {
			vpcID, err = cmd.Flags().GetString("vpc-id")
			if err != nil {
				return errors.Wrap(err, "failed to get 'vpc-id' value")
			}

			subnetID, err = cmd.Flags().GetString("subnet-id")
			if err != nil {
				return errors.Wrap(err, "failed to get 'subnet-id' value")
			}

			if (len(vpcID) == 0 && len(subnetID) > 0) || (len(subnetID) == 0 && len(vpcID) > 0) {
				return fmt.Errorf("both --vpc-id and --subnet-id must be set")
			}
		}

	} else if provider == "gce" || provider == "packet" {
		projectID, _ = cmd.Flags().GetString("project-id")
		if len(projectID) == 0 {
			return fmt.Errorf("--project-id flag must be set")
		}
	}

	var subscriptionID string
	if provider == "azure" {
		subscriptionID, _ = cmd.Flags().GetString("subscription-id")
	}

	provisioner, err := getProvisioner(provider, accessToken, secretKey, organisationID, region, subscriptionID)

	if err != nil {
		return err
	}
	var pro bool

	if v, _ := cmd.Flags().GetBool("pro"); v {
		pro = true
	}

	name := strings.Replace(names.GetRandomName(10), "_", "-", -1)
	userData := provision.MakeExitServerUserdata(inletsOSSControlPort,
		inletsToken,
		inletsOSSVersion,
		inletsPROVersion,
		pro)

	hostReq, err := createHost(provider,
		name,
		region,
		zone,
		projectID,
		userData,
		strconv.Itoa(inletsOSSControlPort),
		vpcID,
		subnetID,
		pro)

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
			if !pro {
				fmt.Printf(`inlets OSS (`+inletsOSSVersion+`) exit-server summary:
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
					hostStatus.IP,
					inletsToken,
					hostStatus.IP,
					inletsOSSControlPort,
					inletsToken,
					provider,
					hostStatus.ID)
				return nil
			}

			fmt.Printf(`inlets PRO (`+inletsPROVersion+`) exit-server summary:
  IP: %s
  Auth-token: %s

Command:
  export LICENSE=""
  export PORTS="8000"
  export UPSTREAM="localhost"

  inlets-pro client --url "wss://%s:%d/connect" \
	--token "%s" \
	--license "$LICENSE" \
	--upstream $UPSTREAM \
	--ports $PORTS

To Delete:
	  inletsctl delete --provider %s --id "%s"
`,
				hostStatus.IP,
				inletsToken,
				hostStatus.IP,
				inletsProControlPort,
				inletsToken,
				provider,
				hostStatus.ID)

			return nil
		}
	}

	return err
}

func getProvisioner(provider, accessToken, secretKey, organisationID, region, subscriptionID string) (provision.Provisioner, error) {
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
	} else if provider == "azure" {
		return provision.NewAzureProvisioner(subscriptionID, accessToken)
	} else if provider == "linode" {
		return provision.NewLinodeProvisioner(accessToken)
	} else if provider == "hetzner" {
		return provision.NewHetznerProvisioner(accessToken)
	} else if provider == "vultr" {
		return provision.NewVultrProvisioner(accessToken)
	}
	return nil, fmt.Errorf("no provisioner for provider: %s", provider)
}

func generateAuth() (string, error) {
	pwdRes, pwdErr := password.Generate(64, 10, 0, false, true)
	return pwdRes, pwdErr
}

func createHost(provider, name, region, zone, projectID, userData, inletsPort string, vpcID string, subnetID string, pro bool) (*provision.BasicHost, error) {
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
				"pro":           fmt.Sprint(pro),
			},
		}, nil
	} else if provider == "ec2" {
		// Ubuntu images can be found here https://cloud-images.ubuntu.com/locator/ec2/
		// Name is used in the OS field so the ami can be lookup up in the region specified

		var additional = map[string]string{
			"inlets-port": inletsPort,
			"pro":         fmt.Sprint(pro),
		}

		if len(vpcID) > 0 {
			additional["vpc-id"] = vpcID
		}

		if len(subnetID) > 0 {
			additional["subnet-id"] = subnetID
		}

		return &provision.BasicHost{
			Name:       name,
			OS:         "ubuntu/images/hvm-ssd/ubuntu-xenial-16.04-amd64-server-20191114",
			Plan:       "t3.nano",
			Region:     region,
			UserData:   base64.StdEncoding.EncodeToString([]byte(userData)),
			Additional: additional,
		}, nil
	} else if provider == "azure" {
		// Ubuntu images can be found here https://docs.microsoft.com/en-us/azure/virtual-machines/linux/cli-ps-findimage#list-popular-images
		// An image includes more than one property, it has publisher, offer, sku and version.
		// So they have to be in "Additional" instead of just "OS".
		return &provision.BasicHost{
			Name:     name,
			OS:       "Additional.imageOffer",
			Plan:     "Standard_B1ls",
			Region:   region,
			UserData: userData,
			Additional: map[string]string{
				"inlets-port":    inletsPort,
				"pro":            fmt.Sprint(pro),
				"imagePublisher": "Canonical",
				"imageOffer":     "UbuntuServer",
				"imageSku":       "16.04-LTS",
				"imageVersion":   "latest",
			},
		}, nil
	} else if provider == "vultr" {
		// OS:
		//  A complete list of available OS is available using: https://api.vultr.com/v1/os/list
		//  215 = Ubuntu 16.04 x64
		// Plans:
		//  A complete list of available OS is available using: https://api.vultr.com/v1/plans/list
		//  201 = 1024 MB RAM,25 GB SSD,1.00 TB BW
		return &provision.BasicHost{
			Name:       name,
			OS:         "215",
			Plan:       "201",
			Region:     region,
			UserData:   userData,
			Additional: map[string]string{},
		}, nil
	} else if provider == "linode" {
		// Image:
		//  List of images can be retrieved using: https://api.linode.com/v4/images
		//  Example response: .."id": "linode/ubuntu16.04lts", "label": "Ubuntu 16.04 LTS"..
		// Type:
		//  Type is the VM plan / size in linode.
		//  List of type and price can be retrieved using curl https://api.linode.com/v4/linode/types
		return &provision.BasicHost{
			Name:     name,
			OS:       "linode/ubuntu16.04lts",
			Plan:     "g6-nanode-1",
			Region:   region,
			UserData: userData,
			Additional: map[string]string{
				"inlets-port": inletsPort,
				"pro":         fmt.Sprint(pro),
			},
		}, nil
	} else if provider == "hetzner" {
		// Easiest way to get the information of available images and server types is through
		// the Hetzner API, but it requires auth for any type of call.
		// Images can be fetched from https://api.hetzner.cloud/v1/images
		// Server types can be fetched from https://api.hetzner.cloud/v1/server_types
		// The regions available are hel1 (Helsinki), nur1 (Nuremberg), fsn1 (Falkenstein)
		return &provision.BasicHost{
			Name:     name,
			Region:   region,
			Plan:     "cx11",
			OS:       "ubuntu-16.04",
			UserData: userData,
		}, nil
	}

	return nil, fmt.Errorf("no provisioner for provider: %q", provider)
}
