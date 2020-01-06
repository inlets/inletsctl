// Copyright (c) Inlets Author(s) 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package cmd

import (
	"fmt"
	"github.com/inlets/inletsctl/pkg/provision"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	inletsCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringP("provider", "p", "digitalocean", "The cloud provider - digitalocean, gce, packet, scaleway, or civo")

	deleteCmd.Flags().StringP("inlets-token", "t", "", "The inlets auth token for your exit node")
	deleteCmd.Flags().StringP("access-token", "a", "", "The access token for your cloud")
	deleteCmd.Flags().StringP("access-token-file", "f", "", "Read this file for the access token for your cloud")

	deleteCmd.Flags().StringP("id", "i", "", "Host ID")

	deleteCmd.Flags().String("secret-key", "", "The access token for your cloud (Scaleway, EC2)")
	deleteCmd.Flags().String("secret-key-file", "", "Read this file for the access token for your cloud (Scaleway, EC2)")
	deleteCmd.Flags().String("organisation-id", "", "Organisation ID (Scaleway)")
}

// clientCmd represents the client sub command.
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an exit node",
	Long: `Delete an exit node

  Example: inletsctl delete --provider digitalocean --id abczsef`,
	RunE:          runDelete,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func runDelete(cmd *cobra.Command, _ []string) error {
	var pConfig provision.ProvisionerRequest
	var err error
	pConfig.Provider, err = cmd.Flags().GetString("provider")
	if err != nil {
		return errors.Wrap(err, "failed to get 'provider' value.")
	}

	fmt.Printf("Using provider: %s\n", pConfig.Provider)

	if cmd.Flags().Changed("region") {
		if regionVal, err := cmd.Flags().GetString("region"); len(regionVal) > 0 {
			if err != nil {
				return errors.Wrap(err, "failed to get 'region' value.")
			}
			pConfig.Region = regionVal
		}
	}

	inletsToken, err := cmd.Flags().GetString("inlets-token")
	if err != nil {
		return errors.Wrap(err, "failed to get 'inlets-token' value.")
	}
	if len(inletsToken) == 0 {
		var passwordErr error
		inletsToken, passwordErr = provision.GenerateAuth()

		if passwordErr != nil {
			return passwordErr
		}
	}

	pConfig.AccessToken, err = getFileOrString(cmd.Flags(), "access-token-file", "access-token", true)
	if err != nil {
		return err
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

	hostID, _ := cmd.Flags().GetString("id")

	if len(hostID) == 0 {
		return fmt.Errorf("give a valid --id for your host")
	}

	provisioner, err := provision.NewProvisioner(pConfig)
	if err != nil {
		return err
	}

	fmt.Printf("Deleting host: %s from %s\n", hostID, pConfig.Provider)

	err = provisioner.Delete(hostID)
	if err != nil {
		return err
	}

	return err
}
