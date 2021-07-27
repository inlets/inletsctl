// Copyright (c) Inlets Author(s) 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package cmd

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/inlets/cloud-provision/provision"

	"github.com/inlets/inletsctl/pkg/env"
	"github.com/inlets/inletsctl/pkg/names"

	"github.com/pkg/errors"
	"github.com/sethvargo/go-password/password"
	"github.com/spf13/cobra"
)

const inletsProDefaultVersion = "0.8.7"
const inletsProControlPort = 8123

func init() {

	inletsCmd.AddCommand(createCmd)

	createCmd.Flags().StringP("provider", "p", "digitalocean", "The cloud provider - digitalocean, gce, ec2, azure, equinix-metal, scaleway, linode, civo, hetzner or vultr")
	createCmd.Flags().StringP("region", "r", "lon1", "The region for your cloud provider")
	createCmd.Flags().StringP("plan", "s", "", "The plan or size for your cloud instance")
	createCmd.Flags().StringP("zone", "z", "us-central1-a", "The zone for the exit-server (gce)")

	createCmd.Flags().StringP("inlets-token", "t", "", "The auth token for the inlets server on your new exit-server, leave blank to auto-generate")
	createCmd.Flags().StringP("access-token", "a", "", "The access token for your cloud")
	createCmd.Flags().StringP("access-token-file", "f", "", "Read this file for the access token for your cloud")

	createCmd.Flags().String("vpc-id", "", "The VPC ID to create the exit-server in (ec2)")
	createCmd.Flags().String("subnet-id", "", "The Subnet ID where the exit-server should be placed (ec2)")
	createCmd.Flags().String("secret-key", "", "The access token for your cloud (scaleway, ec2)")
	createCmd.Flags().String("secret-key-file", "", "Read this file for the access token for your cloud (scaleway, ec2)")
	createCmd.Flags().String("session-token", "", "The session token for ec2 (when using with temporary credentials)")
	createCmd.Flags().String("session-token-file", "", "Read this file for the session token for ec2 (when using with temporary credentials)")

	createCmd.Flags().String("organisation-id", "", "Organisation ID (scaleway)")
	createCmd.Flags().String("project-id", "", "Project ID (equinix-metal, gce)")
	createCmd.Flags().String("subscription-id", "", "Subscription ID (Azure)")

	createCmd.Flags().Bool("tcp", true, `Provision an exit-server with inlets PRO running as a TCP server`)

	createCmd.Flags().StringArray("letsencrypt-domain", []string{}, `Domains you want to get a Let's Encrypt certificate for`)
	createCmd.Flags().String("letsencrypt-issuer", "prod", `The issuer endpoint to use with Let's Encrypt - \"prod\" or \"staging\"`)
	createCmd.Flags().String("letsencrypt-email", "", `The email to register with Let's Encrypt for renewal notices (required)`)

	createCmd.Flags().Bool("pro", true, `Provision an exit-server with inlets PRO (Deprecated)`)
	_ = createCmd.Flags().MarkHidden("pro")
	createCmd.Flags().DurationP("poll", "n", time.Second*2, "poll every N seconds, use a higher value if you encounter rate-limiting")

	createCmd.Flags().String("inlets-pro-version", inletsProDefaultVersion, `Binary release version for inlets PRO`)

}

// clientCmd represents the client sub command.
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an exit-server with inlets PRO preinstalled.",
	Long: `Create an exit-server with inlets PRO preinstalled on cloud infrastructure 
with inlets PRO preloaded as a systemd service. The estimated cost of each 
VM along with what OS version and spec will be used is explained in the 
project docs.`,
	Example: `  inletsctl create  \
    --provider [digitalocean|equinix-metal|ec2|scaleway|civo|gce|azure|linode|hetzner] \
    --access-token-file $HOME/access-token \
    --region lon1`,
	RunE:          runCreate,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// EquinixMetalProvider is a constant string for Equinix Metal
const EquinixMetalProvider = "equinix-metal"

func runCreate(cmd *cobra.Command, _ []string) error {

	provider, err := cmd.Flags().GetString("provider")
	if err != nil {
		return errors.Wrap(err, "failed to get 'provider' value.")
	}

	// Migrate to new name
	if provider == "packet" {
		provider = EquinixMetalProvider
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
	} else if provider == EquinixMetalProvider {
		region = "ams1"
	} else if provider == "ec2" {
		region = "eu-west-1"
	} else if provider == "hetzner" {
		region = "hel1"
	} else if provider == "vultr" {
		region = "LHR" // London
	} else if provider == "linode" {
		region = "eu-west"
	}

	var zone string
	if provider == "gce" {
		zone, err = cmd.Flags().GetString("zone")
		if err != nil {
			return errors.Wrap(err, "failed to get 'zone' value")
		}
	}

	var secretKey string
	var sessionToken string
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
			var sessionTokenErr error
			sessionToken, sessionTokenErr = getFileOrString(cmd.Flags(), "session-token-file", "session-token", false)
			if sessionTokenErr != nil {
				return sessionTokenErr
			}
		}

	} else if provider == "gce" || provider == EquinixMetalProvider {
		projectID, _ = cmd.Flags().GetString("project-id")
		if len(projectID) == 0 {
			return fmt.Errorf("--project-id flag must be set")
		}
	}

	var subscriptionID string
	if provider == "azure" {
		subscriptionID, _ = cmd.Flags().GetString("subscription-id")
	}

	provisioner, err := getProvisioner(provider, accessToken, secretKey, organisationID, region, subscriptionID, sessionToken)

	if err != nil {
		return err
	}

	pro := true
	if cmd.Flags().Changed("pro") {
		fmt.Printf("WARN: --pro is deprecated, use --tcp instead.")
		pro, _ = cmd.Flags().GetBool("pro")
	}
	if cmd.Flags().Changed("tcp") {
		pro, _ = cmd.Flags().GetBool("tcp")
	}

	letsencryptDomains, _ := cmd.Flags().GetStringArray("letsencrypt-domain")
	letsencryptEmail, _ := cmd.Flags().GetString("letsencrypt-email")
	letsencryptIssuer, _ := cmd.Flags().GetString("letsencrypt-issuer")

	if len(letsencryptDomains) > 0 {
		if len(letsencryptEmail) == 0 {
			return fmt.Errorf("--letsencrypt-email is required when --letsencrypt-domain is given")
		}
		if len(letsencryptIssuer) == 0 {
			return fmt.Errorf("--letsencrypt-issuer is required when --letsencrypt-domain is given")
		}
	}

	inletsProVersion, err := cmd.Flags().GetString("inlets-pro-version")
	if err != nil {
		return err
	}
	if len(inletsProVersion) == 0 {
		inletsProVersion = inletsProDefaultVersion
	}

	name := strings.Replace(names.GetRandomName(10), "_", "-", -1)

	var userData string
	if len(letsencryptDomains) > 0 {
		userData = MakeHTTPSUserdata(inletsToken,
			inletsProVersion,
			letsencryptEmail, letsencryptIssuer, letsencryptDomains)
	} else {
		userData = provision.MakeExitServerUserdata(
			inletsToken,
			inletsProVersion)
	}

	hostReq, err := createHost(provider,
		name,
		region,
		zone,
		projectID,
		userData,
		"0",
		vpcID,
		subnetID,
		pro)

	if err != nil {
		return err
	}

	// override default plan/size when provided
	if cmd.Flags().Changed("plan") {
		planOverride, err := cmd.Flags().GetString("plan")
		if err != nil {
			return errors.Wrap(err, "failed to get 'plan' value")
		}
		hostReq.Plan = planOverride
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
			if len(letsencryptDomains) > 0 {
				fmt.Printf(`inlets PRO HTTPS (%s) server summary:
  IP: %s
  HTTPS Domains: %v
  Auth-token: %s

Command:

# Obtain a license at https://inlets.dev
# Store it at $HOME/.inlets/LICENSE or use --help for more options

# Where to route traffic from the inlets server
export UPSTREAM="http://127.0.0.1:8000"

inlets-pro http client --url "wss://%s:%d" \
--token "%s" \
--upstream $UPSTREAM

To delete:
  inletsctl delete --provider %s --id "%s"
`,
					inletsProVersion,
					hostStatus.IP,
					letsencryptDomains,
					inletsToken,
					hostStatus.IP,
					inletsProControlPort,
					inletsToken,
					provider,
					hostStatus.ID)

				return nil
			} else {
				fmt.Printf(`inlets PRO TCP (%s) server summary:
  IP: %s
  Auth-token: %s

Command:

# Obtain a license at https://inlets.dev
# Store it at $HOME/.inlets/LICENSE or use --help for more options
export LICENSE="$HOME/.inlets/LICENSE"

# Give a single value or comma-separated
export PORTS="8000"

# Where to route traffic from the inlets server
export UPSTREAM="localhost"

inlets-pro tcp client --url "wss://%s:%d" \
  --token "%s" \
  --upstream $UPSTREAM \
  --ports $PORTS

To delete:
  inletsctl delete --provider %s --id "%s"
`,
					inletsProVersion,
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
	}

	return err
}

func getProvisioner(provider, accessToken, secretKey, organisationID, region, subscriptionID, sessionToken string) (provision.Provisioner, error) {
	if provider == "digitalocean" {
		return provision.NewDigitalOceanProvisioner(accessToken)
	} else if provider == EquinixMetalProvider {
		return provision.NewEquinixMetalProvisioner(accessToken)
	} else if provider == "civo" {
		return provision.NewCivoProvisioner(accessToken)
	} else if provider == "scaleway" {
		return provision.NewScalewayProvisioner(accessToken, secretKey, organisationID, region)
	} else if provider == "gce" {
		return provision.NewGCEProvisioner(accessToken)
	} else if provider == "ec2" {
		return provision.NewEC2Provisioner(region, accessToken, secretKey, sessionToken)
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
			OS:         "ubuntu-18-04-x64",
			Plan:       "s-1vcpu-1gb",
			Region:     region,
			UserData:   userData,
			Additional: map[string]string{},
		}, nil
	} else if provider == EquinixMetalProvider {
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
			OS:         "ubuntu-focal",
			Plan:       "DEV1-S",
			Region:     region,
			UserData:   userData,
			Additional: map[string]string{},
		}, nil
	} else if provider == "civo" {
		const ubuntu2004ID = "d927ad2f-5073-4ed6-b2eb-b8e61aef29a8"
		return &provision.BasicHost{
			Name:       name,
			OS:         ubuntu2004ID,
			Plan:       "g3.xsmall",
			Region:     region,
			UserData:   userData,
			Additional: map[string]string{},
		}, nil
	} else if provider == "gce" {
		return &provision.BasicHost{
			Name:     name,
			OS:       "projects/ubuntu-os-cloud/global/images/ubuntu-minimal-2004-focal-v20210707",
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
				"imageOffer":     "0001-com-ubuntu-server-focal",
				"imageSku":       "20_04-lts",
				"imageVersion":   "latest",
			},
		}, nil
	} else if provider == "vultr" {
		// OS:
		//  A complete list of available OS is available using: https://api.vultr.com/v1/os/list
		//  387 = Ubuntu 20.04 x64
		// Plans:
		//  A complete list of available OS is available using: https://api.vultr.com/v1/plans/list
		//  201 = 1024 MB RAM,25 GB SSD,1.00 TB BW
		const ubuntu20_04_x64 = "387"
		return &provision.BasicHost{
			Name:       name,
			OS:         ubuntu20_04_x64,
			Plan:       "201",
			Region:     region,
			UserData:   userData,
			Additional: map[string]string{},
		}, nil
	} else if provider == "linode" {
		// Image:
		//  List of images can be retrieved using: https://api.linode.com/v4/images
		//  Example response: .."id": "linode/ubuntu20.04", "label": "Ubuntu 20.04 LTS"..
		// Type:
		//  Type is the VM plan / size in linode.
		//  List of type and price can be retrieved using curl https://api.linode.com/v4/linode/types
		return &provision.BasicHost{
			Name:     name,
			OS:       "linode/ubuntu20.04",
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
			OS:       "ubuntu-20.04",
			UserData: userData,
		}, nil
	}

	return nil, fmt.Errorf("no provisioner for provider: %q", provider)
}

// MakeHTTPSUserdata makes a user-data script in bash to setup inlets
// PRO with a systemd service and the given version.
func MakeHTTPSUserdata(authToken, version, letsEncryptEmail, letsEncryptIssuer string, domains []string) string {

	domainFlags := ""
	for _, domain := range domains {
		domainFlags += fmt.Sprintf("--letsencrypt-domain=%s ", domain)
	}

	return `#!/bin/bash
export AUTHTOKEN="` + authToken + `"
export IP=$(curl -sfSL https://checkip.amazonaws.com)

curl -SLsf https://github.com/inlets/inlets-pro/releases/download/` + version + `/inlets-pro -o /tmp/inlets-pro && \
  chmod +x /tmp/inlets-pro  && \
  mv /tmp/inlets-pro /usr/local/bin/inlets-pro

curl -SLsf https://github.com/inlets/inlets-pro/releases/download/` + version + `/inlets-pro-http.service -o inlets-pro.service && \
  mv inlets-pro.service /etc/systemd/system/inlets-pro.service && \
  echo "AUTHTOKEN=$AUTHTOKEN" >> /etc/default/inlets-pro && \
  echo "IP=$IP" >> /etc/default/inlets-pro && \
  echo "DOMAINS=` + strings.TrimSpace(domainFlags) + `" >> /etc/default/inlets-pro && \
  echo "ISSUER=--letsencrypt-issuer=` + letsEncryptIssuer + `" >> /etc/default/inlets-pro && \
  echo "EMAIL=--letsencrypt-email=` + letsEncryptEmail + `" >> /etc/default/inlets-pro && \
  systemctl daemon-reload && \
  systemctl start inlets-pro && \
  systemctl enable inlets-pro
`
}
