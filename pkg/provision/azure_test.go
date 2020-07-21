package provision

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

func Test_Azure_Auth_Contents_Invalid(t *testing.T) {
	_, err := NewAzureProvisioner("SubscriptionID", "invalid contents")

	if err == nil {
		t.Errorf("want: error, but got: nil")
	}
}

func Test_Azure_Build_Group_Deployment_Name(t *testing.T) {
	hostID := buildAzureHostID("inlets-peaceful-chaum1", "deployments-e589c808-3936-4bd9-b558-36640eb98cb0")
	if hostID != "inlets-peaceful-chaum1|deployments-e589c808-3936-4bd9-b558-36640eb98cb0" {
		t.Errorf("want: inlets-peaceful-chaum1|deployments-e589c808-3936-4bd9-b558-36640eb98cb, but got: %s", hostID)
	}
}

func Test_Azure_Parse_Group_Deployment_Name_Success(t *testing.T) {
	resourceGroupName, deploymentName, err := getAzureFieldsFromID("inlets-peaceful-chaum1|deployments-e589c808-3936-4bd9-b558-36640eb98cb0")
	if err != nil {
		t.Errorf("want: nil, but got: %s", err.Error())
	}
	if resourceGroupName != "inlets-peaceful-chaum1" {
		t.Errorf("want: inlets-peaceful-chaum1, but got: %s", resourceGroupName)
	}
	if deploymentName != "deployments-e589c808-3936-4bd9-b558-36640eb98cb0" {
		t.Errorf("want: deployments-e589c808-3936-4bd9-b558-36640eb98cb0, but got: %s", deploymentName)
	}
}

func Test_Azure_Parse_Group_Deployment_Name_Fail(t *testing.T) {
	_, _, err := getAzureFieldsFromID("INVALID_ID")
	if err == nil {
		t.Errorf("want: error, but got nil")
	}
}

func Test_Azure_Template(t *testing.T) {
	want := map[string]interface{}{
		"$schema":        "http://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
		"contentVersion": "1.0.0.0",
		"outputs": map[string]interface{}{
			"adminUsername": map[string]interface{}{
				"type":  "string",
				"value": "[parameters('adminUsername')]",
			},
			"publicIP": map[string]interface{}{
				"type":  "string",
				"value": "[reference(resourceId('Microsoft.Network/publicIPAddresses', parameters('publicIpAddressName')), '2019-02-01', 'Full').properties.ipAddress]",
			},
		},
		"parameters": map[string]interface{}{
			"addressPrefixes": map[string]interface{}{
				"type": "array",
			},
			"adminPassword": map[string]interface{}{
				"type": "secureString",
			},
			"adminUsername": map[string]interface{}{
				"type": "string",
			},
			"customData": map[string]interface{}{
				"type": "string",
			},
			"location": map[string]interface{}{
				"type": "string",
			},
			"networkInterfaceName": map[string]interface{}{
				"type": "string",
			},
			"networkSecurityGroupName": map[string]interface{}{
				"type": "string",
			},
			"networkSecurityGroupRules": map[string]interface{}{
				"type": "array",
			},
			"osDiskType": map[string]interface{}{
				"type": "string",
			},
			"publicIpAddressName": map[string]interface{}{
				"type": "string",
			},
			"subnetName": map[string]interface{}{
				"type": "string",
			},
			"subnets": map[string]interface{}{
				"type": "array",
			},
			"virtualMachineName": map[string]interface{}{
				"type": "string",
			},
			"virtualMachineRG": map[string]interface{}{
				"type": "string",
			},
			"virtualMachineSize": map[string]interface{}{
				"type": "string",
			},
			"virtualNetworkName": map[string]interface{}{
				"type": "string",
			},
		},
		"resources": []interface{}{
			map[string]interface{}{
				"apiVersion": "2019-07-01",
				"dependsOn": []interface{}{
					"[concat('Microsoft.Network/networkSecurityGroups/', parameters('networkSecurityGroupName'))]",
					"[concat('Microsoft.Network/virtualNetworks/', parameters('virtualNetworkName'))]",
					"[concat('Microsoft.Network/publicIpAddresses/', parameters('publicIpAddressName'))]",
				},
				"location": "[parameters('location')]",
				"name":     "[parameters('networkInterfaceName')]",
				"properties": map[string]interface{}{
					"ipConfigurations": []interface{}{
						map[string]interface{}{
							"name": "ipconfig1",
							"properties": map[string]interface{}{
								"privateIPAllocationMethod": "Dynamic",
								"publicIpAddress": map[string]interface{}{
									"id": "[resourceId(resourceGroup().name, 'Microsoft.Network/publicIpAddresses', parameters('publicIpAddressName'))]",
								},
								"subnet": map[string]interface{}{
									"id": "[variables('subnetRef')]",
								},
							},
						},
					},
					"networkSecurityGroup": map[string]interface{}{
						"id": "[variables('nsgId')]",
					},
				},
				"type": "Microsoft.Network/networkInterfaces",
			},
			map[string]interface{}{
				"apiVersion": "2019-02-01",
				"location":   "eastus",
				"name":       "[parameters('networkSecurityGroupName')]",
				"properties": map[string]interface{}{
					"securityRules": "[parameters('networkSecurityGroupRules')]",
				},
				"type": "Microsoft.Network/networkSecurityGroups",
			},
			map[string]interface{}{
				"apiVersion": "2019-04-01",
				"location":   "eastus",
				"name":       "[parameters('virtualNetworkName')]",
				"properties": map[string]interface{}{
					"addressSpace": map[string]interface{}{
						"addressPrefixes": "[parameters('addressPrefixes')]",
					},
					"subnets": "[parameters('subnets')]",
				},
				"type": "Microsoft.Network/virtualNetworks",
			},
			map[string]interface{}{
				"apiVersion": "2019-02-01",
				"location":   "eastus",
				"name":       "[parameters('publicIpAddressName')]",
				"properties": map[string]interface{}{
					"publicIpAllocationMethod": "Static",
				},
				"sku": map[string]interface{}{
					"name": "Basic",
				},
				"type": "Microsoft.Network/publicIpAddresses",
			},
			map[string]interface{}{
				"apiVersion": "2019-07-01",
				"dependsOn": []interface{}{
					"[concat('Microsoft.Network/networkInterfaces/', parameters('networkInterfaceName'))]",
				},
				"location": "[parameters('location')]",
				"name":     "[parameters('virtualMachineName')]",
				"properties": map[string]interface{}{
					"hardwareProfile": map[string]interface{}{
						"vmSize": "[parameters('virtualMachineSize')]",
					},
					"networkProfile": map[string]interface{}{
						"networkInterfaces": []interface{}{
							map[string]interface{}{
								"id": "[resourceId('Microsoft.Network/networkInterfaces', parameters('networkInterfaceName'))]",
							},
						},
					},
					"osProfile": map[string]interface{}{
						"adminPassword": "[parameters('adminPassword')]",
						"adminUsername": "[parameters('adminUsername')]",
						"computerName":  "[parameters('virtualMachineName')]",
						"customData":    "[base64(parameters('customData'))]",
					},
					"storageProfile": map[string]interface{}{
						"imageReference": map[string]interface{}{
							"offer":     "UbuntuServer",
							"publisher": "Canonical",
							"sku":       "16.04-LTS",
							"version":   "latest",
						},
						"osDisk": map[string]interface{}{
							"createOption": "fromImage",
							"managedDisk": map[string]interface{}{
								"storageAccountType": "[parameters('osDiskType')]",
							},
						},
					},
				},
				"type": "Microsoft.Compute/virtualMachines",
			},
		},
		"variables": map[string]interface{}{
			"nsgId":     "[resourceId(resourceGroup().name, 'Microsoft.Network/networkSecurityGroups', parameters('networkSecurityGroupName'))]",
			"subnetRef": "[concat(variables('vnetId'), '/subnets/', parameters('subnetName'))]",
			"vnetId":    "[resourceId(resourceGroup().name,'Microsoft.Network/virtualNetworks', parameters('virtualNetworkName'))]",
		},
	}
	host := BasicHost{
		Name:     "test",
		OS:       "Additional.imageOffer",
		Plan:     "Standard_B1ls",
		Region:   "eastus",
		UserData: "",
		Additional: map[string]string{
			"inlets-port":    "8080",
			"pro":            "8123",
			"imagePublisher": "Canonical",
			"imageOffer":     "UbuntuServer",
			"imageSku":       "16.04-LTS",
			"imageVersion":   "latest",
		},
	}
	template := getTemplate(host)
	templateBytes, _ := json.Marshal(template)
	wantBytes, _ := json.Marshal(want)
	if !bytes.Equal(wantBytes, templateBytes) {
		t.Errorf("want: %v, but got: %v", want, template)
	}
}

func Test_Azure_Parameters(t *testing.T) {
	want := map[string]interface{}{
		"addressPrefixes": map[string]interface{}{
			"value": []interface{}{
				"10.0.0.0/24",
			},
		},
		"adminPassword": map[string]interface{}{
			"value": "k9eY3m0RY7PYnFAs",
		},
		"adminUsername": map[string]interface{}{
			"value": "inletsuser",
		},
		"customData": map[string]interface{}{
			"value": "foo-bar-baz",
		},
		"location": map[string]interface{}{
			"value": "eastus",
		},
		"networkInterfaceName": map[string]interface{}{
			"value": "inlets-vm-nic",
		},
		"networkSecurityGroupName": map[string]interface{}{
			"value": "inlets-vm-nsg",
		},
		"networkSecurityGroupRules": map[string]interface{}{
			"value": []interface{}{
				map[string]interface{}{
					"name": "HTTPS",
					"properties": map[string]interface{}{
						"access":                   "Allow",
						"destinationAddressPrefix": "*",
						"destinationPortRange":     "443",
						"direction":                "Inbound",
						"priority":                 320,
						"protocol":                 "TCP",
						"sourceAddressPrefix":      "*",
						"sourcePortRange":          "*",
					},
				},
				map[string]interface{}{
					"name": "HTTP",
					"properties": map[string]interface{}{
						"access":                   "Allow",
						"destinationAddressPrefix": "*",
						"destinationPortRange":     "80",
						"direction":                "Inbound",
						"priority":                 340,
						"protocol":                 "TCP",
						"sourceAddressPrefix":      "*",
						"sourcePortRange":          "*",
					},
				},
				map[string]interface{}{
					"name": "HTTP8080",
					"properties": map[string]interface{}{
						"access":                   "Allow",
						"destinationAddressPrefix": "*",
						"destinationPortRange":     "8080",
						"direction":                "Inbound",
						"priority":                 360,
						"protocol":                 "TCP",
						"sourceAddressPrefix":      "*",
						"sourcePortRange":          "*",
					},
				},
			},
		},
		"osDiskType": map[string]interface{}{
			"value": "Standard_LRS",
		},
		"publicIpAddressName": map[string]interface{}{
			"value": "inlets-ip",
		},
		"subnetName": map[string]interface{}{
			"value": "default",
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
		"virtualMachineName": map[string]interface{}{
			"value": "peaceful-chaum1",
		},
		"virtualMachineRG": map[string]interface{}{
			"value": "inlets-peaceful-chaum1",
		},
		"virtualMachineSize": map[string]interface{}{
			"value": "Standard_B1ls",
		},
		"virtualNetworkName": map[string]interface{}{
			"value": "inlets-vnet",
		},
	}
	ctx := context.Background()
	provisioner := AzureProvisioner{
		subscriptionId:    "",
		resourceGroupName: "inlets-peaceful-chaum1",
		authorizer:        nil,
		ctx:               ctx,
	}
	host := BasicHost{
		Name:     "peaceful-chaum1",
		OS:       "Additional.imageOffer",
		Plan:     "Standard_B1ls",
		Region:   "eastus",
		UserData: "foo-bar-baz",
		Additional: map[string]string{
			"inlets-port":    "8080",
			"pro":            "8123",
			"imagePublisher": "Canonical",
			"imageOffer":     "UbuntuServer",
			"imageSku":       "16.04-LTS",
			"imageVersion":   "latest",
			"adminPassword":  "k9eY3m0RY7PYnFAs",
		},
	}
	parameters := getParameters(&provisioner, host)
	parametersBytes, _ := json.Marshal(parameters)
	wantBytes, _ := json.Marshal(want)
	if !bytes.Equal(parametersBytes, wantBytes) {
		t.Errorf("want: %v, but got: %v", want, parameters)
	}
}
