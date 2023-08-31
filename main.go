// Copyright (c) Inlets Author(s) 2023. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package main

import (
	"fmt"
	"os"

	"github.com/inlets/inletsctl/cmd"
)

// These values will be injected into these variables at the build time.
var (
	Version   string
	GitCommit string
)

func main() {
	if err := cmd.Execute(Version, GitCommit); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
}
