// Copyright (c) Inlets Author(s) 2023. All rights reserved.
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

const inletsProDefaultVersion = "0.9.32"
const inletsProControlPort = 8123

func init() {

	inletsCmd.AddCommand(createCmd)

	createCmd.Flags().StringP("provider", "p", "digitalocean", "The cloud provider - digitalocean, gce, ec2, azure, scaleway, linode, hetzner, ovh or vultr")
	createCmd.Flags().StringP("region", "r", "lon1", "The region for your cloud provider")
	createCmd.Flags().StringP("plan", "s", "", "The plan or size for your cloud instance")
	createCmd.Flags().StringP("zone", "z", "us-central1-a", "The zone for the exit-server (gce)")

	createCmd.Flags().StringP("inlets-token", "t", "", "The auth token for the inlets server on your new exit-server, leave blank to auto-generate")
	createCmd.Flags().StringP("access-token", "a", "", "The access token for your cloud")
	createCmd.Flags().StringP("access-token-file", "f", "", "Read this file for the access token for your cloud")

	createCmd.Flags().String("vpc-id", "", "The VPC ID to create the exit-server in (ec2)")
	createCmd.Flags().String("subnet-id", "", "The Subnet ID where the exit-server should be placed (ec2)")
	createCmd.Flags().String("secret-key", "", "The secret key for your cloud (scaleway, ec2)")
	createCmd.Flags().String("secret-key-file", "", "Read this file for the secret key for your cloud (scaleway, ec2)")
	createCmd.Flags().String("session-token", "", "The session token for ec2 (when using with temporary credentials)")
	createCmd.Flags().String("session-token-file", "", "Read this file for the session token for ec2 (when using with temporary credentials)")

	createCmd.Flags().String("organisation-id", "", "Organisation ID (scaleway)")
	createCmd.Flags().String("project-id", "", "Project ID (gce, ovh)")
	createCmd.Flags().String("subscription-id", "", "Subscription ID (Azure)")

	createCmd.Flags().String("endpoint", "ovh-eu", "API endpoint (ovh), default: ovh-eu")
	createCmd.Flags().String("consumer-key", "", "The Consumer Key for using the OVH API")

	createCmd.Flags().Bool("tcp", false, `Provision an exit-server with inlets running as a TCP server`)
	createCmd.Flags().String("aws-key-name", "", "The name of an existing SSH key on AWS to be used to access the EC2 instance for maintenance (optional)")

	createCmd.Flags().StringArray("letsencrypt-domain", []string{}, `Domains you want to get a Let's Encrypt certificate for`)
	createCmd.Flags().String("letsencrypt-issuer", "prod", `The issuer endpoint to use with Let's Encrypt - "prod" or "staging"`)
	createCmd.Flags().String("letsencrypt-email", "", `The email to register with Let's Encrypt for renewal notices (required)`)

	createCmd.Flags().DurationP("poll", "n", time.Second*2, "poll every N seconds, use a higher value if you encounter rate-limiting")

	createCmd.Flags().String("inlets-version", inletsProDefaultVersion, `Binary release version for inlets`)
}

// clientCmd represents the client sub command.
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an exit-server with inlets preinstalled.",
	Long: `Create an exit-server with inlets preinstalled on cloud infrastructure 
with inlets preloaded as a systemd service. The estimated cost of each 
VM along with what OS version and spec will be used is explained in the 
project docs.`,
	Example: `
  # Create a HTTPS tunnel server, terminating TLS with a certificate 
  # from Let's Encrypt called "tunnel-richardcase" so your team mates
  # don't delete your VM unintentionally.
  inletsctl create  \
    tunnel-richardcase \
    --letsencrypt-domain inlets.example.com \
    --letsencrypt-email webmaster@example.com

  # Create a TCP tunnel server with a VM name of ssh-tunnel
  inletsctl create \
    ssh-tunnel \
	--tcp \
    --provider [digitalocean|ec2|scaleway|gce|azure|linode|hetzner] \
    --access-token-file $HOME/access-token \
    --region lon1

  # Create a HTTPS tunnel server with multiple domains and an auto-generated
  # VM name
  inletsctl create  \
    --letsencrypt-domain tunnel1.example.com \
    --letsencrypt-domain tunnel2.example.com \
    --letsencrypt-email webmaster@example.com
`,
	RunE:          runCreate,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func runCreate(cmd *cobra.Command, _ []string) error {

	// Get name from the Args, if not provided, generate a random name
	name := strings.Replace(names.GetRandomName(10), "_", "-", -1)
	if len(cmd.Flags().Args()) > 0 {
		name = cmd.Flags().Args()[0]
	}

	inletsProVersion, err := cmd.Flags().GetString("inlets-version")
	if err != nil {
		return err
	}

	if len(inletsProVersion) == 0 {
		inletsProVersion = inletsProDefaultVersion
	}

	tcp := false
	if cmd.Flags().Changed("tcp") {
		tcp, _ = cmd.Flags().GetBool("tcp")
	}

	awsKeyName, err := cmd.Flags().GetString("aws-key-name")
	if err != nil {
		return err
	}

	provider, err := cmd.Flags().GetString("provider")
	if err != nil {
		return err
	}

	serverMode := "L4 TCP"
	if !tcp {
		serverMode = "L7 HTTPS"
	}

	fmt.Printf("inletsctl version: %v\nTunnel server: %s\tProvider: %s\tinlets-pro version: %s\n",
		getVersion(),
		serverMode, provider, inletsProVersion)

	inletsToken, err := cmd.Flags().GetString("inlets-token")
	if err != nil {
		return err
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
	} else if provider == "ec2" {
		region = "eu-west-1"
	} else if provider == "hetzner" {
		region = "hel1"
	} else if provider == "vultr" {
		region = "LHR" // London
	} else if provider == "linode" {
		region = "eu-west"
	} else if provider == "ovh" {
		region = "DE1"
	} else if provider == "gce" {
		return fmt.Errorf("--region is required for the GCE provider")
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
	if provider == "scaleway" || provider == "ec2" || provider == "ovh" {

		var secretKeyErr error
		secretKey, secretKeyErr = getFileOrString(cmd.Flags(), "secret-key-file", "secret-key", true)
		if secretKeyErr != nil {
			return secretKeyErr
		}

		if provider == "ovh" {
			projectID, _ = cmd.Flags().GetString("project-id")
			if len(projectID) == 0 {
				return fmt.Errorf("--project-id flag must be set")
			}
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

	} else if provider == "gce" {
		projectID, _ = cmd.Flags().GetString("project-id")
		if len(projectID) == 0 {
			return fmt.Errorf("--project-id flag must be set")
		}
	}

	var subscriptionID string
	if provider == "azure" {
		subscriptionID, _ = cmd.Flags().GetString("subscription-id")
	}

	var endpoint string
	var consumerKey string
	if provider == "ovh" {
		endpoint, err = cmd.Flags().GetString("endpoint")
		if err != nil {
			return errors.Wrap(err, "failed to get 'endpoint' value")
		}
		consumerKey, err = cmd.Flags().GetString("consumer-key")
		if err != nil {
			return errors.Wrap(err, "failed to get 'endpoint' value")
		}
	}

	provisioner, err := getProvisioner(provider, accessToken, secretKey, organisationID, region, subscriptionID, sessionToken, endpoint, consumerKey, projectID)
	if err != nil {
		return err
	}

	letsencryptDomains, _ := cmd.Flags().GetStringArray("letsencrypt-domain")
	letsencryptEmail, _ := cmd.Flags().GetString("letsencrypt-email")
	letsencryptIssuer, _ := cmd.Flags().GetString("letsencrypt-issuer")

	if len(letsencryptDomains) == 0 && !tcp {
		return fmt.Errorf("either --letsencrypt-domain (for a HTTPS tunnel) or --tcp (for a TCP tunnel) must be set")
	}

	if len(letsencryptDomains) > 0 {
		if len(letsencryptEmail) == 0 {
			return fmt.Errorf("--letsencrypt-email is required when --letsencrypt-domain is given")
		}
		if len(letsencryptIssuer) == 0 {
			return fmt.Errorf("--letsencrypt-issuer is required when --letsencrypt-domain is given")
		}
		tcp = false
	}

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
		fmt.Sprintf("%d", inletsProControlPort),
		vpcID,
		subnetID,
		awsKeyName,
		tcp,
		letsencryptDomains,
	)
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
		fmt.Printf("Provisioning exit-server: %s in %s [%s]\n", name, zone, provider)
	} else {
		fmt.Printf("Provisioning exit-server: %s in %s [%s]\n", name, region, provider)
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
				fmt.Printf(`inlets HTTPS (%s) server summary:
  IP: %s
  HTTPS Domains: %v
  Auth-token: %s

Command:

inlets-pro http client --url "wss://%s:%d" \
  --token "%s" \
  --upstream http://127.0.0.1:8080

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
				fmt.Printf(`inlets TCP (%s) server summary:
  IP: %s
  Auth-token: %s

Command:

inlets-pro tcp client --url "wss://%s:%d" \
  --token "%s" \
  --upstream 127.0.0.1 \
  --ports 2222

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

func getProvisioner(provider, accessToken, secretKey, organisationID, region, subscriptionID, sessionToken, endpoint, consumerKey, projectID string) (provision.Provisioner, error) {

	switch provider {
	case "digitalocean":
		return provision.NewDigitalOceanProvisioner(accessToken)
	case "scaleway":
		return provision.NewScalewayProvisioner(accessToken, secretKey, organisationID, region)
	case "gce":
		return provision.NewGCEProvisioner(accessToken)
	case "ec2":
		return provision.NewEC2Provisioner(region, accessToken, secretKey, sessionToken)
	case "azure":
		return provision.NewAzureProvisioner(subscriptionID, accessToken)
	case "linode":
		return provision.NewLinodeProvisioner(accessToken)
	case "hetzner":
		return provision.NewHetznerProvisioner(accessToken)
	case "vultr":
		return provision.NewVultrProvisioner(accessToken)
	case "ovh":
		return provision.NewOVHProvisioner(endpoint, accessToken, secretKey, consumerKey, region, projectID)
	default:
		return nil, fmt.Errorf("no provisioner for provider: %s", provider)
	}

}

func generateAuth() (string, error) {
	pwdRes, pwdErr := password.Generate(64, 10, 0, false, true)
	return pwdRes, pwdErr
}

func createHost(provider, name, region, zone, projectID, userData, inletsProControlPort, vpcID, subnetID, awsKeyName string, tcp bool, letsencryptDomains []string) (*provision.BasicHost, error) {
	if provider == "digitalocean" {
		return &provision.BasicHost{
			Name:       name,
			OS:         "ubuntu-22-04-x64",
			Plan:       "s-1vcpu-1gb",
			Region:     region,
			UserData:   userData,
			Additional: map[string]string{},
		}, nil
	} else if provider == "ovh" {
		return &provision.BasicHost{
			Name:       name,
			OS:         "Ubuntu 22.04",
			Plan:       "s1-2",
			Region:     region,
			UserData:   userData,
			Additional: map[string]string{},
		}, nil
	} else if provider == "scaleway" {
		return &provision.BasicHost{
			Name:       name,
			OS:         "ubuntu-jammy",
			Plan:       "DEV1-S",
			Region:     region,
			UserData:   userData,
			Additional: map[string]string{},
		}, nil

	} else if provider == "gce" {
		return &provision.BasicHost{
			Name:     name,
			OS:       "projects/ubuntu-os-cloud/global/images/ubuntu-minimal-2204-jammy-v20240606",
			Plan:     "f1-micro",
			Region:   region,
			UserData: userData,
			Additional: map[string]string{
				"projectid":     projectID,
				"zone":          zone,
				"firewall-name": "inlets",
				"firewall-port": inletsProControlPort,
				"pro":           fmt.Sprint(tcp),
			},
		}, nil
	} else if provider == "ec2" {
		// Ubuntu images can be found here https://cloud-images.ubuntu.com/locator/ec2/
		// Name is used in the OS field so the ami can be lookup up in the region specified

		var additional = map[string]string{
			"inlets-port": inletsProControlPort,
			"pro":         fmt.Sprint(tcp),
		}

		if len(letsencryptDomains) > 0 {
			additional["ports"] = "80,443"
		}

		if len(awsKeyName) > 0 {
			additional["key-name"] = awsKeyName
		}

		if len(vpcID) > 0 {
			additional["vpc-id"] = vpcID
		}

		if len(subnetID) > 0 {
			additional["subnet-id"] = subnetID
		}

		return &provision.BasicHost{
			Name:       name,
			OS:         "ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-20230516",
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
				"inlets-port":    inletsProControlPort,
				"pro":            fmt.Sprint(tcp),
				"imagePublisher": "Canonical",
				"imageOffer":     "0001-com-ubuntu-server-jammy",
				"imageSku":       "22_04-lts-gen2",
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
		const ubuntu22_04_x64 = "1743"
		return &provision.BasicHost{
			Name:       name,
			OS:         ubuntu22_04_x64,
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
			OS:       "linode/ubuntu22.04",
			Plan:     "g6-nanode-1",
			Region:   region,
			UserData: userData,
			Additional: map[string]string{
				"inlets-port": inletsProControlPort,
				"pro":         fmt.Sprint(tcp),
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
			Plan:     "cx22",
			OS:       "ubuntu-22.04",
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
