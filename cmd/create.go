package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/inlets/inletsctl/pkg"

	password "github.com/sethvargo/go-password/password"

	provision "github.com/inlets/inlets-operator/pkg/provision"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	inletsCmd.AddCommand(createCmd)
	createCmd.Flags().StringP("provider", "p", "digitalocean", "The cloud provider")
	createCmd.Flags().StringP("region", "r", "lon1", "The region for your cloud provider")

	createCmd.Flags().StringP("inlets-token", "t", "", "The inlets auth token for your exit node")
	createCmd.Flags().StringP("access-token", "a", "", "The access token for your cloud")
	createCmd.Flags().StringP("access-token-file", "f", "", "Read this file for the access token for your cloud")
	createCmd.Flags().StringP("remote-tcp", "c", "", `Comma-separated TCP ports for inlets-pro i.e. "80,443"`)
}

// clientCmd represents the client sub command.
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an exit node",
	Long: `Create an exit node on your preferred cloud

  Example: inletsctl create --provider digitalocean`,
	RunE: runCreate,
}

func runCreate(cmd *cobra.Command, _ []string) error {

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

	var accessToken string
	accessTokenFile, _ := cmd.Flags().GetString("access-token-file")
	if len(accessTokenFile) > 0 {
		res, err := ioutil.ReadFile(accessTokenFile)
		if err != nil {
			return err
		}
		accessToken = strings.TrimSpace(string(res))
	} else {

		accessTokenVal, err := cmd.Flags().GetString("access-token")
		if err != nil {
			return errors.Wrap(err, "failed to get 'access-token' value.")
		}
		accessToken = accessTokenVal
	}

	if len(accessToken) == 0 {
		return fmt.Errorf("give a cloud provider API token via --access-token or --access-token-file")
	}

	provisioner, err := getProvisioner(provider, accessToken)

	if err != nil {
		return err
	}

	region, err := cmd.Flags().GetString("region")
	if err != nil {
		return errors.Wrap(err, "failed to get 'region' value.")
	}

	name := strings.Replace(pkg.GetRandomName(10), "_", "-", -1)
	inletsControlPort := 8080

	userData := makeUserdata(inletsToken, inletsControlPort, remoteTCP)

	hostReq, err := createHost(provider, name, region, userData)
	if err != nil {
		return err
	}

	fmt.Printf("Requesting host: %s in %s, from %s\n", name, region, provider)
	hostRes, err := provisioner.Provision(*hostReq)
	if err != nil {
		return err
	}

	fmt.Printf("Host: %s, status: %s\n", hostRes.ID, hostRes.Status)

	for i := 0; i < 500; i++ {
		time.Sleep(1 * time.Second)

		hostStatus, err := provisioner.Status(hostRes.ID)
		if err != nil {
			return err
		}

		fmt.Printf("Host: %s, status: %s\n", hostStatus.ID, hostStatus.Status)

		if hostStatus.Status == "active" {
			fmt.Printf(`Exit-node summary:
  IP: %s
  Auth-token: %s

Command:
  export UPSTREAM=http://127.0.0.1:8000
  inlets client --remote "ws://%s:%d" \
	--token "%s" \
	--upstream $UPSTREAM
`,
				hostStatus.IP, inletsToken, hostStatus.IP, inletsControlPort, inletsToken)
			return nil
		}
	}

	return err
}

func getProvisioner(provider, accessToken string) (provision.Provisioner, error) {
	if provider == "digitalocean" {
		return provision.NewDigitalOceanProvisioner(accessToken)
	}
	return nil, fmt.Errorf("no provisioner for provider: %s", provider)
}

func generateAuth() (string, error) {
	pwdRes, pwdErr := password.Generate(64, 10, 0, false, true)
	return pwdRes, pwdErr
}

func createHost(provider, name, region, userData string) (*provision.BasicHost, error) {
	if provider == "digitalocean" {
		return &provision.BasicHost{
			Name:       name,
			OS:         "ubuntu-16-04-x64",
			Plan:       "512mb",
			Region:     region,
			UserData:   userData,
			Additional: map[string]string{},
		}, nil
	}

	return nil, fmt.Errorf("no provisioner for provider: %s", provider)
}

func makeUserdata(authToken string, inletsControlPort int, remoteTCP string) string {

	controlPort := fmt.Sprintf("%d", inletsControlPort)

	if len(remoteTCP) == 0 {
		return `#!/bin/bash
export AUTHTOKEN="` + authToken + `"
export CONTROLPORT="` + controlPort + `"
curl -sLS https://get.inlets.dev | sudo sh

curl -sLO https://raw.githubusercontent.com/inlets/inlets/master/hack/inlets-operator.service  && \
	mv inlets-operator.service /etc/systemd/system/inlets.service && \
	echo "AUTHTOKEN=$AUTHTOKEN" > /etc/default/inlets && \
	echo "CONTROLPORT=$CONTROLPORT" >> /etc/default/inlets && \
	systemctl start inlets && \
	systemctl enable inlets`
	}

	return `#!/bin/bash
	export AUTHTOKEN="` + authToken + `"
	export REMOTETCP="` + remoteTCP + `"
	export IP=$(curl -sfSL https://ifconfig.co)
	
	curl -SLsf https://github.com/inlets/inlets-pro-pkg/releases/download/0.4.0/inlets-pro-linux > inlets-pro-linux && \
	chmod +x ./inlets-pro-linux  && \
	mv ./inlets-pro-linux /usr/local/bin/inlets-pro
	
	curl -sLO https://raw.githubusercontent.com/inlets/inlets/master/hack/inlets-pro.service  && \
		mv inlets-pro.service /etc/systemd/system/inlets-pro.service && \
		echo "AUTHTOKEN=$AUTHTOKEN" >> /etc/default/inlets-pro && \
		echo "REMOTETCP=$REMOTETCP" >> /etc/default/inlets-pro && \
		echo "IP=$IP" >> /etc/default/inlets-pro && \
		systemctl start inlets-pro && \
		systemctl enable inlets-pro`
}
