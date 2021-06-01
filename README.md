# inletsctl - the fastest way to create self-hosted tunnels

[![Build Status](https://travis-ci.com/inlets/inletsctl.svg?branch=master)](https://travis-ci.com/inlets/inletsctl)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Report Card](https://goreportcard.com/badge/github.com/inlets/inletsctl)](https://goreportcard.com/report/github.com/inlets/inletsctl)
[![Documentation](https://godoc.org/github.com/inlets/inletsctl?status.svg)](http://godoc.org/github.com/inlets/inletsctl)
![Downloads](https://img.shields.io/github/downloads/inlets/inletsctl/total)

inletsctl automates the task of creating an exit-server (tunnel server) on public cloud infrastructure.
The `create` command provisions a cheap cloud VM with a public IP and pre-installs inlets PRO for you. You'll then get a connection string that you can use with the inlets client.

**Conceptual diagram**

![Webhook example with Inlets OSS](https://blog.alexellis.io/content/images/2019/09/inletsio--2-.png)

*Case-study with receiving webhooks from https://blog.alexellis.io/webhooks-are-great-when-you-can-get-them/*

Use-cases:

* Setup L4 TCP and HTTPS tunnels for your local services using [inlets PRO](https://inlets.dev/) with `inletsctl create`
* Create tunnels for use with Kubernetes clusters, create the tunnel and use it whenever you need it
* Port-forward services your local Kubernetes cluster using `inletsctl kfwd`

## Contents

- [inletsctl - the fastest way to create self-hosted tunnels](#inletsctl---the-fastest-way-to-create-self-hosted-tunnels)
  - [Contents](#contents)
  - [Video demo](#video-demo)
  - [Features](#features)
  - [How much will this cost?](#how-much-will-this-cost)
  - [Install `inletsctl`](#install-inletsctl)
  - [Looking for documentation?](#looking-for-documentation)
  - [Create a HTTPS tunnel with a custom domain](#create-a-https-tunnel-with-a-custom-domain)
  - [Create a HTTP tunnel](#create-a-http-tunnel)
  - [Create a tunnel for a TCP service](#create-a-tunnel-for-a-tcp-service)
  - [Contributing & getting help](#contributing--getting-help)
    - [Why do we need this tool?](#why-do-we-need-this-tool)
    - [Provisioners](#provisioners)
    - [Community support](#community-support)
    - [License](#license)

## Video demo

In the demo we:

* Create a cloud host on DigitalOcean with a single command
* Run a local Python HTTP server
* Connect our `inlets client`
* Access the Python HTTP server via the DigitalOcean Public IP
* Use the CLI to delete the host

[![asciicast](https://asciinema.org/a/q8vqJ0Fwug47T62biscp7cJ5O.svg)](https://asciinema.org/a/q8vqJ0Fwug47T62biscp7cJ5O)


inletsctl is the quickest and easiest way to automate tunnels, whilst retaining complete control of your tunnel and data.

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

## Create a HTTPS tunnel with a custom domain

This example uses DigitalOcean to create a cloud VM and then exposes a local service via the newly created exit-server.

Let's say we want to expose a Grafana server on our internal network to the Internet via [Let's Encrypt](https://letsencrypt.org/) and HTTPS?

```bash
export DOMAIN="grafana.example.com"

inletsctl create \
  --provider digitalocean \
  --region="lon1" \
  --access-token-file $HOME/do-access-token \
  --letsencrypt-domain $DOMAIN \
  --letsencrypt-email webmaster@$DOMAIN \
  --letsencrypt-issuer prod
```

You can also use `--letsencrypt-issuer` with the `staging` value whilst testing since Let's Encrypt rate-limits how many certificates you can obtain within a week.

Create a DNS A record for the IP address so that `grafana.example.com` for instance resolves to that IP. For instance you could run:

```bash
doctl compute domain create \
  --ip-address 46.101.60.161 grafana.example.com
```

Now run the command that you were given, and if you wish, change the upstream to point to the domain explicitly:

```bash
# Obtain a license at https://inlets.dev
# Store it at $HOME/.inlets/LICENSE or use --help for more options

# Where to route traffic from the inlets server
export UPSTREAM="grafana.example.com=http://192.168.0.100:3000"

inlets-pro http client --url "wss://46.101.60.161:8123" \
--token "lRdRELPrkhA0kxwY0eWoaviWvOoYG0tj212d7Ff0zEVgpnAfh5WjygUVVcZ8xJRJ" \
--upstream $UPSTREAM

To delete:
  inletsctl delete --provider digitalocean --id "248562460"
```

You can also specify more than one domain and upstream for the same tunnel, so you could expose OpenFaaS and Prometheus separately for instance.

Update the `inletsctl create` command with multiple domains such as: `--letsencrypt-domain openfaas.example.com --letsencrypt-domain grafana.example.com`

Then for the `inlets-pro client` command, update the upstream in the same way: `--upstream openfaas.example.com=http://127.0.0.1:8080,grafana.example.com=http://192.168.0.100:3000`

## Create a HTTP tunnel

This example uses Linode to create a cloud VM and then exposes a local service via the newly created exit-server.

```bash
export REGION="eu-west"

inletsctl create \
  --provider linode \
  --region="$REGION" \
  --access-token-file $HOME/do-access-token
```

You'll see the host being provisioned, it usually takes just a few seconds:

```
Using provider: linode
Requesting host: peaceful-lewin8 in eu-west, from linode
2021/06/01 15:56:03 Provisioning host with Linode
Host: 248561704, status: 
[1/500] Host: 248561704, status: new
...
[11/500] Host: 248561704, status: active

inlets PRO (0.7.0) exit-server summary:
  IP: 188.166.168.90
  Auth-token: dZTkeCNYgrTPvFGLifyVYW6mlP78ny3jhyKM1apDL5XjmHMLYY6MsX8S2aUoj8uI
```

Now run the command given to you, changing the `--upstream` URL to match a local URL such as `http://127.0.0.1:3000`

```bash
# Obtain a license at https://inlets.dev
export LICENSE="$HOME/.inlets/license"

# Give a single value or comma-separated
export PORTS="8000"

# Where to route traffic from the inlets server
export UPSTREAM="localhost"

inlets-pro client --url "wss://188.166.168.90:8123/connect" \
  --token "dZTkeCNYgrTPvFGLifyVYW6mlP78ny3jhyKM1apDL5XjmHMLYY6MsX8S2aUoj8uI" \
  --license-file "$LICENSE" \
  --upstream $UPSTREAM \
  --ports $PORTS
```

You can then access your local website via the Internet and the exit-server's IP at:

http://165.232.108.137

When you're done, you can delete the host using its ID or IP address:

```bash
  inletsctl delete --provider linode --id "248561704"
  inletsctl delete --provider linode --ip "188.166.168.90"
```

## Create a tunnel for a TCP service

This example is similar to the previous one, but also adds link-level encryption between your local service and the exit-server.

In addition, you can also expose pure TCP traffic such as SSH or Postgresql.

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

## Contributing & getting help

Before seeking support, make sure you have read the instructions correctly, and try to run through them a second or third time to see if you have missed anything.

Then, try the troubleshooting guide in the official docs (link above).

### Why do we need this tool?

Why is inletsctl a separate binary? This tool is shipped separately, so that the core tunnel binary does not become bloated. The EC2 and AWS SDKs for Golang are very heavy-weight and result in a binary of over 30MB vs the small and nimble inlets-pro binaries.

### Provisioners

inletsctl can provision exit-servers to the following providers: DigitalOcean, Scaleway, Civo.com, Google Cloud, Equinix Metal, AWS EC2, Azure, Linode, Hetzner and Vultr.

An open-source Go package named [provision](https://github.com/inlets/cloud-provision) can be extended for each new provider. This code can be used outside of inletsctl by other projects wishing to create hosts and to run some scripts upon start-up via userdata.

```go
type Provisioner interface {
	Provision(BasicHost) (*ProvisionedHost, error)
	Status(id string) (*ProvisionedHost, error)
	Delete(HostDeleteRequest) error
}
```

### Community support

You can seek out community support through the [OpenFaaS Slack](https://slack.openfaas.io/) in the `#inlets` channel

### License

MIT
