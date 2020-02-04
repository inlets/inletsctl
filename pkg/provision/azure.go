package provision

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2017-12-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2017-09-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/google/uuid"
	"github.com/sethvargo/go-password/password"
	"log"
)

type AzureProvisioner struct {
	subscriptionId    string
	resourceGroupName string
	deploymentName    string
	authorizer        autorest.Authorizer
	ctx               context.Context
}

func NewAzureProvisioner(subscriptionId string) (*AzureProvisioner, error) {
	authorizer, err := auth.NewAuthorizerFromFile(azure.PublicCloud.ResourceManagerEndpoint)
	ctx := context.Background()
	return &AzureProvisioner{
		subscriptionId: subscriptionId,
		authorizer:     authorizer,
		ctx:            ctx,
	}, err
}

// Provision provisions a new Azure instance as an exit node
func (p *AzureProvisioner) Provision(host BasicHost) (*ProvisionedHost, error) {

	log.Printf("Provisioning host with Azure\n")

	p.resourceGroupName = "inlets-" + host.Name
	p.deploymentName = "inlets-deploy-" + uuid.New().String()

	log.Printf("Creating resource group %s", p.resourceGroupName)
	group, err := createGroup(p, host)
	if err != nil {
		return nil, err
	}
	log.Printf("Resource group created %s", *group.Name)

	log.Printf("Creating deployment %s", p.deploymentName)
	err = createDeployment(p, host)
	if err != nil {
		return nil, err
	}
	return &ProvisionedHost{
		IP:     "Creating",
		ID:     p.resourceGroupName,
		Status: ActiveStatus,
	}, nil
}

// Status checks the status of the provisioning Azure exit node
func (p *AzureProvisioner) Status(id string) (*ProvisionedHost, error) {
	deploymentsClient := resources.NewDeploymentsClient(p.subscriptionId)
	deploymentsClient.Authorizer = p.authorizer
	deployment, err := deploymentsClient.Get(p.ctx, p.resourceGroupName, p.deploymentName)
	if err != nil {
		return nil, err
	}
	var deploymentStatus string
	if *deployment.Properties.ProvisioningState == "Succeeded" {
		deploymentStatus = ActiveStatus
	} else {
		deploymentStatus = *deployment.Properties.ProvisioningState
		if deploymentStatus == "Running" {
			deploymentStatus = "deploying"
		}
	}
	IP := "Creating"
	if deploymentStatus == ActiveStatus {
		IP = deployment.Properties.Outputs.(map[string]interface{})["publicIP"].(map[string]interface{})["value"].(string)
	}
	return &ProvisionedHost{
		IP:     IP,
		ID:     id,
		Status: deploymentStatus,
	}, nil
}

// Delete deletes the Azure exit node
func (p *AzureProvisioner) Delete(request HostDeleteRequest) error {
	groupsClient := resources.NewGroupsClient(p.subscriptionId)
	groupsClient.Authorizer = p.authorizer
	groupDeleteFuture, err := groupsClient.Delete(p.ctx, request.ID)
	if err != nil {
		return err
	}
	log.Printf("Waiting for deletion completion (~10 mins)")
	err = groupDeleteFuture.Future.WaitForCompletionRef(p.ctx, groupsClient.BaseClient.Client)
	if err != nil {
		return err
	}
	log.Printf("Done deleting resources")
	_, err = groupDeleteFuture.Result(groupsClient)
	return err
}

func createGroup(p *AzureProvisioner, host BasicHost) (group resources.Group, err error) {
	groupsClient := resources.NewGroupsClient(p.subscriptionId)
	groupsClient.Authorizer = p.authorizer

	return groupsClient.CreateOrUpdate(
		p.ctx,
		p.resourceGroupName,
		resources.Group{
			Location: to.StringPtr(host.Region)})
}

func getSecurityRule(name string, priority int, protocol, destPortRange string) map[string]interface{} {
	return map[string]interface{}{
		"name": name,
		"properties": map[string]interface{}{
			"priority":                 priority,
			"protocol":                 protocol,
			"access":                   "Allow",
			"direction":                "Inbound",
			"sourceAddressPrefix":      "*",
			"sourcePortRange":          "*",
			"destinationAddressPrefix": "*",
			"destinationPortRange":     destPortRange,
		},
	}
}

func azureParameterType(typeName string) map[string]interface{} {
	return map[string]interface{}{
		"type": typeName,
	}
}

func azureParameterValue(typeValue string) map[string]interface{} {
	return map[string]interface{}{
		"value": typeValue,
	}
}

func getTemplateParameterDefinition() map[string]interface{} {
	return map[string]interface{}{
		"location":                  azureParameterType("string"),
		"networkInterfaceName":      azureParameterType("string"),
		"networkSecurityGroupName":  azureParameterType("string"),
		"networkSecurityGroupRules": azureParameterType("array"),
		"subnetName":                azureParameterType("string"),
		"virtualNetworkName":        azureParameterType("string"),
		"addressPrefixes":           azureParameterType("array"),
		"subnets":                   azureParameterType("array"),
		"publicIpAddressName":       azureParameterType("string"),
		"virtualMachineName":        azureParameterType("string"),
		"virtualMachineRG":          azureParameterType("string"),
		"osDiskType":                azureParameterType("string"),
		"virtualMachineSize":        azureParameterType("string"),
		"adminUsername":             azureParameterType("string"),
		"adminPassword":             azureParameterType("secureString"),
		"customData":                azureParameterType("string"),
	}
}

func getTemplateResourceVirtualMachine(host BasicHost) map[string]interface{} {
	return map[string]interface{}{
		"name":       "[parameters('virtualMachineName')]",
		"type":       "Microsoft.Compute/virtualMachines",
		"apiVersion": "2019-07-01",
		"location":   "[parameters('location')]",
		"dependsOn": []interface{}{
			"[concat('Microsoft.Network/networkInterfaces/', parameters('networkInterfaceName'))]",
		},
		"properties": map[string]interface{}{
			"hardwareProfile": map[string]interface{}{
				"vmSize": "[parameters('virtualMachineSize')]",
			},
			"storageProfile": map[string]interface{}{
				"osDisk": map[string]interface{}{
					"createOption": "fromImage",
					"managedDisk": map[string]interface{}{
						"storageAccountType": "[parameters('osDiskType')]",
					},
				},
				"imageReference": map[string]interface{}{
					"publisher": host.Additional["imagePublisher"],
					"offer":     host.Additional["imageOffer"],
					"sku":       host.Additional["imageSku"],
					"version":   host.Additional["imageVersion"],
				},
			},
			"networkProfile": map[string]interface{}{
				"networkInterfaces": []interface{}{
					map[string]interface{}{
						"id": "[resourceId('Microsoft.Network/networkInterfaces', parameters('networkInterfaceName'))]",
					},
				},
			},
			"osProfile": map[string]interface{}{
				"computerName":  "[parameters('virtualMachineName')]",
				"adminUsername": "[parameters('adminUsername')]",
				"adminPassword": "[parameters('adminPassword')]",
				"customData":    "[base64(parameters('customData'))]",
			},
		},
	}
}
func getTemplateResourceNetworkInterface() map[string]interface{} {
	return map[string]interface{}{
		"name":       "[parameters('networkInterfaceName')]",
		"type":       "Microsoft.Network/networkInterfaces",
		"apiVersion": "2019-07-01",
		"location":   "[parameters('location')]",
		"dependsOn": []interface{}{
			"[concat('Microsoft.Network/networkSecurityGroups/', parameters('networkSecurityGroupName'))]",
			"[concat('Microsoft.Network/virtualNetworks/', parameters('virtualNetworkName'))]",
			"[concat('Microsoft.Network/publicIpAddresses/', parameters('publicIpAddressName'))]",
		},
		"properties": map[string]interface{}{
			"ipConfigurations": []interface{}{
				map[string]interface{}{
					"name": "ipconfig1",
					"properties": map[string]interface{}{
						"subnet": map[string]interface{}{
							"id": "[variables('subnetRef')]",
						},
						"privateIPAllocationMethod": "Dynamic",
						"publicIpAddress": map[string]interface{}{
							"id": "[resourceId(resourceGroup().name, 'Microsoft.Network/publicIpAddresses', parameters('publicIpAddressName'))]",
						},
					},
				},
			},
			"networkSecurityGroup": map[string]interface{}{
				"id": "[variables('nsgId')]",
			},
		},
	}
}
func getTemplate(host BasicHost) map[string]interface{} {
	return map[string]interface{}{
		"$schema":        "http://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		"contentVersion": "1.0.0.0",
		"parameters":     getTemplateParameterDefinition(),
		"variables": map[string]interface{}{
			"nsgId":     "[resourceId(resourceGroup().name, 'Microsoft.Network/networkSecurityGroups', parameters('networkSecurityGroupName'))]",
			"vnetId":    "[resourceId(resourceGroup().name,'Microsoft.Network/virtualNetworks', parameters('virtualNetworkName'))]",
			"subnetRef": "[concat(variables('vnetId'), '/subnets/', parameters('subnetName'))]",
		},
		"resources": []interface{}{
			getTemplateResourceNetworkInterface(),
			map[string]interface{}{
				"name":       "[parameters('networkSecurityGroupName')]",
				"type":       "Microsoft.Network/networkSecurityGroups",
				"apiVersion": "2019-02-01",
				"location":   host.Region,
				"properties": map[string]interface{}{
					"securityRules": "[parameters('networkSecurityGroupRules')]",
				},
			},
			map[string]interface{}{
				"name":       "[parameters('virtualNetworkName')]",
				"type":       "Microsoft.Network/virtualNetworks",
				"apiVersion": "2019-04-01",
				"location":   host.Region,
				"properties": map[string]interface{}{
					"addressSpace": map[string]interface{}{
						"addressPrefixes": "[parameters('addressPrefixes')]",
					},
					"subnets": "[parameters('subnets')]",
				},
			},
			map[string]interface{}{
				"name":       "[parameters('publicIpAddressName')]",
				"type":       "Microsoft.Network/publicIpAddresses",
				"apiVersion": "2019-02-01",
				"location":   host.Region,
				"properties": map[string]interface{}{
					"publicIpAllocationMethod": network.Static,
				},
				"sku": map[string]interface{}{
					"name": network.PublicIPAddressSkuNameBasic,
				},
			},
			getTemplateResourceVirtualMachine(host),
		},
		"outputs": map[string]interface{}{
			"adminUsername": map[string]interface{}{
				"type":  "string",
				"value": "[parameters('adminUsername')]",
			},
			"publicIP": map[string]interface{}{
				"type":  "string",
				"value": "[reference(resourceId('Microsoft.Network/publicIPAddresses', parameters('publicIpAddressName')), '2019-02-01', 'Full').properties.ipAddress]",
				// See also https://docs.microsoft.com/en-us/azure/azure-resource-manager/templates/template-functions-resource#reference
				// See also https://docs.microsoft.com/en-us/azure/templates/microsoft.network/2019-02-01/publicipaddresses
			},
		},
	}
}

func getParameters(p *AzureProvisioner, host BasicHost) (parameters map[string]interface{}, err error) {
	adminPassword, err := password.Generate(16, 4, 0, false, true)
	if err != nil {
		return
	}
	host.Additional["adminPassword"] = adminPassword
	return map[string]interface{}{
		"location":                 azureParameterValue(host.Region),
		"networkInterfaceName":     azureParameterValue("inlets-vm-nic"),
		"networkSecurityGroupName": azureParameterValue("inlets-vm-nsg"),
		"networkSecurityGroupRules": map[string]interface{}{
			"value": []interface{}{
				getSecurityRule("SSH", 300, "TCP", "22"),
				getSecurityRule("HTTPS", 320, "TCP", "443"),
				getSecurityRule("HTTP", 340, "TCP", "80"),
				getSecurityRule("HTTP8080", 360, "TCP", "8080"),
			},
		},
		"subnetName":         azureParameterValue("default"),
		"virtualNetworkName": azureParameterValue("inlets-vnet"),
		"addressPrefixes": map[string]interface{}{
			"value": []interface{}{
				"10.0.0.0/24",
			},
		},
		"subnets": map[string]interface{}{
			"value": []interface{}{
				map[string]interface{}{
					"name": "default",
					"properties": map[string]interface{}{
						"addressPrefix": "10.0.0.0/24",
					},
				},
			},
		},
		"publicIpAddressName": azureParameterValue("inlets-ip"),
		"virtualMachineName":  azureParameterValue(host.Name),
		"virtualMachineRG":    azureParameterValue(p.resourceGroupName),
		"osDiskType": map[string]interface{}{
			"value": compute.StandardLRS,
		},
		"virtualMachineSize": azureParameterValue(host.Plan),
		"adminUsername":      azureParameterValue("inletsuser"),
		"adminPassword":      azureParameterValue(adminPassword),
		"customData":         azureParameterValue(host.UserData),
	}, nil
}

func createDeployment(p *AzureProvisioner, host BasicHost) (err error) {
	template := getTemplate(host)
	params, err := getParameters(p, host)
	if err != nil {
		return
	}
	deploymentsClient := resources.NewDeploymentsClient(p.subscriptionId)
	deploymentsClient.Authorizer = p.authorizer

	_, err = deploymentsClient.CreateOrUpdate(
		p.ctx,
		p.resourceGroupName,
		p.deploymentName,
		resources.Deployment{
			Properties: &resources.DeploymentProperties{
				Template:   template,
				Parameters: params,
				Mode:       resources.Complete,
			},
		},
	)
	return
}
