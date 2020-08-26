# provision cloud hosts with user-data

This package can be used to provision cloud hosts using a simple CRUD-style API along with a cloud-init user-data script. It could be used to automate anything from k3s clusters, to blogs, or CI runners. We use it to create the cheapest possible hosts in the cloud with a public IP address.

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

The first rule about the `provision` package is that we don't do SSH. Key management and statefulness are out of scope. Cheap servers should be treated like cattle, not pets. `ssh` may well be enabled by default, but is out of scope for management. For instance, with DigitalOcean, you can get a root password if you need to log in. Configure as much as you can via cloud-init / user-data.

* Use the Ubuntu 16.04 LTS image
* Select the cheapest plan and update the [README](https://github.com/inlets/inletsctl/blob/master/README.md) with the estimated monthly cost
* For inlets OSS open just the required ports
* For inlets-pro you must open all ports since the client advertises, not the server
* This API is event-driven and is expected to use polling from the Kubernetes Operator or inletsctl CLI, not callbacks or waits
* Do not use any wait or blocking calls, all API calls should return ideally within < 1s
* Document how you chose any image or configuration, so that the code can be maintained, so that means links and `// comments`
* All provisioning code should detect the correct "status" for the provider and set the standard known value
* Always show your testing in PRs.

Finally please [add an example to the documentation](https://docs.inlets.dev/#/tools/inletsctl?id=inletsctl-reference-documentation) for your provider in the [inlets/docs](https://github.com/inlets/docs) repo.

If you would like to add a provider please propose it with an Issue, to make sure that the community are happy to accept the change, and to maintain the code on an ongoing basis.

## Maintainers for each provider

* DigitalOcean, Packet, Civo - [alexellis](https://github.com/alexellis/)
* Scaleway - [alexandrevilain](https://github.com/alexandrevilain/)
* AWS EC2 - [adamjohnson01](https://github.com/adamjohnson01/)
* GCE - [utsavanand2](https://github.com/utsavanand2/)
* Azure, Linode - [zechenbit](https://github.com/zechenbit/)
* Hetzner [Johannestegner](https://github.com/johannestegner)