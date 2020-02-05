// Copyright (c) Inlets Author(s) 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/pflag"
)

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
