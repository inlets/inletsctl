// Copyright (c) Inlets Author(s) 2023. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

func subscriptionIDFromAzureAuth(subscriptionID, authContents string) string {
	if len(subscriptionID) > 0 {
		return subscriptionID
	}

	if len(authContents) == 0 {
		return ""
	}

	authMap := map[string]string{}
	if err := json.Unmarshal([]byte(authContents), &authMap); err != nil {
		return ""
	}

	return authMap["subscriptionId"]
}

func sampleDeleteCommand(provider, hostID, subscriptionID string) string {
	cmd := fmt.Sprintf("inletsctl delete --provider %s --id \"%s\"", provider, hostID)
	if provider == "azure" && len(subscriptionID) > 0 {
		cmd += fmt.Sprintf(" --subscription-id %q", subscriptionID)
	}
	return cmd
}

func getFileOrString(flags *pflag.FlagSet, file, value string, required bool) (string, error) {
	var val string
	fileVal, _ := flags.GetString(file)
	if len(fileVal) > 0 {
		res, err := os.ReadFile(fileVal)
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
