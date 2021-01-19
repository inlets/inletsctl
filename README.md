# inletsctl - the fastest way to create self-hosted exit-servers

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

* Setup L4 TCP tunnels for your local services using [inlets](https://inlets.dev/) with `inletsctl create`
* Port-forward services your local Kubernetes cluster using `inletsctl kfwd`

## Video demo

[![asciicast](https://asciinema.org/a/q8vqJ0Fwug47T62biscp7cJ5O.svg)](https://asciinema.org/a/q8vqJ0Fwug47T62biscp7cJ5O)

In the demo we:

* Create a cloud host on DigitalOcean with a single command
* Run a local Python HTTP server
* Connect our `inlets client`
* Access the Python HTTP server via the DigitalOcean Public IP
* Use the CLI to delete the host

inletsctl is the quickest and easiest way to automate `inlets-pro`, whilst retaining complete control of your tunnel and data.

## Provisioners

inletsctl can provision exit-servers to the following providers: DigitalOcean, Scaleway, Civo.com, Google Cloud, Equinix Metal, AWS EC2, Azure, Linode, Hetzner and Vultr.

An open-source Go package named [provision](https://github.com/inlets/cloud-provision) can be extended for each new provider. This code can be used outside of inletsctl by other projects wishing to create hosts and to run some scripts upon start-up via userdata.

```go
type Provisioner interface {
	Provision(BasicHost) (*ProvisionedHost, error)
	Status(id string) (*ProvisionedHost, error)
	Delete(HostDeleteRequest) error
}
```

## Features

* Provision hosts quickly using cloud-init with inlets pre-installed - `inletsctl create`
* Delete hosts by ID or IP address - `inletsctl delete`
* Automate port-forwarding from Kubernetes clusters with `inletsctl kfwd`

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

## Looking for documentation?

To learn about the various features of inletsctl and how to configure each cloud provisioner, head over to the docs:

* [docs.inlets.dev](https://docs.inlets.dev/) 

## Quick-start - create an exit server (inlets PRO)

This example is similar to the previous one, but also adds link-level encryption between your local service and the exit-server.

In addition, you can also expose pure TCP traffic such as SSH or Postgres.

```sh
inletsctl create \
  --provider digitalocean \
  --access-token-file $HOME/do-access-token \
  --pro
```

Note the output:

```bash
inlets PRO (0.7.0) exit-server summary:
  IP: 142.93.34.79
  Auth-token: TUSQ3Dkr9QR1VdHM7go9cnTUouoJ7HVSdiLq49JVzY5MALaJUnlhSa8kimlLwBWb

Command:
  export LICENSE=""
  export PORTS="8000"
  export UPSTREAM="localhost"

  inlets-pro client --url "wss://142.93.34.79:8123/connect" \
        --token "TUSQ3Dkr9QR1VdHM7go9cnTUouoJ7HVSdiLq49JVzY5MALaJUnlhSa8kimlLwBWb" \
        --license "$LICENSE" \
        --upstream $UPSTREAM \
        --ports $PORTS

To Delete:
          inletsctl delete --provider digitalocean --id "205463570"
```

Run a local service that uses TCP such as MariaDB:

```bash
head -c 16 /dev/urandom |shasum 
8cb3efe58df984d3ab89bcf4566b31b49b2b79b9

export PASSWORD="8cb3efe58df984d3ab89bcf4566b31b49b2b79b9"

docker run --name mariadb \
-p 3306:3306 \
-e MYSQL_ROOT_PASSWORD=8cb3efe58df984d3ab89bcf4566b31b49b2b79b9 \
-ti mariadb:latest
```

Connect to the tunnel updating the ports to `3306`

```bash
export LICENSE="$(cat ~/LICENSE)"
export PORTS="3306"
export UPSTREAM="localhost"

inlets-pro client --url "wss://142.93.34.79:8123/connect" \
      --token "TUSQ3Dkr9QR1VdHM7go9cnTUouoJ7HVSdiLq49JVzY5MALaJUnlhSa8kimlLwBWb" \
      --license "$LICENSE" \
      --upstream $UPSTREAM \
      --ports $PORTS
```

Now connect to your MariaDB instance from its public IP address:

```bash
export PASSWORD="8cb3efe58df984d3ab89bcf4566b31b49b2b79b9"
export EXIT_IP="142.93.34.79"

docker run -it --rm mariadb:latest mysql -h $EXIT_IP -P 3306 -uroot -p$PASSWORD

Welcome to the MariaDB monitor.  Commands end with ; or \g.
Your MariaDB connection id is 3
Server version: 10.5.5-MariaDB-1:10.5.5+maria~focal mariadb.org binary distribution

Copyright (c) 2000, 2018, Oracle, MariaDB Corporation Ab and others.

Type 'help;' or '\h' for help. Type '\c' to clear the current input statement.

MariaDB [(none)]> create database test; 
Query OK, 1 row affected (0.039 sec)
```


## Quick-start - create an exit server

This example uses DigitalOcean to create a cloud VM and then exposes a local service via the newly created exit-server.

```bash
inletsctl create \
  --provider digitalocean \
  --region="lon1" \
  --access-token-file $HOME/do-access-token
```

You'll see the host being provisioned, it usually takes just a few seconds:

```
Using provider: digitalocean
Requesting host: gifted-mestorf9 in lon1, from digitalocean
2020/08/26 10:58:36 Provisioning host with DigitalOcean
Host: 205463148, status: 
[1/500] Host: 205463148, status: new
...
[16/500] Host: 205463148, status: active
inlets OSS exit-server summary:
  IP: 165.232.108.137
  Auth-token: TlwzS2ze3hQEZTU3lvOk1dgQeHYQtyTX8ELlCYhdjis4FAMw1EDqlJfqr9w0XW5S

Command:
  export UPSTREAM=http://127.0.0.1:8000
  inlets client --remote "ws://165.232.108.137:8080" \
        --token "TlwzS2ze3hQEZTU3lvOk1dgQeHYQtyTX8ELlCYhdjis4FAMw1EDqlJfqr9w0XW5S" \
        --upstream $UPSTREAM

To Delete:
        inletsctl delete --provider digitalocean --id "205463148"
```

Now run the command given to you, changing the `--upstream` URL to match a local URL such as `http://127.0.0.1:3000`

```bash
  export UPSTREAM=http://127.0.0.1:3000
  inlets client --remote "ws://165.232.108.137:8080" \
        --token "TlwzS2ze3hQEZTU3lvOk1dgQeHYQtyTX8ELlCYhdjis4FAMw1EDqlJfqr9w0XW5S" \
        --upstream $UPSTREAM
```

You can then access your local website via the Internet and the exit-server's IP at:

http://165.232.108.137

When you're done, you can delete the host using its ID or IP address:

```bash
inletsctl delete --id 205463148
```

## Contributing & getting help

Before seeking support, make sure you have read the instructions correctly, and try to run through them a second or third time to see if you have missed anything.

Then, try the troubleshooting guide in the official docs (link above).

### Community support

You can seek out community support through the [OpenFaaS Slack](https://slack.openfaas.io/) in the `#inlets` channel

### License

MIT
