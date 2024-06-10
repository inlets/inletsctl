# inletsctl - create inlets servers on the top cloud platforms

[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Documentation](https://godoc.org/github.com/inlets/inletsctl?status.svg)](http://godoc.org/github.com/inlets/inletsctl)
![Downloads](https://img.shields.io/github/downloads/inlets/inletsctl/total) <a href="https://actuated.dev/"><img alt="Arm CI sponsored by Actuated" src="https://docs.actuated.dev/images/actuated-badge.png" width="120px"></img></a>

inletsctl automates the task of creating an exit-server (tunnel server) on public cloud infrastructure.
The `create` command provisions a cheap cloud VM with a public IP and pre-installs inlets for you. You'll then get a connection string that you can use with the inlets client.

**Conceptual diagram**

![Webhook example with Inlets OSS](https://blog.alexellis.io/content/images/2019/09/inletsio--2-.png)

*Case-study with receiving webhooks from https://blog.alexellis.io/webhooks-are-great-when-you-can-get-them/*

Use-cases:

* Setup L4 TCP and HTTPS tunnels for your local services using [inlets-pro](https://inlets.dev/) with `inletsctl create`
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
  - [Contributing & getting help](#contributing--getting-help)
    - [Why do we need this tool?](#why-do-we-need-this-tool)
    - [Provisioners](#provisioners)
    - [Community support](#community-support)
    - [License](#license)

## Video demo

In the demo we:

* Create a cloud host on DigitalOcean with a single command
* Run a local Python HTTP server
* Connect our `inlets-pro client`
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

You can download the latest release from the [releases page](https://github.com/inlets/inletsctl/releases) or use [arkade](https://arkade.dev/):

```bash
# Install directly to /usr/local/bin/
curl -sLSf https://get.arkade.dev | sudo sh

# Install to local directory (for Windows users)
curl -sLSf https://get.arkade.dev | sh
```

The command can install inletsctl initially, and also update it later on:

```bash
arkade get inletsctl
```

Windows users are encouraged to use [git bash](https://git-scm.com/downloads) to install the inletsctl binary.

## Looking for documentation?

To learn about the various features of inletsctl and how to configure each cloud provisioner, head over to the docs:

* [docs.inlets.dev](https://docs.inlets.dev/#/tools/inletsctl?id=inletsctl-reference-documentation) 

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

### License

inletsctl is distributed under the MIT license. inlets-pro, which inletsctl uses is licensed under the [inlets-pro End User License Agreement (EULA)](https://github.com/inlets/inlets-pro/blob/master/EULA.md) and requires [a personal or business subscription](https://store.openfaas.com/).
