# inlets provision package

This package can be used to provision cloud hosts using a simple CRUD-style API:

[provision.go](https://github.com/inlets/inletsctl/blob/master/pkg/provision/provision.go)

```go
type Provisioner interface {
	Provision(BasicHost) (*ProvisionedHost, error)
	Status(id string) (*ProvisionedHost, error)
	Delete(HostDeleteRequest) error
}
```

## Usage of the package

This package is used by:

* [inletsctl](https://github.com/inlets/inletsctl) - Go CLI to create/delete exit-servers and inlets/-pro tunnels
* [inlets-operator](https://github.com/inlets/inlets-operator) - Kubernetes operator to automate exit-servers and inlets/-pro tunnels via CRDs and Service definitions

## Rules for adding a new provisioner

* Use the Ubuntu 16.04 LTS image
* Select the cheapest plan and update the [README](https://github.com/inlets/inletsctl/blob/master/README.md) with the estimated monthly cost
* For inlets OSS open just the required ports
* For inlets-pro you must open all ports since the client advertises, not the server
* This API is event-driven and is expected to use polling from the Kubernetes Operator or inletsctl CLI, not callbacks or waits
* Do not use any wait or blocking calls, all API calls should return ideally within < 1s
* Document how you chose any image or configuration, so that the code can be maintained, so that means links and `// comments`
* All provisioning code should detect the correct "status" for the provider and set the standard known value
* Always show your testing in PRs.

If you would like to add a provider please propose it with an Issue, to make sure that the community are happy to accept the change, and to maintain the code on an ongoing basis.

## Maintainers for each provider

* DigitalOcean, Packet, Civo - [alexellis](https://github.com/alexellis/)
* Scaleway - [alexandrevilain](https://github.com/alexandrevilain/)
* AWS EC2 - [adamjohnson01](https://github.com/adamjohnson01/)
* GCE - [utsavanand2](https://github.com/utsavanand2/)
