# inletsctl

[![Build Status](https://travis-ci.org/inlets/inletsctl.svg?branch=master)](https://travis-ci.org/inlets/inletsctl)

Provision exit-nodes for use with inlets

## Getting `inletsctl`

```sh
curl -sLSf https://raw.githubusercontent.com/inlets/inletsctl/master/get.sh | sudo sh
```

## Status

Completed:

* [x] DigitalOcean support
* [x] Scaleway support

Pending:

* [ ] Packet.com support
* [ ] `inletsctl delete` command

## Examples

Examples on how to run `inletsctl` to create an exit node

### Example usage with DigitalOcean

```sh
inletsctl create --access-token-file $HOME/Downloads/do-access-token \
  --region="nyc1"
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
