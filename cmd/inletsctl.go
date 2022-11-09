// Copyright (c) Inlets Author(s) 2019. All rights reserved.
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
	Version string

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
inletsctl automates the task of creating an exit-node on cloud infrastructure.
Once provisioned, you'll receive a command to connect with. You can use this 
tool whether you want to use inlets or inlets-pro for L4 TCP.

See also: inlets-operator which does the same job, but for Kubernetes services.
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
