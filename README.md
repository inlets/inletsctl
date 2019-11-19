# inletsctl

[![Build Status](https://travis-ci.org/inlets/inletsctl.svg?branch=master)](https://travis-ci.org/inlets/inletsctl)

Provision cloud hosts as exit-nodes for [inlets](https://inlets.dev/) with a single command.

## Status

Completed:

* [x] DigitalOcean support
* [x] Scaleway support
* [x] `inletsctl delete` command
* [x] Add Civo.com
* [x] Add poll interval `--poll 5s` for use with Civo that applies rate-limiting

Pending:

* [ ] Add CLI flags to support Packet.com

### Related projects

Inlets is [listed on the Cloud Native Landscape](https://landscape.cncf.io/category=service-proxy&format=card-mode&grouping=category&sort=stars) as a Service Proxy

* [inlets](https://github.com/inlets/inlets) - open-source L7 HTTP tunnel and reverse proxy
* [inlets-pro](https://github.com/inlets/inlets-pro-pkg) - L4 TCP load-balancer
* [inlets-operator](https://github.com/inlets/inlets-operator) - deep integration for inlets in Kubernetes, expose Service type LoadBalancer
* [inletsctl](https://github.com/inlets/inletsctl) - CLI tool to provision exit-nodes for use with inlets or inlets-pro

## Getting `inletsctl`

```sh
curl -sLSf https://raw.githubusercontent.com/inlets/inletsctl/master/get.sh | sudo sh
```

Or

```sh
curl -sLSf https://inletsctl.inlets.dev | sudo sh
```

Windows users are encouraged to use [git bash](https://git-scm.com/downloads) to install the inletsctl binary.

## Costs for exit-nodes

See notes for [inlets-operator](https://github.com/inlets/inlets-operator#provider-pricing)

## Examples

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

## Example with inlets-pro

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

## Contributing

### Add another cloud provisioner

Add a provisioner by sending a PR to the [inlets-operator's provision package](https://github.com/inlets/inlets-operator/tree/master/pkg/provision), once released, you can vendor the package here and add any flags that are required.

> Note: only clouds that support cloud-init can be added

### License

MIT
