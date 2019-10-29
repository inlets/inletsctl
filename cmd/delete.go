// Copyright (c) Inlets Author(s) 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package cmd

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	inletsCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringP("provider", "p", "digitalocean", "The cloud provider")

	deleteCmd.Flags().StringP("inlets-token", "t", "", "The inlets auth token for your exit node")
	deleteCmd.Flags().StringP("access-token", "a", "", "The access token for your cloud")
	deleteCmd.Flags().StringP("access-token-file", "f", "", "Read this file for the access token for your cloud")

	deleteCmd.Flags().StringP("id", "i", "", "Host ID")

	deleteCmd.Flags().String("secret-key", "", "The access token for your cloud (Scaleway)")
	deleteCmd.Flags().String("secret-key-file", "", "Read this file for the access token for your cloud (Scaleway)")
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

	accessToken, err := getFileOrString(cmd.Flags(), "access-token-file", "access-token", true)
	if err != nil {
		return err
	}

	var secretKey string
	var organisationID string
	if provider == "scaleway" {
		var secretKeyErr error
		secretKey, secretKeyErr = getFileOrString(cmd.Flags(), "secret-key-file", "secret-key", true)
		if secretKeyErr != nil {
			return secretKeyErr
		}

		organisationID, _ = cmd.Flags().GetString("organisation-id")
		if len(organisationID) == 0 {
			return fmt.Errorf("--organisation-id cannot be empty")
		}
	}

	provisioner, err := getProvisioner(provider, accessToken, secretKey, organisationID)

	if err != nil {
		return err
	}

	hostID, _ := cmd.Flags().GetString("id")

	if len(hostID) == 0 {
		return fmt.Errorf("give a valid --id for your host")
	}

	fmt.Printf("Deleting host: %s from %s\n", hostID, provider)

	err = provisioner.Delete(hostID)
	if err != nil {
		return err
	}

	return err
}
