# Azure Hub-Spoke Terraform module

## Table of Contents

1. [Description](#description)
2. [Prerequisites](#prerequisites)
3. [Getting Started](#getting-started)
4. [Versions](#versions)
5. [Parameters](#parameters)
6. [Return Values](#return-values)
7. [Examples](#examples)

## Description

Welcome to the Azure Hub-Spoke Terraform module. This module is designed to make the deployment of any hub-spoke network topology as easy as 1-2-3. The module is built on a concept of a single input variable called 'Typology_object', which can then contain a huge subset of custom configurations. The module supports name injection, automatic subnetting, Point-to-Site VPN, firewall, routing, and much more! Because it's built for Azure, it uses the architectural design from the Microsoft CAF concepts, which can be read more about at <a href="https://learn.microsoft.com/en-us/azure/architecture/networking/architecture/hub-spoke?tabs=cli">Hub-Spoke typology</a>

OBS. The module does NOT support building hub-spokes over multiple subscriptions YET, but is planned to be released in version 1.1.0

Just below here, two different visual examples of types of hub-spokes can be seen. Both can be directly deployed with the module, see the for the actual code.

<b>Example 1: Deployment of a simple hub-spoke</b>
</br>
</br>
<img src="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/Graphic%20material/DrawIO/Simple-hub-spoke-Simple-Hub-Spoke.png"/>
</br>
</br>
</br>
<b>Example 2: Deployment of an advanced hub-spoke</b>
</br>
</br>
<img src="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/Graphic%20material/DrawIO/Simple-hub-spoke-Complex%20Hub-Spoke.drawio.png"/>

[Back to the top](#table-of-contents)
## Prerequisites

Before using this module, make sure you have the following:
- Active Azure Subscription
  - Must either have RBAC roles:
    - Contributor
- Installed terraform (download [here](https://www.terraform.io/downloads.html))
- Azure CLI installed for authentication (download [here](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli))

[Back to the top](#table-of-contents)
## Getting Started
Remember to have read the chapter <a href="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-hub-spoke-/readme.md#prerequisites">Prerequisites</a> before getting started.

1. Create a new terraform script file in any folder
2. Define terraform boilerplate code
```hcl
provider "azurerm" {
  features{}
  //Can define a specific context, but we will use an interrogated one.
}
```
3. Login to Azure with an active subscription using az cli
```powershell
az login //Web browser interactive prompt.
```
4. Define the module definition
```hcl
module "simple_hub_spoke" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=1.0.0" //Always use a specific version of the module
  
  typology_object = {
    name_prefix = "test" #Will add a prefix of "test" On all resources - Can also be set as "name_suffix" Which will rotate names. See the input variables description for more details

    hub_object = {
      network = {
        address_spaces = ["10.0.0.0/24"]
      }
    }

    spoke_objects = [
      {
        network = {
           address_spaces = ["172.16.0.0/20"] //All subnets in the spoke will use this to CIDR from, because "address_prefix" IS not defined for ANY subnet

           subnet_objects = [
            {
                #Will use default settings and TAKE THE FIRST /26 FROM the address space
                use_first_subnet = true
            },
            {
                #Will use default settings - BUT TAKE THE LAST /26 FROM the address space
                use_last_subnet = true
            }
           ]
        }
      }
    ]
  }
}
```
5. Run terraform init & terraform apply
```hcl
terraform init
terraform apply

//Plan output
Plan: 8 to add, 0 to change, 0 to destroy.
Terraform will perform the following actions:

  # module.simple_hub_spoke.azurerm_resource_group.rg_object["rg-test-hub"] will be created
  + resource "azurerm_resource_group" "rg_object" {
      + id       = (known after apply)
      + location = "westeurope"
      + name     = "rg-test-hub"
    }

  # module.simple_hub_spoke.azurerm_resource_group.rg_object["rg-test-spoke1"] will be created
  + resource "azurerm_resource_group" "rg_object" {
      + id       = (known after apply)
      + location = "westeurope"
      + name     = "rg-test-spoke1"
    }

  # module.simple_hub_spoke.azurerm_subnet.subnet_object["subnet1-test-spoke1"] will be created
  + resource "azurerm_subnet" "subnet_object" {
      + address_prefixes                               = [
          + "172.16.0.0/26",
        ]
      + default_outbound_access_enabled                = true
      + enforce_private_link_endpoint_network_policies = (known after apply)
      + enforce_private_link_service_network_policies  = (known after apply)
      + id                                             = (known after apply)
      + name                                           = "subnet1-test-spoke1"
      + private_endpoint_network_policies              = (known after apply)
      + private_endpoint_network_policies_enabled      = (known after apply)
      + private_link_service_network_policies_enabled  = (known after apply)
      + resource_group_name                            = "rg-test-spoke1"
      + virtual_network_name                           = "vnet-test-spoke1"
    }

  # module.simple_hub_spoke.azurerm_subnet.subnet_object["subnet2-test-spoke1"] will be created
  + resource "azurerm_subnet" "subnet_object" {
      + address_prefixes                               = [
          + "172.16.8.128/26",
        ]
      + default_outbound_access_enabled                = true
      + enforce_private_link_endpoint_network_policies = (known after apply)
      + enforce_private_link_service_network_policies  = (known after apply)
      + id                                             = (known after apply)
      + name                                           = "subnet2-test-spoke1"
      + private_endpoint_network_policies              = (known after apply)
      + private_endpoint_network_policies_enabled      = (known after apply)
      + private_link_service_network_policies_enabled  = (known after apply)
      + resource_group_name                            = "rg-test-spoke1"
      + virtual_network_name                           = "vnet-test-spoke1"
    }

  # module.simple_hub_spoke.azurerm_virtual_network.vnet_object["vnet-test-hub"] will be created
  + resource "azurerm_virtual_network" "vnet_object" {
      + address_space       = [
          + "10.0.0.0/24",
        ]
      + dns_servers         = (known after apply)
      + guid                = (known after apply)
      + id                  = (known after apply)
      + location            = "westeurope"
      + name                = "vnet-test-hub"
      + resource_group_name = "rg-test-hub"
      + subnet              = (known after apply)
    }

  # module.simple_hub_spoke.azurerm_virtual_network.vnet_object["vnet-test-spoke1"] will be created
  + resource "azurerm_virtual_network" "vnet_object" {
      + address_space       = [
          + "172.16.0.0/20",
        ]
      + dns_servers         = (known after apply)
      + guid                = (known after apply)
      + id                  = (known after apply)
      + location            = "westeurope"
      + name                = "vnet-test-spoke1"
      + resource_group_name = "rg-test-spoke1"
      + subnet              = (known after apply)
    }

  # module.simple_hub_spoke.azurerm_virtual_network_peering.peering_object["peering-from-hub-to-spoke1"] will be created
  + resource "azurerm_virtual_network_peering" "peering_object" {
      + allow_forwarded_traffic      = true
      + allow_gateway_transit        = true
      + allow_virtual_network_access = true
      + id                           = (known after apply)
      + name                         = "peering-from-hub-to-spoke1"
      + remote_virtual_network_id    = (known after apply)
      + resource_group_name          = "rg-test-hub"
      + use_remote_gateways          = false
      + virtual_network_name         = "vnet-test-hub"
    }

  # module.simple_hub_spoke.azurerm_virtual_network_peering.peering_object["peering-from-spoke1-to-hub"] will be created
  + resource "azurerm_virtual_network_peering" "peering_object" {
      + allow_forwarded_traffic      = true
      + allow_gateway_transit        = false
      + allow_virtual_network_access = true
      + id                           = (known after apply)
      + name                         = "peering-from-spoke1-to-hub"
      + remote_virtual_network_id    = (known after apply)
      + resource_group_name          = "rg-test-spoke1"
      + use_remote_gateways          = false
      + virtual_network_name         = "vnet-test-spoke1"
    }

Plan: 8 to add, 0 to change, 0 to destroy.
────────────────────────────────────────────────────────────────────────────────── 

//press yes
yes

//apply output
Apply complete! Resources: 8 added, 0 changed, 0 destroyed.
```

6. There is a ton more to explore with the module, see the <a href="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-hub-spoke/readme.md#examples">Examples</a> for details

[Back to the top](#table-of-contents)
## Versions
The table below outlines the compatibility of the module:

Please take note of the 'Module version' among the provider utilized by the module. Keep in mind that there WILL be a required minimum version, and this requirement can vary with each module version.

<b>"Module version" 1.0.0 requires the following provider versions:</b>

| Provider name | Provider url | Minimum version |
| -------------- | ------------ | ---------------- |
| azurerm        | [hashicorp/azurerm](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs) | 3.99.0 |

For the latest updates of the terraform module, check the <a href="https://github.com/ChristofferWin/codeterraform/releases">release page</a>

Make sure, if using a static version, that it follows above version table, otherwise the following error will occur:
```hcl
//Showcasing issue with using too old providers
terraform {
  required_providers {
    azurerm = {
      source = "hashicorp/azurerm"
      version = "3.64.0"
    }
  }
}

//run terraform init
terraform init

//Init results:

│ Error: Failed to query available provider packages
│
│ Could not retrieve the list of available versions for provider hashicorp/azurerm: no available releases match the given constraints 3.64.0, >= 3.99.0
```
To solve it, simply remove the version parameter OR use a version that is the minimum requirement from <a href="https://github.com/ChristofferWin/codeterraform/tree/main/terraform%20projects/modules/azurerm-hub-spoke#versions">Versions</a>:
```hcl
//Remove the version parameter entirely which causes terraform to use the latest version of azurerm
terraform {
  required_providers {
    azurerm = {
      source = "hashicorp/azurerm"
    }
  }
}

terraform init

//Init results:
- Installed hashicorp/azurerm v3.99.0 (signed by HashiCorp)

Terraform has been successfully initialized!
```
Please see the <a href="https://github.com/ChristofferWin/codeterraform/tree/main/terraform%20projects/modules/azurerm-hub-spoke#parameters">Parameters</a> section for a better understanding of what the module can take as inputs

[Back to the top](#table-of-contents)
## Parameters
For assisting in understanding the actual structure of the only input variable "typology_object" Please see below code:
```hcl
module "show_case_object" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=1.0.0"
  typology_object = { //The "root" is an OBJECT
    //Many different overall settings for the entire deployment can be set here. See below the code snippet for details.

    hub_object = { //The "hub_object" is an OBJECT - Object path is then <typology_object.hub_object>
      //Less but specific attributes can be set for the hub here. See below the code snippet for details.

      network = { //The object "network" is an OBJECT - Object path is then <typology_object.hub_object.network>
        //Multiple different attributes with relevance to network can be set for the hub here. See below the code snippet for details.

        vpn = { //The object "vpn" is an OBJECT - Object path is then <typology_object.hub_object.network.vpn>
          //Specific attributes related to configuring a Point-2-Site VPN. See below the code snippet for details.
        }

        firewall = { //The object "vpn" is an OBJECT - Object path is then <typology_object.hub_object.network.firewall>
          //Specific attributes related to configuring an Azure Firewall. See below the code snippet for details.
        }

        subnet_objects = [ //The list of objects "subnet_objects" is a LIST OF OBJECT - Object path is then <typology_object.hub_object.network.subnet_objects[index]>
          {
            //For each {} block, define specific attributes related to Azure subnets. See below the code snippet for details.
          }
        ]
      }
    }

    spoke_objects = [ //The list of objects "spoke_objects" is a LIST OF OBJECT - Object path is then <typology_object.spoke_objects[index]>
      {
        //For each {} block, many spokes can be deployed. Minimum 1. See below the code snippet for details.
      
        network = { //The object "network" is an OBJECT - Object path is then <typology_object.spoke_objects[index].network>
          //Multiple different attributes with relevance to network can be set for each spoke here. See below the code snippet for details.

          subnet_objects = [ //The list of objects "subnet_objects" is a LIST OF OBJECT - Object path is then <typology_object.spoke_objects[index].network.subnet_objects[index]>
            {
              //For each {} block, define specific attributes related to Azure subnets. See below the code snippet for details.
            }
          ]
        }
      }
    ]
  }
}
```

### Attributes on the "top" Level of the "typology_object"
1. customer_name = (optional) A string defining the name of the customer. Will be injected into the overall resource names. OBS. Using this variable requires both either "name_prefix" OR "name_suffix" AND "env_name" to be provided as well

2. location = (optional) A string defining the location of ALL resources deployed (overwrites ANY lower set location)

3. name_prefix = (optional) A string to inject a prefix into all resource names - This variable makes it so names follow a naming standard: \<resource abbreviation>\-<name_prefix>\-\<Identier, either "hub" or "spoke">

4. name_suffix = (optional) A string to inject a suffix into all resource names - This variable also makes names follow a naming standard: <Identifier, either "hub" or "spoke">\-\<name_suffix>\-\<resource abbreviation>

5. env_name = (optional) A string defining an environment name to inject into all resource names. OBS. Using this variable requires both either "name_prefix" OR "name_suffix" AND "customer_name" To be provided as well

6. dns_servers = (optional) A list of strings defining DNS server IP  to set for ALL vnets in the typology (overwrites ANY lower set DNS servers)

7. tags = (optional) A map of strings defining any tags to set on ALL vnets and resource groups (Any tags set lower will be appended to these tags set here)

8. subnets_cidr_notation = (optional) A string defining what specific subnet size that ALL subnets should have - Defaults to "/26"

### Attributes on the "hub_object" level of the "typology_object"
1. rg_name = (optional) A string defining the specific name of the hub resource group resource (Overwrites any name injection defined in the top level attributes)

2. location = (optional) A string defining the location of which to deploy the hub to (If the top level location is set, this will be overwritten)

3. tags = (optional) A map og strings defining any tags to set on the hub resources

4. network = (required) An object structured as:
    1. vnet_name = (optional) A string defining the name of the hub Azure Virtual Network resource (Overwrites any name injection defined in the top level attributes)

    2. vnet_cidr_notation = (optional) A string to be used in case you do NOT parse the attribute "address_spaces" The module will then instead use a base CIDR block of ["10.0.0.0/16] and use the attribute "vnet_cidr_notation" to subnet the "address_spaces" for the hub Azure Virtual Network resource. Must be parsed in the form of "/\<CIDR>" e.g "/24"

    3. address_spaces = (optional) A list of strings to be used in case you do NOT provide the attribute "vnet_cidr_notation" By providing a value for this attribute, you completely define the exact CIDR block for the hub Azure Virtual Network resource

    4. dns_servers = (optional) A list of strings defining DNS server IP addresses to set for the spoke Azure Virtual Network resource (Will be overwritten in case the attribute is set on the top level object)

    5. tags = (optional) A map og strings defining any tags to set on the spoke resources

    6. vnet_peering_allow_virtual_network_access = (optional) (NOT RECOMMENDED TO CHANGE) A bool used to disable whether the spoke vnet´s Azure Virtual machine resources can reach the hub

    7. vnet_peering_allow_forwarded_traffic = (optional) (NOT RECOMMENDED TO CHANGE) A bool used to disable whether the hub vnet can recieve forwarded traffic from the spoke vnet

    8. vpn = (optional) An object structured as:
       
       1. gw_name = (optional) A string to define the custom name of the Azure Virtual Network Gateway resource (Overwrites any naming injection defined in the top level object)

        2. address_space = (optional) A list of strings defining the CIDR block to be used by the Point-2-Site VPN connections, for the DHCP scope

        3. gw_sku = (optional) (NOT RECOMMENDED TO CHANGE) A string used to define the SKU for the Azure Virtual Gateway resource. Defaults to "VpnGw2"

        4. pip_name = (optional) A string defining the custom name of the Azure Public IP to be used on the VPN (Overwrites any naming injection defined in the top level object)
    
    9. firewall = (optional) An object structured as:
        
        1. name = (optional) A string to define the custom name of the Azure Firewall resource (Overwrites any naming injection defined in the top level object)

        2. sku_tier = (optional) A string defining the SKU tier of the Azure Firewall resource. Defaults to "Standard"

        3. threat_intel_mode = (optional) A bool defining whether the mode of the automatic detection shall be set to "Deny" Mode. Defaults to "Alert"

        4. pip_name = (optional) A string defining the custom name of the Azure Public IP to be used on the Firewall (Overwrites any naming injection defined in the top level object)

        5. log_name = (optional) A string defining the custom name of the Azure Log Analytics workspace resource (Overwrites any naming injection defined in the top level object)

        6. log_daily_quota_gb = (optional) A number defining the daily quota in GB that can be injested into the Azure Log Analytics workspace. Defaults to -1 which means NO limit

        7. no_logs = (optional) A bool to determine whether the module shall NOT create an Azure Log Analytics workspace and Azure Diagnostic settings for the Azure Firewall. Pr. default both resources will be created IF the Firewall is also created

        8. no_rules = (optional) A bool to determine whether the module shall NOT create Azure Firewall rules. Pr. default Azure Firewall network rules will be created IF the Firewall is also created. (The specific rules applied can be seen via [Advanced spoke](#description))
    
    10. subnet_objects = (optional) A list og objects structured as:
        
        1. name = (optional) A string defining the custom name of the Azure Subnet (Overwrites any naming injection defined in the top level object)
        
        2. use_first_subnet = (optional) A bool to use in case the attribute "address_prefix" is NOT used - Tells the module to create a subnet CIDR from the START of the CIDR block used in the deployment. See the [Examples](#examples) for more details

        3. use_last_subnet = (optional) A bool to use in case the attribute "address_prefix" is NOT used - Tells the module to create a subnet CIDR from the END of the CIDR block used in the deployment. See the [Examples](#examples) for more details

        4. address_prefix = (optional) An address space specifically defined for the subnet. Its NOT recommended to define this manually in case the overall vnets "address_spaces" Attribute is NOT populated.

        5. service_endpoints = (optional) A string defining Microft Azure Service Endpoints to add to the subnet

        6. service_endpoint_policy_ids = (optional) A set of strings defining any Azure Service Endpoint policy id's to add to the subnet

        7. delegation = (optional) A list of objects structured as:
            1. name = optional(string) A custom name to add as the display name for the deletation added to the subnet
            2. service_name_pattern = optional(string) A string defining a pattern to match a specific Azure delegation for the subnet. For a showcasing of how to use the filter see the [How to easily deploy delegations](#3-Using-the-subnet-delegation-filter-attribute-called-service_name_pattern) for more details

### Attributes on the "spoke_objects" level of the "typology_object"
1. Minimum of 1 spoke must be defined
2. All attributes on the top level of this object can be defined exactly as for the "hub_object"
3. The "network" Block is described exactly the same as for the "hub_object" With the ONLY differences being you can ONLY define "subnet_objects", no Firewall or VPN settings. See the [Examples](#examples) for more details

[Back to the top](#table-of-contents)

## Return Values
Its important to state that almost all values returned from the module is of type map. This can either be used to our advantage by making our variable references more type-safe
or we can simply use a function like 'values' to make the return value a list of object instead, where we can then simply use int index-based references like [0]

See below list of possible return values:

1. rg_return_objects = map of objects containing all the same return attributes as the provider => <a href="https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/resource_group#attributes-reference">Azurerm Resource group</a>

2. vnet_return_objects = map of objects containing all the same return attributes as the provider => <a href="https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/virtual_network#attributes-reference">Azurerm Virtual network</a>

3. subnet_return_objects = map of objects containing all the same return attributes as the provider => <a href="https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/subnet#attributes-reference">Azurerm Subnet</a>

4. rt_return_objects = map of objects containing all the same return attributes as the provider => <a href="https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/route_table#attributes-reference">Azurerm Route table</a>

5. fw_return_object = object containing all the same return attributes as the provider => <a href="https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/firewall#attributes-reference">Azurerm Firewall</a>

6. gw_return_object = object containing all the same return attributes as the provider => <a href="https://registry.terraform.io/providers/hashicorp/Azurerm/latest/docs/resources/virtual_network_gateway#attributes-reference">Azurerm Virtual network gateway</a>

7. pip_return_object = map of object containing all the same return attributes as the provider => <a href="https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/public_ip#attributes-reference">Azurerm Public IP</a>

8. log_return_object = object containing all the same return attributes as the provider => <a href="https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/log_analytics_workspace.html#attributes-reference">Azurerm Log Analytics workspace</a>

[Back to the top](#table-of-contents)
## Examples
<b>This section is split into 2 different sub sections:</b>

- <a href="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-hub-spoke/readme.md#simple-examples---separated-on-topics">Simple examples</a> = Meant to showcase how to deploy simple hub-spoke typologies
- <a href="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-hub-spoke/readme.md#advanced-examples---seperated-on-topics">Advanced examples</a> = Meant to showcase how to deploy advanced hub-spoke typologies

[Back to the top](#table-of-contents)

### Simple examples - Separated on topics
1. [Deploy a simple hub and 2 spokes with minimum config](#1-Deploy-a-simple-hub-and-2-spokes-with-minimum-config)
2. [Simple hub-spoke and ready for Bastion](#2-Simple-hub-spoke-and-ready-for-Bastion)
3. [Using the subnet delegation filter attribute called "service_name_pattern"](#3-Using-the-subnet-delegation-filter-attribute-called-service_name_pattern)



### (1) Deploy a simple hub and 2 spokes with minimum config
```hcl
module "hub_and_2_spokes" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=main"
  //We want to deploy a hub with 0 subnets and default settings
  //We want to deploy 2 spokes, with 2 subnets in each
  typology_object = {
    
    hub_object = {
      network = {
        //We wont add any custom config
      }
    }

    spoke_objects = [
      {
        network = {
          subnet_objects = [
            {
               #We will only provide an empty {}, all default values (Spoke 1, subnet 1)
            },
            {
               #We will only provide an empty {}, all default values (Spoke 1, subnet 2)
            }
          ]
        }      
      },
      {
        network = {
          subnet_objects = [
            {
                #We will only provide an empty {}, all default values (Spoke 2, subnet 1)
            },
            {
                #We will only provide an empty {}, all default values (Spoke 2, subnet 2)
            }
          ]
        }
      }
    ]
  }
}

//TF Plan output:
Plan: 14 to add, 0 to change, 0 to destroy.
Terraform will perform the following actions:

  # module.hub_and_2_spokes.azurerm_resource_group.rg_object["rg-hub"] will be created
  + resource "azurerm_resource_group" "rg_object" {
      + id       = (known after apply)
      + location = "westeurope"
      + name     = "rg-hub"
    }

  # module.hub_and_2_spokes.azurerm_resource_group.rg_object["rg-spoke1"] will be created
  + resource "azurerm_resource_group" "rg_object" {
      + id       = (known after apply)
      + location = "westeurope"
      + name     = "rg-spoke1"
    }

  # module.hub_and_2_spokes.azurerm_resource_group.rg_object["rg-spoke2"] will be created
  + resource "azurerm_resource_group" "rg_object" {
      + id       = (known after apply)
      + location = "westeurope"
      + name     = "rg-spoke2"
    }

  # module.hub_and_2_spokes.azurerm_subnet.subnet_object["subnet1-spoke1"] will be created
  + resource "azurerm_subnet" "subnet_object" {
      + address_prefixes                               = [
          + "10.0.1.0/26",
        ]
      + default_outbound_access_enabled                = true
      + enforce_private_link_endpoint_network_policies = (known after apply)
      + enforce_private_link_service_network_policies  = (known after apply)
      + id                                             = (known after apply)
      + name                                           = "subnet1-spoke1"
      + private_endpoint_network_policies              = (known after apply)
      + private_endpoint_network_policies_enabled      = (known after apply)
      + private_link_service_network_policies_enabled  = (known after apply)
      + resource_group_name                            = "rg-spoke1"
      + virtual_network_name                           = "vnet-spoke1"
    }

  # module.hub_and_2_spokes.azurerm_subnet.subnet_object["subnet1-spoke2"] will be created
  + resource "azurerm_subnet" "subnet_object" {
      + address_prefixes                               = [
          + "10.0.2.0/26",
        ]
      + default_outbound_access_enabled                = true
      + enforce_private_link_endpoint_network_policies = (known after apply)
      + enforce_private_link_service_network_policies  = (known after apply)
      + id                                             = (known after apply)
      + name                                           = "subnet1-spoke2"
      + private_endpoint_network_policies              = (known after apply)
      + private_endpoint_network_policies_enabled      = (known after apply)
      + private_link_service_network_policies_enabled  = (known after apply)
      + resource_group_name                            = "rg-spoke2"
      + virtual_network_name                           = "vnet-spoke2"
    }

  # module.hub_and_2_spokes.azurerm_subnet.subnet_object["subnet2-spoke1"] will be created
  + resource "azurerm_subnet" "subnet_object" {
      + address_prefixes                               = [
          + "10.0.1.64/26",
        ]
      + default_outbound_access_enabled                = true
      + enforce_private_link_endpoint_network_policies = (known after apply)
      + enforce_private_link_service_network_policies  = (known after apply)
      + id                                             = (known after apply)
      + name                                           = "subnet2-spoke1"
      + private_endpoint_network_policies              = (known after apply)
      + private_endpoint_network_policies_enabled      = (known after apply)
      + private_link_service_network_policies_enabled  = (known after apply)
      + resource_group_name                            = "rg-spoke1"
      + virtual_network_name                           = "vnet-spoke1"
    }

  # module.hub_and_2_spokes.azurerm_subnet.subnet_object["subnet2-spoke2"] will be created
  + resource "azurerm_subnet" "subnet_object" {
      + address_prefixes                               = [
          + "10.0.2.64/26",
        ]
      + default_outbound_access_enabled                = true
      + enforce_private_link_endpoint_network_policies = (known after apply)
      + enforce_private_link_service_network_policies  = (known after apply)
      + id                                             = (known after apply)
      + name                                           = "subnet2-spoke2"
      + private_endpoint_network_policies              = (known after apply)
      + private_endpoint_network_policies_enabled      = (known after apply)
      + private_link_service_network_policies_enabled  = (known after apply)
      + resource_group_name                            = "rg-spoke2"
      + virtual_network_name                           = "vnet-spoke2"
    }

  # module.hub_and_2_spokes.azurerm_virtual_network.vnet_object["vnet-hub"] will be created
  + resource "azurerm_virtual_network" "vnet_object" {
      + address_space       = [
          + "10.0.0.0/24",
        ]
      + dns_servers         = (known after apply)
      + guid                = (known after apply)
      + id                  = (known after apply)
      + location            = "westeurope"
      + name                = "vnet-hub"
      + resource_group_name = "rg-hub"
      + subnet              = (known after apply)
    }

  # module.hub_and_2_spokes.azurerm_virtual_network.vnet_object["vnet-spoke1"] will be created
  + resource "azurerm_virtual_network" "vnet_object" {
      + address_space       = [
          + "10.0.1.0/24",
        ]
      + dns_servers         = (known after apply)
      + guid                = (known after apply)
      + id                  = (known after apply)
      + location            = "westeurope"
      + name                = "vnet-spoke1"
      + resource_group_name = "rg-spoke1"
      + subnet              = (known after apply)
    }

  # module.hub_and_2_spokes.azurerm_virtual_network.vnet_object["vnet-spoke2"] will be created
  + resource "azurerm_virtual_network" "vnet_object" {
      + address_space       = [
          + "10.0.2.0/24",
        ]
      + dns_servers         = (known after apply)
      + guid                = (known after apply)
      + id                  = (known after apply)
      + location            = "westeurope"
      + name                = "vnet-spoke2"
      + resource_group_name = "rg-spoke2"
      + subnet              = (known after apply)
    }

  # module.hub_and_2_spokes.azurerm_virtual_network_peering.peering_object["peering-from-hub-to-spoke1"] will be created
  + resource "azurerm_virtual_network_peering" "peering_object" {
      + allow_forwarded_traffic      = true
      + allow_gateway_transit        = true
      + allow_virtual_network_access = true
      + id                           = (known after apply)
      + name                         = "peering-from-hub-to-spoke1"
      + remote_virtual_network_id    = (known after apply)
      + resource_group_name          = "rg-hub"
      + use_remote_gateways          = false
      + virtual_network_name         = "vnet-hub"
    }

  # module.hub_and_2_spokes.azurerm_virtual_network_peering.peering_object["peering-from-hub-to-spoke2"] will be created
  + resource "azurerm_virtual_network_peering" "peering_object" {
      + allow_forwarded_traffic      = true
      + allow_gateway_transit        = true
      + allow_virtual_network_access = true
      + id                           = (known after apply)
      + name                         = "peering-from-hub-to-spoke2"
      + remote_virtual_network_id    = (known after apply)
      + resource_group_name          = "rg-hub"
      + use_remote_gateways          = false
      + virtual_network_name         = "vnet-hub"
    }

  # module.hub_and_2_spokes.azurerm_virtual_network_peering.peering_object["peering-from-spoke1-to-hub"] will be created
  + resource "azurerm_virtual_network_peering" "peering_object" {
      + allow_forwarded_traffic      = true
      + allow_gateway_transit        = false
      + allow_virtual_network_access = true
      + id                           = (known after apply)
      + name                         = "peering-from-spoke1-to-hub"
      + remote_virtual_network_id    = (known after apply)
      + resource_group_name          = "rg-spoke1"
      + use_remote_gateways          = false
      + virtual_network_name         = "vnet-spoke1"
    }

  # module.hub_and_2_spokes.azurerm_virtual_network_peering.peering_object["peering-from-spoke2-to-hub"] will be created
  + resource "azurerm_virtual_network_peering" "peering_object" {
      + allow_forwarded_traffic      = true
      + allow_gateway_transit        = false
      + allow_virtual_network_access = true
      + id                           = (known after apply)
      + name                         = "peering-from-spoke2-to-hub"
      + remote_virtual_network_id    = (known after apply)
      + resource_group_name          = "rg-spoke2"
      + use_remote_gateways          = false
      + virtual_network_name         = "vnet-spoke2"
    }
```

[Back to the Examples](#examples)
### (2) Simple hub-spoke and ready for Bastion
Please pay close attention to the comments within the code-snippet below

```hcl
module "hub_and_1_spoke_custom_subnets" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=main"
  //We want to deploy a hub with 1 subnet with a custom "name" So that its a valid Bastion subnet
  //We want to deploy 1 spoke, with 1 subnet and a custom "address_prefix" Which will consume the entire default address space provided to the spoke vnet
  typology_object = {
    
    hub_object = {
      network = {
        
        subnet_objects = [
          {
            name = "AzureBastionSubnet"
            use_last_subnet = true //We want to take the last possible CIDR of /26 from the /24 block provided to the vnet
            //We do NOT need to define more, module always defaults to subnets CIDR /26
          }
        ]
      }
    }

    spoke_objects = [
      {
        network = {
          subnet_objects = [
            {
               name = "vm-subnet"
               address_prefix = ["10.0.1.0/24"] //Because we use default values, the CIDR block available will be ["10.0.0.0/16"] and all vnets defaults to /24 and subnets to /26
               //Also, the module automatically divides all available CIDR blocks between vnets, where the hub will ALWAYS recieve the first address space available, then spokes are +1
            }
          ]
        }      
      }
    ]
  }
}

//TF Plan output:
Plan: 8 to add, 0 to change, 0 to destroy.
Terraform will perform the following actions:

  # module.hub_and_2_spokes_custom_subnets.azurerm_resource_group.rg_object["rg-hub"] will be created
  + resource "azurerm_resource_group" "rg_object" {
      + id       = (known after apply)
      + location = "westeurope"
      + name     = "rg-hub"
    }

  # module.hub_and_2_spokes_custom_subnets.azurerm_resource_group.rg_object["rg-spoke1"] will be created
  + resource "azurerm_resource_group" "rg_object" {
      + id       = (known after apply)
      + location = "westeurope"
      + name     = "rg-spoke1"
    }

  # module.hub_and_2_spokes_custom_subnets.azurerm_subnet.subnet_object["AzureBastionSubnet"] will be created
  + resource "azurerm_subnet" "subnet_object" {
      + address_prefixes                               = [
          + "10.0.0.192/26",
        ]
      + default_outbound_access_enabled                = true
      + enforce_private_link_endpoint_network_policies = (known after apply)
      + enforce_private_link_service_network_policies  = (known after apply)
      + id                                             = (known after apply)
      + name                                           = "AzureBastionSubnet"
      + private_endpoint_network_policies              = (known after apply)
      + private_endpoint_network_policies_enabled      = (known after apply)
      + private_link_service_network_policies_enabled  = (known after apply)
      + resource_group_name                            = "rg-hub"
      + virtual_network_name                           = "vnet-hub"
    }

  # module.hub_and_2_spokes_custom_subnets.azurerm_subnet.subnet_object["vm-subnet"] will be created
  + resource "azurerm_subnet" "subnet_object" {
      + address_prefixes                               = [
          + "10.0.1.0/24",
        ]
      + default_outbound_access_enabled                = true
      + enforce_private_link_endpoint_network_policies = (known after apply)
      + enforce_private_link_service_network_policies  = (known after apply)
      + id                                             = (known after apply)
      + name                                           = "vm-subnet"
      + private_endpoint_network_policies              = (known after apply)
      + private_endpoint_network_policies_enabled      = (known after apply)
      + private_link_service_network_policies_enabled  = (known after apply)
      + resource_group_name                            = "rg-spoke1"
      + virtual_network_name                           = "vnet-spoke1"
    }

  # module.hub_and_2_spokes_custom_subnets.azurerm_virtual_network.vnet_object["vnet-hub"] will be created
  + resource "azurerm_virtual_network" "vnet_object" {
      + address_space       = [
          + "10.0.0.0/24",
        ]
      + dns_servers         = (known after apply)
      + guid                = (known after apply)
      + id                  = (known after apply)
      + location            = "westeurope"
      + name                = "vnet-hub"
      + resource_group_name = "rg-hub"
      + subnet              = (known after apply)
    }

  # module.hub_and_2_spokes_custom_subnets.azurerm_virtual_network.vnet_object["vnet-spoke1"] will be created
  + resource "azurerm_virtual_network" "vnet_object" {
      + address_space       = [
          + "10.0.1.0/24",
        ]
      + dns_servers         = (known after apply)
      + guid                = (known after apply)
      + id                  = (known after apply)
      + location            = "westeurope"
      + name                = "vnet-spoke1"
      + resource_group_name = "rg-spoke1"
      + subnet              = (known after apply)
    }

  # module.hub_and_2_spokes_custom_subnets.azurerm_virtual_network_peering.peering_object["peering-from-hub-to-spoke1"] will be created
  + resource "azurerm_virtual_network_peering" "peering_object" {
      + allow_forwarded_traffic      = true
      + allow_gateway_transit        = true
      + allow_virtual_network_access = true
      + id                           = (known after apply)
      + name                         = "peering-from-hub-to-spoke1"
      + remote_virtual_network_id    = (known after apply)
      + resource_group_name          = "rg-hub"
      + use_remote_gateways          = false
      + virtual_network_name         = "vnet-hub"
    }

  # module.hub_and_2_spokes_custom_subnets.azurerm_virtual_network_peering.peering_object["peering-from-spoke1-to-hub"] will be created
  + resource "azurerm_virtual_network_peering" "peering_object" {
      + allow_forwarded_traffic      = true
      + allow_gateway_transit        = false
      + allow_virtual_network_access = true
      + id                           = (known after apply)
      + name                         = "peering-from-spoke1-to-hub"
      + remote_virtual_network_id    = (known after apply)
      + resource_group_name          = "rg-spoke1"
      + use_remote_gateways          = false
      + virtual_network_name         = "vnet-spoke1"
    }
```

[Back to the Examples](#examples)
### (3) Using the subnet delegation filter attribute called service_name_pattern
```hcl
module "using_subnet_delegation" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=main"
  //We want to deploy a hub with 0 subnets and default settings
  //We want to deploy 1 spoke, with 1 subnet which must be delegated to server farms
  typology_object = {
    
    hub_object = {
      network = {
        
      }
    }

    spoke_objects = [
      {
        network = {
          subnet_objects = [
            {
               name = "app-services-subnet"
               use_last_subnet = true
        
               delegation = [
                 {
                    name = "delegation-by-terraform"
                    service_name_pattern = "Web/server" //Make sure the pattern is "close enough" To the right delegation such that the module does NOT try to add conflicting delegations
                    //E.g. typing pattern "Web" Will both create a delegation for "Microsoft.Web/Hosting" AND "Microsoft.Web/server" which is not possible. By adding "Web/server" We secure only 1 of the delegations
                    //For other patterns, please see the buttom of this code snippet
                 }
               ]
            }
          ]
        }      
      }
    ]
  }
}

Plan: 7 to add, 0 to change, 0 to destroy.
Terraform will perform the following actions:
Terraform will perform the following actions:

  # module.using_subnet_delegation.azurerm_resource_group.rg_object["rg-hub"] will be created
  + resource "azurerm_resource_group" "rg_object" {
      + id       = (known after apply)
      + location = "westeurope"
      + name     = "rg-hub"
    }

  # module.using_subnet_delegation.azurerm_resource_group.rg_object["rg-spoke1"] will be created
  + resource "azurerm_resource_group" "rg_object" {
      + id       = (known after apply)
      + location = "westeurope"
      + name     = "rg-spoke1"
    }

  # module.using_subnet_delegation.azurerm_subnet.subnet_object["app-services-subnet"] will be created
  + resource "azurerm_subnet" "subnet_object" {
      + address_prefixes                               = [
          + "10.0.1.192/26",
        ]
      + default_outbound_access_enabled                = true
      + enforce_private_link_endpoint_network_policies = (known after apply)
      + enforce_private_link_service_network_policies  = (known after apply)
      + id                                             = (known after apply)
      + name                                           = "app-services-subnet"
      + private_endpoint_network_policies              = (known after apply)
      + private_endpoint_network_policies_enabled      = (known after apply)
      + private_link_service_network_policies_enabled  = (known after apply)
      + resource_group_name                            = "rg-spoke1"
      + virtual_network_name                           = "vnet-spoke1"

      + delegation {
          + name = "Web/serverFarms"

          + service_delegation {
              + actions = [
                  + "Microsoft.Network/virtualNetworks/subnets/action",
                ]
              + name    = "Microsoft.Web/serverFarms"
            }
        }
    }

  # module.using_subnet_delegation.azurerm_virtual_network.vnet_object["vnet-hub"] will be created
  + resource "azurerm_virtual_network" "vnet_object" {
      + address_space       = [
          + "10.0.0.0/24",
        ]
      + dns_servers         = (known after apply)
      + guid                = (known after apply)
      + id                  = (known after apply)
      + location            = "westeurope"
      + name                = "vnet-hub"
      + resource_group_name = "rg-hub"
      + subnet              = (known after apply)
    }

  # module.using_subnet_delegation.azurerm_virtual_network.vnet_object["vnet-spoke1"] will be created
  + resource "azurerm_virtual_network" "vnet_object" {
      + address_space       = [
          + "10.0.1.0/24",
        ]
      + dns_servers         = (known after apply)
      + guid                = (known after apply)
      + id                  = (known after apply)
      + location            = "westeurope"
      + name                = "vnet-spoke1"
      + resource_group_name = "rg-spoke1"
      + subnet              = (known after apply)
    }

  # module.using_subnet_delegation.azurerm_virtual_network_peering.peering_object["peering-from-hub-to-spoke1"] will be created
  + resource "azurerm_virtual_network_peering" "peering_object" {
      + allow_forwarded_traffic      = true
      + allow_gateway_transit        = true
      + allow_virtual_network_access = true
      + id                           = (known after apply)
      + name                         = "peering-from-hub-to-spoke1"
      + remote_virtual_network_id    = (known after apply)
      + resource_group_name          = "rg-hub"
      + use_remote_gateways          = false
      + virtual_network_name         = "vnet-hub"
    }

  # module.using_subnet_delegation.azurerm_virtual_network_peering.peering_object["peering-from-spoke1-to-hub"] will be created
  + resource "azurerm_virtual_network_peering" "peering_object" {
      + allow_forwarded_traffic      = true
      + allow_gateway_transit        = false
      + allow_virtual_network_access = true
      + id                           = (known after apply)
      + name                         = "peering-from-spoke1-to-hub"
      + remote_virtual_network_id    = (known after apply)
      + resource_group_name          = "rg-spoke1"
      + use_remote_gateways          = false
      + virtual_network_name         = "vnet-spoke1"
    }
  
  //Notice how the delegation completes simply by using pattern "Web/server" Which finds the delegation "Web/serverFarms"
  //The pattern could even be "Web/s" In the case above, but in terms of readabillity in a terraform script, the more descriptive pattern makes sense
  //It also adds the underlying actions for set delegation

  //More pattern values:
  // "Fabric", "Logic", "Batch", "PostgreSQL" And so many more - The entire list can be found in the local variable called "subnet_list_of_delegations" of the source code, link below
  ```

[Source code of the module](https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-hub-spoke/azurerm-hub-spoke.tf){:target="_blank"}

[Back to the Examples](#examples)
### Advanced examples - Seperated on topics
1. [Define custom vnet, subnet, bastion and both nic and public ip directly on a windows vm object](#1-define-custom-vnet-subnet-bastion-and-both-nic-and-public-ip-directly-on-a-windows-vm-object)
2. [Use of default settings combined with specialized vm configurations on multiple vms](#2-use-of-default-settings-combined-with-specialized-vm-configurations-on-multiple-vms)

### (1) Define custom vnet, subnet, bastion and both nic and public ip directly on a windows vm object
```hcl
module "custom_advanced_settings" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=1.3.0"

  rg_name = "custom-advanced-settings-rg"

  //Windows 10 with a custom public ip and NIC configurations
  vm_windows_objects = [
    {
      name = "win10"
      os_name = "windows10"

      public_ip = {
        name = "vm-custom-pip"
        allocation_method = "Dynamic"
        sku = "Basic"
        
        tags = {
          "environment" = "prod"
        }
      }

      nic = {
        name = "vm-custom-nic"
        dns_servers = ["8.8.8.8", "8.8.4.4"] //Google DNS
        enable_ip_forwarding = true
        
        ip_configuration = {
          name = "ip-config"
          private_ip_address_version = "IPv4"
          private_ip_address_allocation = "Static"
          private_ip_address = "10.0.0.5" //First possible address in the subnet we are deploying, as Azure takes the first 4 and last 1
        }

        tags = {
          "vm_name" = "win10"
        }
      }
    }
  ]

  vnet_object = {
    name = "custom-with-bastion-vnet"
    address_space = ["10.0.0.0/20"]
  }

  subnet_objects = [
    {
      //Name wont matter as it will always be forced to be "AzureBastionSubnet" because we have defined that we also want a bastion host
      address_prefixes = ["10.0.10.0/26"]
    },
    {
      name = "custom-vm-subnet"
      address_prefixes = ["10.0.0.0/24"]

      tags = {
        "environment" = "prod"
      }
    }
  ]

  bastion_object = {
    name = "custom-bastion" //must contain 'bastion'
    copy_paste_enabled = true
    file_copy_enabled = true
    sku = "Standard"
    scale_units = 5

    tags = {
      "environment" = "mgmt"
    }
  }
}

output "custom_advanced_settings" {
  value = module.custom_advanced_settings.summary_object
}

//Sample output
/*

*/
```
How it looks in Azure:

<img src="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-vm-bundle/pictures/8th-vm-black.png" />

[Back to the Examples](#advanced-examples---seperated-on-topics)
### (2) Use of default settings combined with specialized vm configurations on multiple vms
```hcl
module "custom_combined_with_default" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=1.3.0"

  rg_id = module.custom_advanced_settings.rg_object.id

  env_name = "prd" //prod, pd, and so on will indicate prod
  create_nsg = true
  create_public_ip = true //Will create a default public ip for each vm that does not have a specific public ip configuration set
  create_diagnostic_settings = true //Will create a default storage account that will be used by any vm with NO specific configuration set
  create_kv_for_vms = true //Will deploy keyvault + role assignment + secrets
  
  vm_linux_objects = [
    {
      name = "advanced-linux-redhat"
      os_name = "redhat"
      computer_name = "redhat"
      secure_boot_enabled = true

      os_disk = {
        name = "advanced-os-disk-redhat"
        caching = "ReadWrite"
        disk_size_gb = 512
        security_encryption_type = "asdasd"
        write_accelerator_enabled = true
        storage_account_type = "LRS"
      }

      admin_ssh_key = [
        {
          public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDjm7vUE6KhuZN3yWT+JirtSI62YsNyywvf6//IjTVQq/SLLfybSDerV9LsyHG7VaqAGqLGLfjwGDdGaSB++Tm9qfWne5oh0cS2wscHoCzzt1/3pBd8C1cq9GmWnVo5rAdHnRp/XUvVFortwR0DnIOvVnMJxK1mpnnHwLdqWmyb7msZhizc6T+ipzN2V7oYY01gbndsn0+ZYkBSWz22eEZoMRDUdgiE+ZeMnCRZLSMxIDSK+6cxaE7L+MFJU45KMPcvdD3ZM/WKiZl2knNbdJbuytOESyWgDxfnDMVO9YztH3sHRlIf1a/COfc7sKgQH0vXFf9GU0Uzf24pW9D9OdlJ"
          username = "redhat"
        }
      ]

      boot_diagnostics = {
        storage_account = {
          name = "customstorage121das"
          access_tier = "Hot"
          public_network_access_enabled = false
          account_replication_type = "LRS"

          network_rules = {
            //By simply adding the block, the module will create a rule allowing the vm subnet to access the storage account
          }
        }
      }

      nic = {
        name = "advanced-vm-nic" //Name must contain 'vm'
        enable_ip_forwarding = true

        ip_configuration = {
          name = "advanced-config"
          private_ip_address_version = "IPv4"
          private_ip_address = "10.0.0.5"
          private_ip_address_allocation = "Static"
        }
      }

      public_ip = {
        name = "advanced-vm-pip"
        sku = "Standard"
        allocation_method = "static"
      }

      termination_notification = {
        enabled = true
        timeout = "PT10M"
      }
    },
    {
      name = "custom-sku-ubuntu"
      os_name = "ubuntu"

      source_image_reference = {
        offer = "UbuntuServer"
        publisher = "Canonical"
        sku = "16.04.0-LTS"
        version = "16.04.202109280"
      }
    }
  ]

  vm_windows_objects = [
    {
      name = "Server2016-vm01"
      os_name = "SERVER2016"
    }
  ]
}


output "custom_combined_with_default" {
  value = module.custom_combined_with_default
}

[Back to the Examples](#advanced-examples---seperated-on-topics)