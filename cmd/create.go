// Copyright (c) Inlets Author(s) 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	names "github.com/inlets/inletsctl/pkg/names"
	provision "github.com/inlets/inletsctl/pkg/provision"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {
	inletsCmd.AddCommand(createCmd)
	createCmd.Flags().StringP("provider", "p", "digitalocean", "The cloud provider - digitalocean, gce, ec2, packet, scaleway, or civo")
	createCmd.Flags().StringP("region", "r", "", "The region for your cloud provider")
	createCmd.Flags().StringP("zone", "z", "", "The zone for the exit node (Google Compute Engine)")

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
	var pConfig provision.ProvisionerRequest
	var err error

	pConfig.Provider, err = cmd.Flags().GetString("provider")
	if err != nil {
		return errors.Wrap(err, "failed to get 'provider' value.")
	}

	fmt.Printf("Using provider: %s\n", pConfig.Provider)

	inletsToken, err := cmd.Flags().GetString("inlets-token")
	if err != nil {
		return errors.Wrap(err, "failed to get 'inlets-token' value.")
	}
	if len(inletsToken) == 0 {
		inletsToken, err = provision.GenerateAuth()

		if err != nil {
			return err
		}
	}

	var poll time.Duration
	pollOverride, pollOverrideErr := cmd.Flags().GetDuration("poll")
	if pollOverrideErr == nil {
		poll = pollOverride
	}

	pConfig.AccessToken, err = getFileOrString(cmd.Flags(), "access-token-file", "access-token", true)
	if err != nil {
		return err
	}

	if cmd.Flags().Changed("region") {
		if pConfig.Region, err = cmd.Flags().GetString("region"); len(pConfig.Region) == 0 {
			if err != nil {
				return errors.Wrap(err, "failed to get 'region' value.")
			}
			pConfig.Region = provision.Defaults[pConfig.Provider].Region
		}
	} else {
		pConfig.Region = provision.Defaults[pConfig.Provider].Region
	}

	var zone string
	if pConfig.Provider == provision.GCEProvider {
		if cmd.Flags().Changed("zone") {
			if zone, err = cmd.Flags().GetString("zone"); len(zone) == 0 {
				if err != nil {
					return errors.Wrap(err, "failed to get 'zone' value.")
				}
				zone = provision.Defaults[pConfig.Provider].Zone
			}
		} else {
			zone = provision.Defaults[pConfig.Provider].Zone
		}
	}

	if pConfig.Provider == provision.ScalewayProvider || pConfig.Provider == provision.EC2Provider {
		pConfig.SecretKey, err = getFileOrString(cmd.Flags(), "secret-key-file", "secret-key", true)
		if err != nil {
			return err
		}
		if pConfig.Provider == provision.ScalewayProvider {
			pConfig.OrganisationID, _ = cmd.Flags().GetString("organisation-id")
			if len(pConfig.OrganisationID) == 0 {
				return fmt.Errorf("--organisation-id cannot be empty")
			}
		}
	}

	provisioner, err := provision.NewProvisioner(pConfig)

	if err != nil {
		return err
	}

	remoteTCP, _ := cmd.Flags().GetString("remote-tcp")
	name := strings.Replace(names.GetRandomName(10), "_", "-", -1)

	userData := provision.MakeUserdata(provision.UserDataRequest{
		AuthToken:         inletsToken,
		InletsControlPort: provision.ControlPort,
		RemoteTCP:         remoteTCP,
	})

	projectID, _ := cmd.Flags().GetString("project-id")
	hostReq, err := provision.NewBasicHost(pConfig.Provider, name, pConfig.Region, projectID, zone, userData)
	if err != nil {
		return err
	}

	if pConfig.Provider == provision.GCEProvider {
		fmt.Printf("Requesting host: %s in %s, from %s\n", name, zone, pConfig.Provider)
	} else {
		fmt.Printf("Requesting host: %s in %s, from %s\n", name, pConfig.Region, pConfig.Provider)
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
  inlets client --remote "ws://%s:%s" \
	--token "%s" \
	--upstream $UPSTREAM

To Delete:
  inletsctl delete --provider %s --id "%s"
`,
					hostStatus.IP, inletsToken, hostStatus.IP, provision.ControlPort, inletsToken, pConfig.Provider, hostStatus.ID)
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
