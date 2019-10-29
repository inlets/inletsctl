// Copyright (c) Inlets Author(s) 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package cmd

import (
	"fmt"
	"os"

	"github.com/morikuni/aec"
	"github.com/spf13/cobra"
)

var (
	Version   string
	GitCommit string
)

const WelcomeMessage = "Welcome to inletsctl! Find out more at https://github.com/inlets/inletsctl"

func init() {
	inletsCmd.AddCommand(versionCmd)
}

// inletsCmd represents the base command when called without any sub commands.
var inletsCmd = &cobra.Command{
	Use:   "inletsctl",
	Short: "Create exit nodes for use with inlets.",
	Long: `
inletsctl can create exit nodes for you on your preferred cloud provider
so that you can run a single command and then connect with your inlets
client.`,
	Run: runInlets,
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

	fmt.Println("Version:", getVersion())
	fmt.Println("Git Commit:", GitCommit)
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
