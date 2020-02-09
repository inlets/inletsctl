# inletsctl

[![Build Status](https://travis-ci.com/inlets/inletsctl.svg?branch=master)](https://travis-ci.com/inlets/inletsctl)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/inlets/inletsctl)](https://goreportcard.com/report/github.com/inlets/inletsctl)
[![Documentation](https://godoc.org/github.com/inlets/inletsctl?status.svg)](http://godoc.org/github.com/inlets/inletsctl) [![Derek App](https://alexellis.o6s.io/badge?owner=inlets&repo=inletsctl)](https://github.com/alexellis/derek/)

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
* [x] `inletsctl delete` command
* [x] Add poll interval `--poll 5s` for use with Civo that applies rate-limiting
* [x] Install `inlets/inlets-pro` via `inletsctl download` [#12](https://github.com/inlets/inletsctl/issues/12)

Pending:

* [ ] Enable `inletsctl delete` via `--ip` vs. instance ID [#2](https://github.com/inlets/inletsctl/issues/2)
* [ ] Enable `inlets-pro` and TCP with `inletsctl kfwd` [#13](https://github.com/inlets/inletsctl/issues/13)
* [ ] Generate systemd unit files for tunnels

### Related projects

Inlets is [listed on the Cloud Native Landscape](https://landscape.cncf.io/category=service-proxy&format=card-mode&grouping=category&sort=stars) as a Service Proxy

* [inlets](https://github.com/inlets/inlets) - open-source L7 HTTP tunnel and reverse proxy
* [inlets-pro](https://github.com/inlets/inlets-pro) - L4 TCP load-balancer
* [inlets-operator](https://github.com/inlets/inlets-operator) - deep integration for inlets in Kubernetes, expose Service type LoadBalancer
* [inletsctl](https://github.com/inlets/inletsctl) - CLI tool to provision exit-nodes for use with inlets or inlets-pro

## How much will this cost?

The `inletsctl create` command will provision a cloud host with the provider and region of your choice and then start running `inlets server`. The host is configured with the standard VM image for Ubuntu or Debian Linux and inlets is installed via userdata/cloud-init.

The [provision](https://github.com/inlets/inletsctl/tree/master/pkg/provision) package contains defaults for OS images to use and for cloud host plans and sizing. You'll find all available options on `inletsctl create --help`

The cost for cloud hosts varies depending on a number of factors such as the region, bandwidth used, and so forth. A rough estimation is that it could cost around 5 USD / month to host a VM on for DigitalOcean, Civo, or Scaleway. The VM is required to provide your public IP. Some hosting providers supply credits and a free-tier such as GCE and AWS.

See the pricing grid on the [inlets-operator](https://github.com/inlets/inlets-operator#provider-pricing) for a detailed breakdown.

inletsctl does not automatically delete your exit nodes (read cloud hosts), so you'll need to do that in your dashboard or via `inletsctl delete` when you are done.

## Install `inletsctl`

```bash
# Install to local directory
curl -sLSf https://raw.githubusercontent.com/inlets/inletsctl/master/get.sh | sh

# Install to /usr/local/bin/
curl -sLSf https://raw.githubusercontent.com/inlets/inletsctl/master/get.sh | sudo sh
```

Or

```bash
# Install to local directory
curl -sLSf https://inletsctl.inlets.dev | sh

# Install to /usr/local/bin/
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

## Example for GCE

Follow the steps here to [configure your service account](https://github.com/inlets/inlets-operator#running-in-cluster-using-google-compute-engine-for-the-exit-node-using-helm)

## Examples for `inletsctl kfwd`

The `inletsctl kfwd` command can port-forward services from within your local Kubernetes cluster to your local network or computer.

Example usage:

```sh
inletsctl kfwd --if 192.168.0.14 --from openfaas-figlet:8080
```

Then access the service via `http://127.0.0.1:8080`.


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


## Contributing

### Add another cloud provisioner

Add a provisioner by sending a PR to the [inlets-operator's provision package](https://github.com/inlets/inlets-operator/tree/master/pkg/provision), once released, you can vendor the package here and add any flags that are required.

> Note: only providers and platforms which support cloud-init's user-data scripts are supported.

### License

MIT
