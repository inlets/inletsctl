// Copyright (c) Inlets Author(s) 2019. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for full license information.

package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	v1 "github.com/alexellis/go-execute/pkg/v1"
	"github.com/spf13/cobra"
)

func init() {
	inletsCmd.AddCommand(kfwdCmd)

	kfwdCmd.Flags().StringP("from", "f", "", "From service for the inlets client to forward")
	kfwdCmd.Flags().StringP("if", "i", "", "The address of your laptop, for the inlets PRO server to connect to")
	kfwdCmd.Flags().StringP("namespace", "n", "default", "Source service namespace")
	kfwdCmd.Flags().String("license", "", "Inlets PRO license key")
	kfwdCmd.Flags().Bool("tcp", false, "Use inlets PRO in TCP mode, or if set to false, in HTTP mode")
}

// clientCmd represents the client sub command.
var kfwdCmd = &cobra.Command{
	Use:   "kfwd",
	Short: "Forward a Kubernetes service to the local machine",
	Long: `Forward a Kubernetes service to the local machine using the --if flag to 
specify an ethernet address accessible from within the Kubernetes cluster

An inlets PRO HTTP or TCP client is run within the cluster as a deployment, 
and then the server process is run by inletsctl. The cluster starts the 
client which then establishes the connection to the server on your 
local machine.`,
	Example: `  # Forward a HTTP service
  inletsctl kfwd --from test-app-expressjs-k8s:8080 --if 192.168.0.14

  # Forward a TCP service
  inletsctl kfwd --from nats:4222 --tcp
`,
	RunE:          runKfwd,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func fwdTCP(cmd *cobra.Command, eth, port, upstream, ns, inletsToken, license string) error {

	deployment := makeTCPDeployment(eth,
		port,
		upstream,
		ns,
		inletsToken,
		license)

	tmpPath := path.Join(os.TempDir(), "inlets-"+upstream+".yaml")
	err := ioutil.WriteFile(tmpPath, []byte(deployment), 0600)
	if err != nil {
		return err
	}

	fmt.Printf("%s written.\n", tmpPath)

	task := v1.ExecTask{
		Command: "kubectl",
		Args:    []string{"apply", "-f", tmpPath},
	}

	res, err := task.Execute()
	if err != nil {
		return err
	}

	if res.ExitCode != 0 {
		return fmt.Errorf("exit code unexpected: %d, stderr: %s", res.ExitCode, res.Stderr)
	}

	fmt.Println("inlets PRO client scheduled inside your cluster.")

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM)
		signal.Notify(sig, syscall.SIGINT)

		<-sig

		log.Printf("Interrupt received..\n")

		task := v1.ExecTask{
			Command: "kubectl",
			Args:    []string{"delete", "-f", tmpPath},
		}
		res, err := task.Execute()

		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			return
		}

		if res.ExitCode != 0 {
			fmt.Fprintf(os.Stderr, fmt.Errorf("exit code unexpected from kubectl delete: %d, stderr: %s", res.ExitCode, res.Stderr).Error())
			return
		}
	}()

	fmt.Printf(`inlets PRO server now listening.

%s:%s

Hit Control+C to cancel.
`, eth, port)

	serverTask := v1.ExecTask{
		Command: "inlets-pro",
		Args: []string{
			"server",
			"--token=" + inletsToken,
			"--auto-tls-san=" + eth,
			"--auto-tls=true",
		},
	}

	serverRes, serverErr := serverTask.Execute()

	if serverErr != nil {
		return fmt.Errorf("error with server: %s", serverErr.Error())
	}

	if serverRes.ExitCode != 0 {
		return fmt.Errorf("exit code unexpected from inlets server: %d, stderr: %s, stdout: %s", serverRes.ExitCode, serverRes.Stderr, serverRes.Stdout)

	}

	return nil
}

func runKfwd(cmd *cobra.Command, _ []string) error {

	ns, err := cmd.Flags().GetString("namespace")
	if err != nil {
		return err
	}

	eth, err := cmd.Flags().GetString("if")
	if err != nil {
		return err
	}

	if len(eth) == 0 {
		return fmt.Errorf("give --if with the IP of your local network from ifconfig or similar")
	}

	from, err := cmd.Flags().GetString("from")
	if err != nil {
		return err
	}

	if len(from) == 0 {
		return fmt.Errorf("give a --from service")
	}

	portSep := strings.Index(from, ":")
	if portSep < 0 {
		return fmt.Errorf("no port given in --from flag")
	}

	upstream := from[:portSep]
	port := from[portSep+1:]

	fmt.Println(upstream, "=", port)

	inletsToken, passwordErr := generateAuth()
	if passwordErr != nil {
		return passwordErr
	}
	license, err := cmd.Flags().GetString("license")
	if err != nil {
		return err
	}
	if len(license) == 0 {
		return fmt.Errorf("--license is required for use with inlets PRO, get a free trial at inlets.dev")
	}

	if tcp, _ := cmd.Flags().GetBool("tcp"); tcp {
		fmt.Println("Forwarding in TCP mode")
		return fwdTCP(cmd, eth, port, upstream, ns, inletsToken, license)
	}

	fmt.Println("Forwarding in HTTP mode")

	deployment := makeHTTPDeployment(eth, port, upstream, ns, inletsToken, license)
	tmpPath := path.Join(os.TempDir(), "inlets-"+upstream+".yaml")
	err = ioutil.WriteFile(tmpPath, []byte(deployment), 0600)
	if err != nil {
		return err
	}
	fmt.Printf("%s written.\n", tmpPath)

	task := v1.ExecTask{
		Command: "kubectl",
		Args:    []string{"apply", "-f", tmpPath},
	}
	res, err := task.Execute()
	if err != nil {
		return err
	}

	if res.ExitCode != 0 {
		return fmt.Errorf("exit code unexpected: %d, stderr: %s", res.ExitCode, res.Stderr)
	}

	fmt.Println("Inlets PRO HTTP client scheduled inside your cluster.")

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM)
		signal.Notify(sig, syscall.SIGINT)

		<-sig

		log.Printf("Interrupt received..\n")

		task := v1.ExecTask{
			Command: "kubectl",
			Args:    []string{"delete", "-f", tmpPath},
		}
		res, err := task.Execute()

		if err != nil {
			fmt.Fprintf(os.Stderr, err.Error())
			return
		}

		if res.ExitCode != 0 {
			fmt.Fprintf(os.Stderr, fmt.Errorf("exit code unexpected from kubectl delete: %d, stderr: %s", res.ExitCode, res.Stderr).Error())
			return
		}
	}()

	fmt.Printf(`Inlets PRO TCP server now listening.

http://%s:%s

Hit Control+C to cancel.
`, eth, port)

	serverTask := v1.ExecTask{
		Command: "inlets-pro",
		Args: []string{
			"http",
			"server",
			"--auto-tls=true",
			"--auto-tls-san=" + eth,
			"--port=" + port,
			"--token=" + inletsToken,
		},
	}

	serverRes, err := serverTask.Execute()
	if err != nil {
		return fmt.Errorf("error with server: %s", err.Error())
	}

	if serverRes.ExitCode != 0 {
		return fmt.Errorf("exit code unexpected from server: %d, stderr: %s, stdout: %s",
			serverRes.ExitCode,
			serverRes.Stderr,
			serverRes.Stdout)

	}

	return nil
}

func makeTCPDeployment(remote, ports, upstream, ns, inletsToken, license string) string {

	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: inlets-pro-client-%s
  namespace: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: inlets-pro-client
  template:
    metadata:
      labels:
        app.kubernetes.io/name: inlets-pro-client
    spec:
      containers:
      - name: inlets-pro-client
        image: ghcr.io/inlets/inlets-pro:0.9.1
        imagePullPolicy: IfNotPresent
        command: ["inlets-pro"]
        args:
        - "tcp"
        - "client"
        - "--auto-tls=true"
        - "--url=wss://%s:8123"
        - "--upstream=%s"
        - "--ports=%s"
        - "--token=%s"
        - "--license=%s"
`, upstream, ns, remote, upstream, ports, inletsToken, license)
}

func makeHTTPDeployment(remote, port, upstream, ns, inletsToken, license string) string {

	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: inlets-http-%s
  namespace: %s
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: inlets-http
  template:
    metadata:
      labels:
        app.kubernetes.io/name: inlets-http
    spec:
      containers:
      - name: inlets
        image: ghcr.io/inlets/inlets-pro:0.9.1
        imagePullPolicy: IfNotPresent
        command: ["inlets-pro"]
        args:
        - "http"
        - "client"
        - "--url=wss://%s:8123"
        - "--auto-tls=true"
        - "--upstream=http://%s:%s"
        - "--token=%s"
        - "--license=%s"
`, upstream, ns, remote, upstream, port, inletsToken, license)
}
