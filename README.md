# inletsctl

[![Build Status](https://travis-ci.com/inlets/inletsctl.svg?branch=master)](https://travis-ci.com/inlets/inletsctl)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/inlets/inletsctl)](https://goreportcard.com/report/github.com/inlets/inletsctl)
[![Documentation](https://godoc.org/github.com/inlets/inletsctl?status.svg)](http://godoc.org/github.com/inlets/inletsctl)
![Downloads](https://img.shields.io/github/downloads/inlets/inletsctl/total)

inletsctl automates the task of creating an exit-node on cloud infrastructure.
Once provisioned, you'll receive a command to connect with. You can use this 
tool whether you want to use inlets or inlets-pro for L4 TCP.

It needs to exist as a separate binary and CLI, so that the core inlets tool does not become bloated. The EC2 and AWS SDKs for Golang are very heavy-weight and result in a binary of over 30MB vs the small and nimble inlets and inlets-pro binaries.

**Conceptual diagram**

![Webhook example with Inlets OSS](https://blog.alexellis.io/content/images/2019/09/inletsio--2-.png)

*Case-study with receiving webhooks from https://blog.alexellis.io/webhooks-are-great-when-you-can-get-them/*

Use-cases:

* Setup L7 HTTP and L4 TCP tunnels for your local services using [inlets](https://inlets.dev/) with `inletsctl create`
* Port-forward services your local Kubernetes cluster using `inletsctl kfwd`

## Video demo

[![asciicast](https://asciinema.org/a/wVapSMsxpTdU9SBpRXwULaKE4.svg)](https://asciinema.org/a/wVapSMsxpTdU9SBpRXwULaKE4)

In the demo we:

* Create a cloud host on DigitalOcean with a single command
* Run a local Python HTTP server
* Connect our `inlets client`
* Access the Python HTTP server via the DigitalOcean Public IP
* Use the CLI to delete the host

inletsctl is the quickest and easiest way to automate both `inlets` and `inlets-pro`, whilst retaining complete control.

## Features/backlog

Completed:

* [x] Provisioner: DigitalOcean
* [x] Provisioner: Scaleway
* [x] Provisioner: Civo.com support
* [x] Provisioner: Google Cloud
* [x] Provisioner: Packet.com
* [x] Provisioner: AWS EC2
* [x] Provisioner: Azure
* [x] Provisioner: Linode
* [x] Provisioner: Hetzner
* [x] `inletsctl delete` command
* [x] Add poll interval `--poll 5s` for use with Civo that applies rate-limiting
* [x] Install `inlets/inlets-pro` via `inletsctl download` [#12](https://github.com/inlets/inletsctl/issues/12)

Pending:

* [ ] Enable `inletsctl delete` via `--ip` vs. instance ID [#2](https://github.com/inlets/inletsctl/issues/2)
* [ ] Enable `inlets-pro` and TCP with `inletsctl kfwd` [#13](https://github.com/inlets/inletsctl/issues/13)
* [ ] Generate systemd unit files for tunnels

### inlets projects

Inlets is a Cloud Native Tunnel and is [listed on the Cloud Native Landscape](https://landscape.cncf.io/category=service-proxy&format=card-mode&grouping=category&sort=stars) under *Service Proxies*.

* [inlets](https://github.com/inlets/inlets) - Cloud Native Tunnel for L7 / HTTP traffic written in Go
* [inlets-pro](https://github.com/inlets/inlets-pro-pkg) - Cloud Native Tunnel for L4 TCP
* [inlets-operator](https://github.com/inlets/inlets-operator) - Public IPs for your private Kubernetes Services and CRD
* [inletsctl](https://github.com/inlets/inletsctl) - Automate the cloud for fast HTTP (L7) and TCP (L4) tunnels

## How much will this cost?

The `inletsctl create` command will provision a cloud host with the provider and region of your choice and then start running `inlets server`. The host is configured with the standard VM image for Ubuntu or Debian Linux and inlets is installed via userdata/cloud-init.

The [provision](https://github.com/inlets/inletsctl/tree/master/pkg/provision) package contains defaults for OS images to use and for cloud host plans and sizing. You'll find all available options on `inletsctl create --help`

The cost for cloud hosts varies depending on a number of factors such as the region, bandwidth used, and so forth. A rough estimation is that it could cost around 5 USD / month to host a VM on for DigitalOcean, Civo, or Scaleway. The VM is required to provide your public IP. Some hosting providers supply credits and a free-tier such as GCE and AWS.

See the pricing grid on the [inlets-operator](https://github.com/inlets/inlets-operator#provider-pricing) for a detailed breakdown.

inletsctl does not automatically delete your exit nodes (read cloud hosts), so you'll need to do that in your dashboard or via `inletsctl delete` when you are done.

## Install `inletsctl`

```bash
# Install to local directory (and for Windows users)
curl -sLSf https://inletsctl.inlets.dev | sh

# Install directly to /usr/local/bin/
curl -sLSf https://inletsctl.inlets.dev | sudo sh
```

Windows users are encouraged to use [git bash](https://git-scm.com/downloads) to install the inletsctl binary.

## Examples for `inletsctl create`

Examples on how to run `inletsctl` to create an exit node.

Pre-reqs:

* You will need [inlets](https://inlets.dev/) on your client

Workflow:

* After running `inletsctl create`, the IP address of your exit-node will be returned along with a sample `inlets client` command, for instance:

  ```sh
  Inlets OSS exit-node summary:
    IP: 209.97.131.180
    Auth-token: qFyFzKYQvFSgtl7TM76p5SwWpmHaQGMT405HajiMzIYmwYVgJt1lvAMXfV4S3KlS

  Command:
    export UPSTREAM=http://127.0.0.1:8000
    inlets client --remote "ws://209.97.131.180:8080" \
          --token "qFyFzKYQvFSgtl7TM76p5SwWpmHaQGMT405HajiMzIYmwYVgJt1lvAMXfV4S3KlS" \
          --upstream $UPSTREAM
  ```

* You can delete your exit node using the `id` given by your cloud provider

  ```sh
  inletsctl delete --access-token-file ~/Downloads/do-access-token --id 164857028
  ```

### Example usage with Google Compute Engine

* One time setup required for a service account key

> It is assumed that you have gcloud installed and configured on your machine.
If not, then follow the instructions [here](https://cloud.google.com/sdk/docs/quickstarts)

```sh
# Get current projectID
export PROJECTID=$(gcloud config get-value core/project 2>/dev/null)

# Create a service account
gcloud iam service-accounts create inlets \
--description "inlets-operator service account" \
--display-name "inlets"

# Get service account email
export SERVICEACCOUNT=$(gcloud iam service-accounts list | grep inlets | awk '{print $2}')

# Assign appropriate roles to inlets service account
gcloud projects add-iam-policy-binding $PROJECTID \
--member serviceAccount:$SERVICEACCOUNT \
--role roles/compute.admin

gcloud projects add-iam-policy-binding $PROJECTID \
--member serviceAccount:$SERVICEACCOUNT \
--role roles/iam.serviceAccountUser

# Create inlets service account key file
gcloud iam service-accounts keys create key.json \
--iam-account $SERVICEACCOUNT
```

* Run inlets OSS or inlets-pro

```sh
# Create a tunnel with inlets OSS
inletsctl create -p gce --project-id=$PROJECTID -f=key.json

## Create a TCP tunnel with inlets-pro
inletsctl create -p gce -p $PROJECTID --remote-tcp=127.0.0.1 -f=key.json

# Or specify any valid Google Cloud Zone optional zone, by default it get provisioned in us-central1-a
inletsctl create -p gce --project-id=$PROJECTID -f key.json --zone=us-central1-a
```

### Example usage with DigitalOcean

```sh
inletsctl create --access-token-file $HOME/Downloads/do-access-token \
  --region="nyc1"
```

### Example with inlets-pro

Let's say we want to forward TCP connections to the IP `192.168.0.26` within our client's network, using inlets-pro, we'd run this using the `--remote-tcp` flag.

```sh
inletsctl create digitalocean --access-token-file ~/Downloads/do-access-token \
  --remote-tcp 192.168.0.26
```

### Example usage with Scaleway

```sh
# Obtain from your Scaleway dashboard:
export TOKEN=""
export SECRET_KEY=""
export ORG_ID=""

inletsctl create --provider scaleway \
  --access-token $TOKEN
  --secret-key $SECRET_KEY --organisation-id $ORG_ID
```

The region is hard-coded to France / Paris 1.

## Examples for `inletsctl kfwd`

The `inletsctl kfwd` command can port-forward services from within your local Kubernetes cluster to your local network or computer.

Example usage:

```sh
inletsctl kfwd --if 192.168.0.14 --from openfaas-figlet:8080
```

Then access the service via `http://127.0.0.1:8080`.

### Example usage with Azure

Prerequisites:

* You will need `az`. See [Install the Azure CLI](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli)


Generate Azure auth file 
```sh
az ad sp create-for-rbac --sdk-auth > ~/Downloads/client_credentials.json
```

Create
```sh
inletsctl create --provider=azure --subscription-id=4d68ee0c-7079-48d2-b15c-f294f9b11a9e \
  --region=eastus --access-token-file=~/Downloads/client_credentials.json 
```

Delete
```sh
inletsctl delete --provider=azure --id inlets-clever-volhard8 \
  --subscription-id=4d68ee0c-7079-48d2-b15c-f294f9b11a9e \
  --region=eastus --access-token-file=~/Downloads/client_credentials.json
```

### Example usage with Linode

Prerequisites:

* Prepare a Linode API Access Token. See [Create Linode API Access token](https://www.linode.com/docs/platform/api/getting-started-with-the-linode-api/#get-an-access-token)  


Create
```sh
inletsctl create --provider=linode --access-token=<API Access Token> --region=us-east
```

Delete
```sh
inletsctl delete --provider=linode --access-token=<API Access Token> --id <instance id>
```


## Downloading inlets or inlets-pro

The `inletsctl download` command can be used to download the inlets or inltets-pro binaries from github

Example usage:

```sh
# Download the latest inlets binary
inletsctl download

#Download the latest inlets-pro binary
inletsctl download --pro

# Download a specific version of inlets/inlets-pro
inletsctl download --version 2.6.2
```

## Contributing & getting help

Before seeking support, make sure you have read the instructions correctly, and try to run through them a second or third time to see if you have missed anything.

Then, try the troubleshooting guide below.

## Troubleshooting

inletsctl provisions a host called an exit node or exit server using public cloud APIs. It then 
prints out a connection string.

Are you unable to connect your client to the exit server?

### inlets PRO

If using auto-tls (the default), check that port 8123 is accessible. It should be serving a file with a self-signed certificate, run the following:

```bash
export IP=192.168.0.1
curl -k https://$IP:8123/.well-known/ca.crt
```

If you see connection refused, log in to the host over SSH and check the service via systemctl:

```bash
sudo systemctl status inlets-pro

# Check its logs
sudo journalctl -u inlets-pro
```

You can also check the configuration in `/etc/default/inlets-pro`, to make sure that an IP address and token are configured.

### inlets OSS

Try to connect on port 8080, where the control-port is being served. Does it connect, or not?

Connect with ssh to the exit-server and check the logs of the inlets service:

```bash
sudo systemctl status inlets

# Check its logs
sudo journalctl -u inlets
```

You can also check the configuration in `/etc/default/inlets`, to make sure that an IP address and token are configured.

## Configuration using environment variables

You may want to set an environment variable that points at your `access-token-file` or `secret-key-file`

Inlets will look for the following:

```sh
# For providers that use --access-token-file
INLETS_ACCESS_TOKEN

# For providers that use --secret-key-file
INLETS_SECRET_KEY

```
With the correct one of these set you wont need to add the flag on every command execution. 

You can set the following syntax in your `bashrc` (or equivalent for your shell)

```sh
export INLETS_ACCESS_TOKEN=$(cat my-token.txt)

# or set the INLETS_SECRET_KEY for those providors that use this
export INLETS_SECRET_KEY=$(cat my-token.txt)
```

### Community support

You can seek out community support through the [OpenFaaS Slack](https://slack.openfaas.io/) in the `#inlets` channel

### Add another cloud provisioner

Add a provisioner by sending a PR to the [inlets-operator's provision package](https://github.com/inlets/inlets-operator/tree/master/pkg/provision), once released, you can vendor the package here and add any flags that are required.

> Note: only providers and platforms which support cloud-init's user-data scripts are supported.

### License

MIT
