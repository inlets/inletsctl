// Copyright (c) Inlets Author(s) 2023. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package cmd

import (
	"fmt"

	"github.com/inlets/cloud-provision/provision"
	"github.com/inlets/inletsctl/pkg/env"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	inletsCmd.AddCommand(deleteCmd)
	deleteCmd.Flags().StringP("provider", "p", "digitalocean", "The cloud provider - digitalocean, gce, ec2, azure, scaleway, linode, hetzner or vultr")
	deleteCmd.Flags().StringP("region", "r", "lon1", "The region for your cloud provider")
	deleteCmd.Flags().StringP("zone", "z", "us-central1-a", "The zone for the exit node (gce)")

	deleteCmd.Flags().StringP("access-token", "a", "", "The access token for your cloud")
	deleteCmd.Flags().StringP("access-token-file", "f", "", "Read this file for the access token for your cloud")

	deleteCmd.Flags().StringP("id", "i", "", "Host ID")
	deleteCmd.Flags().String("ip", "", "Host IP")

	deleteCmd.Flags().String("secret-key", "", "The secret key for your cloud (scaleway, ec2)")
	deleteCmd.Flags().String("secret-key-file", "", "Read this file for the secret key for your cloud (scaleway, ec2)")
	deleteCmd.Flags().String("session-token", "", "The session token for ec2 (when using with temporary credentials)")
	deleteCmd.Flags().String("session-token-file", "", "Read this file for the session token for ec2 (when using with temporary credentials)")

	deleteCmd.Flags().String("organisation-id", "", "Organisation ID (scaleway)")
	deleteCmd.Flags().String("project-id", "", "Project ID (gce)")
	deleteCmd.Flags().String("subscription-id", "", "Subscription ID (azure)")

	deleteCmd.Flags().String("endpoint", "ovh-eu", "API endpoint (ovh), default: ovh-eu")
	deleteCmd.Flags().String("consumer-key", "", "The Consumer Key for using the OVH API")
}

// deleteCmd represents the client sub command
var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an exit node",
	Long: `Delete an exit node created at an earlier time by inletsctl using an API 
key for your cloud host.`,
	Example: `  inletsctl delete --provider digitalocean --id 1235678
	inletsctl delete --access-token-file $HOME/access-token --region lon1
`,
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

	var region string
	if cmd.Flags().Changed("region") {
		if regionVal, err := cmd.Flags().GetString("region"); isSet(regionVal) {
			if err != nil {
				return errors.Wrap(err, "failed to get 'region' value.")
			}
			region = regionVal
		}

	} else if provider == "scaleway" {
		region = "fr-par-1"
	} else if provider == "ec2" {
		region = "eu-west-1"
	}

	accessToken, err := env.GetRequiredFileOrString(cmd.Flags(),
		"access-token-file",
		"access-token",
		"INLETS_ACCESS_TOKEN",
	)
	if err != nil {
		return err
	}

	var secretKey string
	var sessionToken string
	var organisationID string
	if provider == "scaleway" || provider == "ec2" || provider == "ovh" {
		var secretKeyErr error
		secretKey, secretKeyErr = env.GetRequiredFileOrString(cmd.Flags(),
			"secret-key-file",
			"secret-key",
			"INLETS_SECRET_KEY",
		)
		if secretKeyErr != nil {
			return secretKeyErr
		}

		if provider == "ec2" {
			var sessionTokenErr error
			sessionToken, sessionTokenErr = getFileOrString(cmd.Flags(), "session-token-file", "session-token", false)
			if sessionTokenErr != nil {
				return sessionTokenErr
			}
		}

		if provider == "scaleway" {
			organisationID, _ = cmd.Flags().GetString("organisation-id")
			if len(organisationID) == 0 {
				return fmt.Errorf("--organisation-id cannot be empty")
			}
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

	projectID, _ := cmd.Flags().GetString("project-id")
	provisioner, err := getProvisioner(provider, accessToken, secretKey, organisationID, region, subscriptionID, sessionToken, endpoint, consumerKey, projectID)
	if err != nil {
		return err
	}

	hostID, _ := cmd.Flags().GetString("id")
	hostIP, _ := cmd.Flags().GetString("ip")
	zone, _ := cmd.Flags().GetString("zone")

	if isNotSet(hostID) && isNotSet(hostIP) {
		return fmt.Errorf("give a valid --id or --ip for your host")
	}

	if provider == "gce" && isSet(hostIP) {
		if isNotSet(projectID) {
			return fmt.Errorf("--ip requires --project-id to be set for provider")
		}
	}

	deleteRequest := provision.HostDeleteRequest{
		ID:        hostID,
		IP:        hostIP,
		ProjectID: projectID,
		Zone:      zone,
		Region:    region,
	}

	fmt.Printf("Deleting host: %s%s from %s\n", hostID, hostIP, provider)

	if err = provisioner.Delete(deleteRequest); err != nil {
		return err
	}

	return err
}

func isNotSet(s string) bool {
	return len(s) == 0
}

func isSet(s string) bool {
	return len(s) > 0
}
