// Copyright (c) Inlets Author(s) 2023. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package cmd

import (
	"fmt"
	"os"
	"runtime"

	"github.com/morikuni/aec"
	"github.com/spf13/cobra"
)

var (
	// Version as per git repo
	Version string = "dev"

	// GitCommit as per git repo
	GitCommit string
)

// WelcomeMessage to introduce inletsctl
const WelcomeMessage = "Welcome to inletsctl! Find out more at https://github.com/inlets/inletsctl"

func init() {
	inletsCmd.AddCommand(versionCmd)
	inletsCmd.AddCommand(makeUpdate())
}

// inletsCmd represents the base command when called without any sub commands.
var inletsCmd = &cobra.Command{
	Use:   "inletsctl",
	Short: "Create exit nodes for use with inlets.",
	Long: `
Use inletsctl to create a VM (aka exit-node) with the inlets-server
preinstalled on cloud infrastructure. Once provisioned, you'll receive a
connection string for the inlets-pro client.

For HTTPS tunnels (L7):
The tunnel server will terminate TLS for you, just include the 
--letsencrypt-domain flag for each domain you want to expose via the exit-node.

For TCP tunnels (L4):
Use the --tcp flag to create a TCP tunnel via inletsctl create. This is
best suited to SSH, TLS, reverse proxies, databases, etc.

See also: inlets-operator which automates L4 TCP tunnels for any
Kubernetes LoadBalancer services found in a cluster.
`,
	Run:           runInlets,
	SilenceErrors: true,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the clients version information.",
	Run:   parseBaseCommand,
}

func getVersion() string {
	if len(Version) != 0 {
		return Version
	}
	return "dev"
}

func parseBaseCommand(_ *cobra.Command, _ []string) {
	printLogo()

	fmt.Printf("Version: %s\n", getVersion())
	fmt.Printf("Git Commit: %s\n", GitCommit)
	fmt.Printf("Build target: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	os.Exit(0)
}

// Execute adds all child commands to the root command(InletsCmd) and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the InletsCmd.
func Execute(version, gitCommit string) error {

	// Get Version and GitCommit values from main.go.
	Version = version
	GitCommit = gitCommit

	if err := inletsCmd.Execute(); err != nil {
		return err
	}
	return nil
}

func runInlets(cmd *cobra.Command, args []string) {
	printLogo()
	cmd.Help()
}

func printLogo() {
	inletsLogo := aec.WhiteF.Apply(inletsFigletStr)
	fmt.Println(inletsLogo)
}

const inletsFigletStr = ` _       _      _            _   _ 
(_)_ __ | | ___| |_ ___  ___| |_| |
| | '_ \| |/ _ \ __/ __|/ __| __| |
| | | | | |  __/ |_\__ \ (__| |_| |
|_|_| |_|_|\___|\__|___/\___|\__|_|
`
