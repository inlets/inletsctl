package cmd

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/inlets/inletsctl/pkg"

	provision "github.com/inlets/inlets-operator/pkg/provision"
	"github.com/pkg/errors"
	password "github.com/sethvargo/go-password/password"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {
	inletsCmd.AddCommand(createCmd)
	createCmd.Flags().StringP("provider", "p", "digitalocean", "The cloud provider")
	createCmd.Flags().StringP("region", "r", "lon1", "The region for your cloud provider")

	createCmd.Flags().StringP("inlets-token", "t", "", "The inlets auth token for your exit node")
	createCmd.Flags().StringP("access-token", "a", "", "The access token for your cloud")
	createCmd.Flags().StringP("access-token-file", "f", "", "Read this file for the access token for your cloud")

	createCmd.Flags().String("secret-key", "", "The access token for your cloud (Scaleway)")
	createCmd.Flags().String("secret-key-file", "", "Read this file for the access token for your cloud (Scaleway)")
	createCmd.Flags().String("organisation-id", "", "Organisation ID (Scaleway)")

	createCmd.Flags().StringP("remote-tcp", "c", "", `Remote host for inlets-pro to use for forwarding TCP connections`)
}

// clientCmd represents the client sub command.
var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an exit node",
	Long: `Create an exit node on your preferred cloud

  Example: inletsctl create --provider digitalocean`,
	RunE:          runCreate,
	SilenceUsage:  true,
	SilenceErrors: true,
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

	region, err := cmd.Flags().GetString("region")
	if err != nil {
		return errors.Wrap(err, "failed to get 'region' value.")
	}

	remoteTCP, _ := cmd.Flags().GetString("remote-tcp")

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
			if len(remoteTCP) == 0 {
				fmt.Printf(`Inlets OSS exit-node summary:
  IP: %s
  Auth-token: %s

Command:
  export UPSTREAM=http://127.0.0.1:8000
  inlets client --remote "ws://%s:%d" \
	--token "%s" \
	--upstream $UPSTREAM
`,
					hostStatus.IP, inletsToken, hostStatus.IP, inletsControlPort, inletsToken)
				fmt.Printf(`Inlets OSS exit-node summary:
  IP: %s
  Auth-token: %s

Command:
  export UPSTREAM=http://127.0.0.1:8000
  inlets client --remote "ws://%s:%d" \
	--token "%s" \
	--upstream $UPSTREAM
`,
					hostStatus.IP, inletsToken, hostStatus.IP, inletsControlPort, inletsToken)
			} else {
				proPort := 8123
				fmt.Printf(`inlets-pro exit-node summary:
  IP: %s
  Auth-token: %s

Command:
  export TCP_PORTS="8000"
  export LICENSE=""
  inlets-pro client --connect "ws://%s:%d/connect" \
	--token "%s" \
	--license "$LICENSE" \
	--tcp-ports 8000
`,
					hostStatus.IP, inletsToken, hostStatus.IP, proPort, inletsToken)
			}

			return nil
		}
	}

	return err
}

func getProvisioner(provider, accessToken, secretKey, organisationID string) (provision.Provisioner, error) {
	if provider == "digitalocean" {
		return provision.NewDigitalOceanProvisioner(accessToken)
	} else if provider == "scaleway" {
		return pkg.NewScalewayProvisioner(accessToken, secretKey, organisationID)
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
	} else if provider == "scaleway" {
		return &provision.BasicHost{
			Name:       name,
			OS:         "ubuntu-bionic",
			Plan:       "DEV1-S",
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
curl -sLS https://get.inlets.dev | sh

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
