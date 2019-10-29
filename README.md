# inletsctl

[![Build Status](https://travis-ci.org/inlets/inletsctl.svg?branch=master)](https://travis-ci.org/inlets/inletsctl)

Provision cloud hosts as exit-nodes for [inlets](https://inlets.dev/) with a single command.

## Status

Completed:

* [x] DigitalOcean support
* [x] Scaleway support

Pending:

* [ ] Packet.com support
* [ ] `inletsctl delete` command

## Related projects

* [inlets](https://github.com/inlets/inlets) - open-source L7 HTTP tunnel and reverse proxy
* [inlets-pro](https://github.com/inlets/inlets-pro-pkg) - commercial L4 TCP load-balancer
* [inlets-operator](https://github.com/inlets/inlets-operator) - deep integration for inlets in Kubernetes, expose Service type LoadBalancer
* inletsctl - provision exit-nodes for use with inlets or inlets-pro

## Getting `inletsctl`

```sh
curl -sLSf https://raw.githubusercontent.com/inlets/inletsctl/master/get.sh | sudo sh
```

## Examples

Examples on how to run `inletsctl` to create an exit node.

Pre-reqs:

* You will need [inlets](https://inlets.dev/) on your client

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
