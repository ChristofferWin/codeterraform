# Azure Hub-Spoke Terraform module

## Table of Contents

1. [Description](#description)
2. [Prerequisites](#prerequisites)
3. [Getting Started](#getting-started)
4. [Versions](#versions)
5. [Parameters](#parameters)
6. [Return Values](#return-values)
7. [Examples version 1.0.0 SAME SUB ONLY](#examples-same-sub-only)
8. [Examples version 2.0.0 DIFFERENT SUBS ONLY](#examples-different-subs-only)
9. [Known errors](#known-errors) 

## Description

Welcome to the Azure Hub-Spoke Terraform module. This module is designed to make the deployment of any hub-spoke network topology as easy as 1-2-3. The module is built on a concept of a single input variable called 'topology object', which can then contain a huge subset of custom configurations. The module supports name injection, automatic subnetting, Point-to-Site VPN, firewall, routing, and much more! Because it's built for Azure, it uses the architectural design from the Microsoft CAF concepts, which can be read more about at <a href="https://learn.microsoft.com/en-us/azure/architecture/networking/architecture/hub-spoke?tabs=cli">Hub-Spoke topology</a>

<b>ATTENTION:</b> The module can run in 2 different modes: 
1. Deploy both hub & spokes to the same Azure Subscription
2. Deploy the hub in 1 subscription and 1 to MANY spokes in 1 Azure subscription
   1. To deploy spokes in different Subscriptions, simply define more module calls. See below for details!
   2. ATTENTION: When running an isolated x amount of spoke module calls - Make sure to make use of naming attributes to make the spoke names unique between each other!

The module behaves a little different & has different requirements depending on the mode that is used - The mode is determined by the VERSION of the module that is called. See the [Getting Started](#getting-started) for details on that.

Also for specific release notes for the newest release, please see the <a href="https://github.com/ChristofferWin/codeterraform/releases">Releases page</a> for more details.

I plan to release multiple blog posts about the use of this module in different scenarios over on <a href="https://codeterraform.com/blog">Codeterraform</a>, so stay tuned!

I really appriciate you - I would really appriciate any criticism / feedback, possible feature improvements and overall good karma :)

Just below here, two different visual examples of types of hub-spokes can be seen. Both can be directly deployed with the module.

<b>Example 1: Deployment of a simple hub-spoke</b>
</br>
</br>
<img src="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/Graphic%20material/DrawIO/Simple-hub-spoke-Simple-Hub-Spoke.png"/>
</br>
</br>
</br>
<b>Example 2: Deployment of an advanced hub-spoke (As of version 2.0.0-hub-spoke the entire topology in DIFFERENT SUBS)</b>
</br>
</br>
<img src="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/Graphic%20material/DrawIO/Simple-hub-spoke-Complex%20Hub-Spoke.drawio.png"/>
</br>

[Back to the top](#table-of-contents)
## Prerequisites

Before using this module, make sure you have the following:
- Active Azure Subscription
  - Must be able to WRITE to the subscription
- Installed terraform (download [here](https://www.terraform.io/downloads.html))
- Azure CLI installed for authentication (download [here](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli))
- Make sure that the user context used by Terraform has WRITE access to all Azure Subscriptions

[Back to the top](#table-of-contents)
## Getting Started
Remember to have read the chapter [Prerequisites](#prerequisites) before getting started.

1. Create a new terraform script file in any folder

2. Login to Azure with an active subscription using az cli
```powershell
az login //Web browser interactive prompt.
```
3. Define the module definition (VERSION 2.0.0 used = MULTIPLE SUBS)
```hcl
//We need to define a minimum of 2 providers as the module requires this
//In this example we will simply use the default azurerm provider + 1 alias provider

provider "azurerm" { //Because no alias is defined, this will be the default
  features {}
  //We can add a subscription_id here explicity if we want to
  //But for this example we simply use whatever subscription az cli logged into in-line from step 3
}

provider "azurerm" {
  features {}
  alias = "hubsub" //Our custom name for the hub sub context
  subscription_id = "<a valid subscription>"
}

//We define a module call to ONLY create the hub (Spokes can still be created here though)
//We will simply use default values, look through the input variables & examples for more advanced setups
//Even though we do NOT create any direct spokes in the hub module call, we still ALWAYS must parse the spoke provider
module "our_hub" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=2.0.0-hub-spoke"

  topology_object = {

    hub_object = {
      network = {
        //We use all default values
      }
    }
  }

  //Inside the module call to create the hub, both providers must be parsed, otherwise Terraform will fail
  providers = {
    azurerm.hub = azurerm.hubsub //The azurern.hub part is the specific alias name from the module. Parse the hub provider here
    azurerm.spoke = azurerm //The azurerm.spoke part is the alias name from the module. Parse the spoke provider, in this case we will use our default provider
  }
}

//Create 1 default spoke and connect it to the hub above
//ATTENTION: The hub MUST be deployed before running the deployment of ANY spoke, otherwise Terraform will fail

module "only_simple_spoke" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=2.0.0-hub-spoke"

  topology_object = {
    //No hub will be deployed, but we must as a MINIMUM define the below to tell the module to connect to the specific HUB
    hub_object = {
      network = {
        vnet_resource_id = values(module.our_hub.vnet_return_objects)[0].id //We must parse the deployed HUB vnet resource ID
      }
    }

    spoke_objects = [
      {
        network = {
          subnet_objects = [
            {
              //Nothing defined = Using only default values
            }
          ]
        }
      }
    ]
  }
 
 providers = {
    azurerm.hub = azurerm.hubsub //The azurern.hub part is the specific alias name from the module. Parse the hub provider here
    azurerm.spoke = azurerm //The azurerm.spoke part is the alias name from the module. Parse the spoke provider, in this case we will use our default provider
  }
}
```
4. Define the module definition (VERSION 1.0.0 used = ONLY SAME SUB FOR ALL COMPONENTS)
```hcl
module "simple_hub_spoke" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=1.0.0-hub-spoke" //Always use a specific version of the module
  
  topology object = {
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

6. There is a ton more to explore with the module, see the [Examples version 2.0.0 DIFFERENT SUBS ONLY](#examples-different-subs-only) for details

[Back to the top](#table-of-contents)
## Versions
The table below outlines the compatibility of the module:

Please take note of the 'Module version' among the provider utilized by the module. Keep in mind that there WILL be a required minimum version, and this requirement can vary with each module version.

<b>"Module version" '1.0.0-hub-spoke' & '2.0.0-hub-spoke' requires the following provider versions:</b>

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
For assisting in understanding the actual structure of the only input variable "topology object" Please see below code:
```hcl
module "show_case_object" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=1.0.0-hub-spoke"
  topology object = { //The "root" is an OBJECT
    //Many different overall settings for the entire deployment can be set here. See below the code snippet for details.

    hub_object = { //The "hub_object" is an OBJECT - Object path is then <topology object.hub_object>
      //Less but specific attributes can be set for the hub here. See below the code snippet for details.

      network = { //The object "network" is an OBJECT - Object path is then <topology object.hub_object.network>
        //Multiple different attributes with relevance to network can be set for the hub here. See below the code snippet for details.

        vpn = { //The object "vpn" is an OBJECT - Object path is then <topology object.hub_object.network.vpn>
          //Specific attributes related to configuring a Point-2-Site VPN. See below the code snippet for details.
        }

        firewall = { //The object "firewall" is an OBJECT - Object path is then <topology object.hub_object.network.firewall>
          //Specific attributes related to configuring an Azure Firewall. See below the code snippet for details.
        }

        subnet_objects = [ //The list of objects "subnet_objects" is a LIST OF OBJECT - Object path is then <topology object.hub_object.network.subnet_objects[index]>
          {
            //For each {} block, define specific attributes related to Azure subnets. See below the code snippet for details.
          }
        ]
      }
    }

    spoke_objects = [ //The list of objects "spoke_objects" is a LIST OF OBJECT - Object path is then <topology object.spoke_objects[index]>
      {
        //For each {} block, many spokes can be deployed. Minimum 1. See below the code snippet for details.
      
        network = { //The object "network" is an OBJECT - Object path is then <topology object.spoke_objects[index].network>
          //Multiple different attributes with relevance to network can be set for each spoke here. See below the code snippet for details.

          subnet_objects = [ //The list of objects "subnet_objects" is a LIST OF OBJECT - Object path is then <topology object.spoke_objects[index].network.subnet_objects[index]>
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

### Attributes on the "top" Level of the "topology object"
1. project_name = (optional) A string defining the name of the project / landing zone. Will be injected into the overall resource names. OBS. Using this variable requires both either "name_prefix" OR "name_suffix" AND "env_name" to be provided as well

2. location = (optional) A string defining the location of ALL resources deployed (overwrites ANY lower set location)

3. name_prefix = (optional) A string to inject a prefix into all resource names - This variable makes it so names follow a naming standard: \<resource abbreviation>\-<name_prefix>\-\<Identier, either "hub" or "spoke">

4. name_suffix = (optional) A string to inject a suffix into all resource names - This variable also makes names follow a naming standard: <Identifier, either "hub" or "spoke">\-\<name_suffix>\-\<resource abbreviation>

5. env_name = (optional) A string defining an environment name to inject into all resource names. OBS. Using this variable requires both either "name_prefix" OR "name_suffix" AND "project_name" To be provided as well

6. dns_servers = (optional) A list of strings defining DNS server IP  to set for ALL vnets in the topology (overwrites ANY lower set DNS servers)

7. tags = (optional) A map of strings defining any tags to set on ALL vnets and resource groups, VPN and Firewall (Any tags set lower will be appended to these tags set here)

8. subnets_cidr_notation = (optional) A string defining what specific subnet size that ALL subnets should have - Defaults to "/26"

Its possible to define VERY little attributes on the top level "topology object" See the [Examples](#examples-different-subs-only) For details

### Attributes on the "hub_object" level of the "topology object" (This is an object described as topology object.hub_object = {})
1. rg_name = (optional) A string defining the specific name of the hub resource group resource (Overwrites any name injection defined in the top level attributes)

2. location = (optional) A string defining the location of which to deploy the hub to (If the top level location is set, this will be overwritten)

3. tags = (optional) A map og strings defining any tags to set on the hub resources

4. network = (<b>required</b>) An object structured as: (Object can be left as {} which will cause the module to create a hub with 0 subnets)
    1. vnet_name = (optional) A string defining the name of the hub Azure Virtual Network resource (Overwrites any name injection defined in the top level attributes)

    2. vnet_cidr_notation = (optional) A string to be used in case you do NOT parse the attribute "address_spaces" The module will then instead use a base CIDR block of ["10.0.0.0/16] and use the attribute "vnet_cidr_notation" to subnet the "address_spaces" for the hub Azure Virtual Network resource. Must be parsed in the form of "/\<CIDR>" e.g "/24"

    3. address_spaces = (optional) A list of strings to be used in case you do NOT provide the attribute "vnet_cidr_notation" By providing a value for this attribute, you completely define the exact CIDR block for the hub Azure Virtual Network resourc
    
    4. vnet_spoke_address_spaces (SPECIFIC TO VERSION 2.0.0) (optional) A list of strings defining all the SPOKE Azure Virtual Networks address spaces. This attribute MUST be used in case the hub has a Azure Firewall defined as the address spaces will be parsed to the Firewall rules

    5. dns_servers = (optional) A list of strings defining DNS server IP addresses to set for the spoke Azure Virtual Network resource (Will be overwritten in case the attribute is set on the top level object)

    6. tags = (optional) A map og strings defining any tags to set on the hub vnets - Tags here will append to all other tags

    7. vnet_peering_allow_virtual_network_access = (optional) (NOT RECOMMENDED TO CHANGE) A bool used to disable whether the spoke vnet´s Azure Virtual machine resources can reach the hub

    8. vnet_peering_allow_forwarded_traffic = (optional) (NOT RECOMMENDED TO CHANGE) A bool used to disable whether the hub vnet can recieve forwarded traffic from the spoke vnet

    9. fw_resource_id = (SPECIFIC TO VERSION 2.0.0) (optional) The string resource id of the hub Azure Firewall - MUST be defined in the module call for where SPOKES are created IF the HUB defined also has a Azure Firewall defined.

    10. fw_private_ip = (SPECIFIC TO VERSION 2.0.0) (optional) The string private IP of the hub Azure Firewall - Can be used INSTEAD OF attribute "fw_resource_id" This attribute stops the module from having to retrieve the Azure Firewall from the hub to automatically retrieve the Azure Firewall private ip

    10. vpn = (optional) An object structured as:
       
       1. gw_name = (optional) A string to define the custom name of the Azure Virtual Network Gateway resource (Overwrites any naming injection defined in the top level object)

        2. address_space = (optional) A list of strings defining the CIDR block to be used by the Point-2-Site VPN connections, for the DHCP scope

        3. gw_sku = (optional) (NOT RECOMMENDED TO CHANGE) A string used to define the SKU for the Azure Virtual Gateway resource. Defaults to "VpnGw2"

        4. pip_name = (optional) A string defining the custom name of the Azure Public IP to be used on the VPN (Overwrites any naming injection defined in the top level object)

        5. tags = (optional) A map of strings defining any tags to set for the VPN - Since tags can be set on many different levels see the [Using tags at different levels of the topology object](#4-using-tags-at-different-levels-of-the-topology-object) example for more details on tags
    
    12. firewall = (optional) An object structured as:
        
        1. name = (optional) A string to define the custom name of the Azure Firewall resource (Overwrites any naming injection defined in the top level object)

        2. sku_tier = (optional) A string defining the SKU tier of the Azure Firewall resource. Defaults to "Standard"

        3. threat_intel_mode = (optional) A bool defining whether the mode of the automatic detection shall be set to "Deny" Mode.

        4. pip_name = (optional) A string defining the custom name of the Azure Public IP to be used on the Firewall (Overwrites any naming injection defined in the top level object)

        5. log_name = (optional) A string defining the custom name of the Azure Log Analytics workspace resource (Overwrites any naming injection defined in the top level object)

        6. log_diag_name = (optional) A string defining the custom name of the Azure Diagnostic settings name, defaults to "fw-logs-to-log-analytics"

        7. log_daily_quota_gb = (optional) A number defining the daily quota in GB that can be injested into the Azure Log Analytics workspace. Defaults to -1 which means NO limit

        8. no_logs = (optional) A bool to determine whether the module shall NOT create an Azure Log Analytics workspace and Azure Diagnostic settings for the Azure Firewall. Pr. default both resources will be created IF the Firewall is also created

        9. no_internet = (optional) A bool to determine whether the specific Firewall Rule "ALLOW INTERNET FROM SPOKES" shall NOT be deployed. OBS. Using this bool is overwritten by the Bool "no_rules"

        10. no_rules = (optional) A bool to determine whether the module shall NOT create Azure Firewall rules. Pr. default Azure Firewall network rules will be created IF the Firewall is also created. (The specific rules applied can be seen via [Advanced spoke](#description))

        11. tags = (optional) A map of strings defining any tags to set for the Firewall - Since tags can be set on many different levels see the [Using tags at different levels of the topology object](#4-using-tags-at-different-levels-of-the-topology-object) example for more details on tags
    
    13. subnet_objects = (optional) A list og objects structured as:
        
        1. name = (optional) A string defining the custom name of the Azure Subnet (Overwrites any naming injection defined in the top level object). If you include the segnemt: "mgmt" OR "management" the subnet will be used as the ONLY subnet to be allowed access to rdp / ssh for spoke vms via the firewall rule, if no subnet name is custom and includes the segnment, the entire vnet hub address space will be used as the source address for the firewall rule (This ONLY has impact if the firewall is also created) See the [Use a specific subnet as the ONLY allowed subnet to use RDP and SSH to spoke vms](#3-use-a-specific-subnet-as-the-only-allowed-subnet-to-use-rdp-and-ssh-to-spoke-vms)
        
        2. use_first_subnet = (optional) A bool to use in case the attribute "address_prefix" is NOT used - Tells the module to create a subnet CIDR from the START of the CIDR block used in the deployment. See the [Examples](#examples-different-subs-only) for more details

        3. use_last_subnet = (optional) A bool to use in case the attribute "address_prefix" is NOT used - Tells the module to create a subnet CIDR from the END of the CIDR block used in the deployment. See the [Examples](#examples-different-subs-only) for more details

        4. address_prefix = (optional) An address space specifically defined for the subnet. Its NOT recommended to define this manually in case the overall vnets "address_spaces" Attribute is NOT populated.

        5. service_endpoints = (optional) A string defining Microft Azure Service Endpoints to add to the subnet

        6. service_endpoint_policy_ids = (optional) A set of strings defining any Azure Service Endpoint policy id's to add to the subnet

        7. delegation = (optional) A list of objects structured as:
            1. name = optional(string) A custom name to add as the display name for the deletation added to the subnet
            2. service_name_pattern = optional(string) A string defining a pattern to match a specific Azure delegation for the subnet. For a showcasing of how to use the filter see the [How to easily deploy delegations](#3-Using-the-subnet-delegation-filter-attribute-called-service_name_pattern) for more details

Its possible to define VERY little attributes on the hub / spoke level of the "topology object" 
See the [Examples version 1.0.0 SAME SUB ONLY](#examples-same-sub-only) & [Examples version 2.0.0 DIFFERENT SUBS ONLY](#examples-different-subs-only) For details


### Attributes on the "spoke_objects" level of the "topology object" (This is a list of objects described as topology object.spoke_objects[index] = [{}])
1. network = (<b>required</b>) An object describing the network structure of the spoke (NOT REQUIRED IN VERSION 2.0.0 to allow ONLY creating a hub and then spokes seperately)
   1. same attributes can be set here, as for the "network" object under the hub
   2. subnet_objects = (<b>required</b>) A list of objects describing each subnet, at least 1 subnet must be created, which is different from the hub, where the attribute can even be null

See the [Examples version 1.0.0 SAME SUB ONLY](#examples-same-sub-only) OR [Examples version 2.0.0 DIFFERENT SUBS ONLY](#examples-different-subs-only) for more details

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

7. pip_return_objects = map of object containing all the same return attributes as the provider => <a href="https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/public_ip#attributes-reference">Azurerm Public IP</a>

8. log_return_object = object containing all the same return attributes as the provider => <a href="https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/log_analytics_workspace.html#attributes-reference">Azurerm Log Analytics workspace</a>

[Back to the top](#table-of-contents)
## Examples SAME SUB ONLY
<b>This section is split into 2 different sub sections:</b>

- <a href="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-hub-spoke/readme.md#simple-examples---separated-on-topics">Simple examples</a> = Meant to showcase how to deploy simple hub-spoke topologies
- <a href="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-hub-spoke/readme.md#advanced-examples---seperated-on-topics">Advanced examples</a> = Meant to showcase how to deploy advanced hub-spoke topologies


### Simple examples - Separated on topics
1. [Deploy a simple hub and 2 spokes with minimum config](#1-Deploy-a-simple-hub-and-2-spokes-with-minimum-config)
2. [Simple hub-spoke and ready for Bastion](#2-Simple-hub-spoke-and-ready-for-Bastion)
3. [Using the subnet delegation filter attribute called "service_name_pattern"](#3-Using-the-subnet-delegation-filter-attribute-called-service_name_pattern)
4. [Using tags at different levels of the topology object](#4-using-tags-at-different-levels-of-the-topology-object)

### (1) Deploy a simple hub and 2 spokes with minimum config
```hcl
module "hub_and_2_spokes" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=1.0.0-hub-spoke"
  //We want to deploy a hub with 0 subnets and default settings
  //We want to deploy 2 spokes, with 2 subnets in each
  topology object = {
    
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

[Back to the Examples](#examples-same-sub-only)
### (2) Simple hub-spoke and ready for Bastion
Please pay close attention to the comments within the code-snippet below

```hcl
module "hub_and_1_spoke_custom_subnets" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=1.0.0-hub-spoke"
  //We want to deploy a hub with 1 subnet with a custom "name" So that its a valid Bastion subnet
  //We want to deploy 1 spoke, with 1 subnet and a custom "address_prefix" Which will consume the entire default address space provided to the spoke vnet
  topology object = {
    
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

[Back to the Examples](#examples-same-sub-only)
### (3) Using the subnet delegation filter attribute called service_name_pattern
```hcl
module "using_subnet_delegation" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=1.0.0-hub-spoke"
  //We want to deploy a hub with 0 subnets and default settings
  //We want to deploy 1 spoke, with 1 subnet which must be delegated to server farms
  topology object = {
    
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

//TF Plan output:
Plan: 7 to add, 0 to change, 0 to destroy.
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

<a href="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-hub-spoke/azurerm-hub-spoke.tf" target="_blank">source code of the module</a>

[Back to the Examples](#examples-same-sub-only)

### (4) Using tags at different levels of the topology object
This example simply showcases all the possible levels of which to set tags in the "topology object"
All objects added is ONLY done so to make the code deployable - The important points are the tags themselves - Please notice the exact behaviour within the comments of the code-snippet

```hcl
 module "tags_at_all_possible_levels" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=1.0.0-hub-spoke"

  topology object = {
    tags = {
      "top-level-tags" = "tag1" #This tag will append to ALL RGs, VNETS, Firewall, VPN and Log space
    }

    hub_object = {
      tags = {
        "hub-rg-level-tags" = "tag2" #This tag will apply to ONLY the HUB RG - This tag will NOT append on VNETS or anything else within the HUB
      }

      network = {
        tags = {
          "hub-vnet-level-tags" = "tag3"
        }
        #No subnets to create, this is simply to showcase tags - But this code will STILL deploy
      }
    }

    spoke_objects = [
      {
        tags = {
          "spoke1-level-tags" = "tag4" #This tag will apply to ONLY the SPOKE1 RG - This tag will NOT append on VNETS or anything else within the spoke
        }

        network = {
          tags = {
            "spoke1-vnet-level-tags" = "tag5" #This tag will apply to ONLY the SPOKE1 VNET
          }

          subnet_objects = [
            {
              #The module requires a minimum of 1 subnet in 1 spoke to be created. This example only wants to showcase the function of the tags
              #Therefor this subnet is ONLY applied to make the code valid for the module to comsume
            }
          ]
        }
      }
    ]
  }
}

//TF Plan output: (Notice how all the resources have BOTH the top level tags AND EITHER the vnet or rg tags depending on the resource type ofc)
//In other words - If tags are defined under the root of "topology object" These will be inherited by almost all resource types
Plan: 7 to add, 0 to change, 0 to destroy.
```
[Back to the Examples](#examples-same-sub-only)

### Advanced examples - Seperated on topics
1. [Hub-spoke with both firewall and vpn](#1-Hub-spoke-with-both-firewall-and-vpn)
2. [Custom settings for peerings between the hub and the spokes](#2-custom-settings-for-peerings-between-the-hub-and-the-spokes)
3. [Use a specific subnet as the ONLY allowed subnet to use RDP and SSH to spoke vms](#3-Use-a-specific-subnet-as-the-only-allowed-subnet-to-use-rdp-and-ssh-to-spoke-vms)

### (1) Hub-spoke with both firewall and vpn
Deploy an advanced hub-spoke topology containing both an Azure Firewall and Azure Point-2-site VPN. Because we deploy the Firewall, route tables are also created. Please note that a lot more specific configuration can be achieved on the "vpn" And "firewall" Objects respectively - See the [Parameters](#parameters) Section for more details

```hcl
module "advanced_spoke_with_all_components" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=1.0.0-hub-spoke"
  //We want to use name injection on all resources + add a few custom names
  //We want to use top level attributes, to enforce location, a custom CIDR block for ALL vnets to use and more
  //We want to deploy a hub with 3 subnets, 1 for Bastion, 1 for the Firewall and 1 for the VPN
  //We want the Firewall to be the first subnet AND take first possible CIDR block available
  //We want to deploy the Point-2-Site VPN wih a custom address space for the VPN DHCP
  //We want to customize the firewall object
  //We want to deploy 2 spokes, each with 2 subnets, where we will use a mix of first possible CIDR block and last possible
  topology object = {
    name_suffix = "lab"
    project_name = "contoso"
    env_name = "prod" //Because the project name is defined, we must also define an env_name
    location = "westus" //Forcing the location of ALL resources to be set location
    address_spaces = ["172.16.0.0/20"] //Custom CIDR block to replace the default within the module of ["10.0.0.0/16"]
    dns_servers = ["8.8.8.8", "8.8.4.4"] //Forcing DNS to be google for ALL vnets

    hub_object = {
      network = {

        firewall = {
          //Instead of defining custom names for both the fw and pip, we let the attributes from the root object inject into the names
          threat_intel_mode = true //Overwrite the default behaviour of "Alert" When it comes to the Azure Firewall packet inspection to "Deny"
          log_daily_quota_gb = 5 //By default the Log Analytics workspace created does NOT have a limit - Here we Overwrite it to being 5gb
        }

        vpn = {
          address_space = ["192.168.0.0/24"] //Changing the default address space used by the Point-2-Site VPN, default is 172.16.99.0/24
        }

        subnet_objects = [
          {
            name = "AzureFirewallSubnet" //Overwrites anything defined in the above levels
            //use_first_subnet = true //If the subnet object definition is left as an empty object {}, the subnetting defaults to using the first available CIDR block
          },
          {
            name = "AzureBastionSubnet"
            use_last_subnet = true
          },
          {
            name = "GatewaySubnet"
            //use_first_subnet = true //If the subnet object definition is left as an empty object {}, the subnetting defaults to using the first available CIDR block
          }
        ]
      }
    }

    spoke_objects = [
      { #spoke 1
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
            },
            {

            }
          ]
        }      
      },
      { #spoke 2
        rg_name = "spoke-2-custom-name" #Will overwrite ALL naming injection from the top level attributes
        tags = {
          "environment" = "production"
        }

         network = {
           subnet_objects = [
             { #subnet_1_spoke_2
               name = "vm-subnet"
               address_prefix = ["172.16.2.0/26"] //Because we define a custom "address_spaces" In the top level object, we know the spoke 2 vnet will have CIDR block ["172.16.2.0/24"]
             },
             { #subnet_2_spoke_2
               use_last_subnet = true //This will not overlap with subnet one, as we manually defined it as the first possible /26 CIDR block and now we instead take the last possible block
               //The address_prefix will then automatically be calculated to value "10.0.2.128/26"
             }
           ]
         }
      }
    ]
  }
}

//TF Plan output (Only most interesting objects are shown):
```

[Back to the Examples](#advanced-examples---seperated-on-topics)
### (2) Custom settings for peerings between the hub and the spokes
Its possible to further secure what is allowed on the peering FROM the hub and TO the spokes
Only change such setting if your sure of the effect - It might stop connectivity from working

```hcl
module "advanced_spoke_with_all_components2" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=1.0.0-hub-spoke"
  //In this example we will use more custom names instead of naming injection
  //Custom peering settings - WARNING - This might stop traffic from flowing to and from the hub vnet
  //Adding tags - Tags append - Since we both defined them in the hub_object root and inside "network" Both tags will be added to the vnet
  //We define custom FW settings such that the module will NOT deploy log analytics, diagnostic settings or FW network rules
  //Because we define custom peering names, these will ONLY effect the peerings inside the hub - It will also use the same name twice and simply add +1 at the end of the name

  topology object = {
    hub_object = {
      rg_name = "custom-rg-hub"
      location = "northeurope"

      tags = {
        "custom" = "tag"
      }
        network = {
          vnet_name = "hub-custom-vnet"
          address_spaces = ["172.16.0.0/22"]
          dns_servers = ["1.1.1.1", "8.8.8.8"]
          vnet_peering_name = "custom-peering"
          vnet_peering_allow_virtual_network_access = false //Only effects the peerings from the HUB to SPOKES
          vnet_peering_allow_forwarded_traffic = false //Only effects the peerings from the HUB to SPOKES

          tags = {
            "custom2" = "tag"
          }

          vpn = {
            gw_name = "advanced-gw"
            address_space = ["192.168.0.0/24"]
            pip_name = "advanced-pip"
          }

          firewall = {
            name = "custom-fw"
            threat_intel_mode = true
            pip_name = "fw-custom-pip"
            no_logs = true //Will make the module NOT make log analytics workspace + diag settings for FW
            no_rules = true //Will make the module NOT make the 2 default FW rules as shown in advanced example 1
          }

          subnet_objects = [
            {
              name = "subnet1-customhub"
              address_prefix = ["172.16.0.0/27"]
            },
            {
              name = "subnet2-customhub"
              address_prefix = ["172.16.0.32/27"]
            }
          ]
        }
      }

      spoke_objects = [
        {
          rg_name = "spoke1-custom-rg"
          location = "eastus"
          
          tags = {
            "custom2" = "tag"
          }

          network = {
            subnet_objects = [
              {
                name = "subnet-custom1-spoke1"
                use_last_subnet = true
              },
              {
                name = "subnet-custom2-spoke1"
                use_last_subnet = true
              },
              {
                name = "AzureFirewallSubnet"
              },
              {
                name = "GatewaySubnet"
              }
            ]
          }
        },
        {
          rg_name = "spoke2-custom-rg"
          location = "westus"
          
          tags = {
            "custom1" = "tag"
          }

          network = {
            subnet_objects = [
              {
                name = "subnet-custom1-spoke2"
                use_first_subnet = true
              },
              {
                name = "subnet-custom2-spoke2"
                use_first_subnet = true
              }
            ]
          }
        }
      ]
    }
}

//TF Plan output (ALL resources are shown, to showcase that NO log space, diag settings and fw rules will be deployed):
Plan: 34 to add, 0 to change, 0 to destroy.
```
[Back to the Examples](#advanced-examples---seperated-on-topics)

### (3) Use a specific subnet as the ONLY allowed subnet to use RDP and SSH to spoke vms
Imagine a scenario where you want a topology setup with many different custom CIDR subnetting and naming settings

We want to have an Azure Firewall with no allowed internet access and we want to control the specific subnet of which is used as the source address for the firewall rule to allow rdp / ssh to spoke vms

Take the below example code snippet and please pay close attention to the comments

```hcl
module "control_subnet_used_for_fw_rule_rdp_ssh" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=1.0.0-hub-spoke"
  
  topology object = {
    name_suffix = "contoso"
    project_name = "security"
    location = "westus"
    subnets_cidr_notation = "/27" #Forcing ALL subnets who has not been assigned a specific prefix
    #Since we wont allow the spoke vms to use the internet, we must create some private DNS service
    #After the private DNS service is created, we can then ADD private DNS server addresses for the hub vnet by simply using the attribute "dns_servers"

    hub_object = {
      
      network = {
        vnet_name = "custom-vnet-name" #Ignoring ALL naming injection from the top level object
        address_spaces = ["192.168.0.0/22"] #Using a custom address space
        
        firewall = {
          #Simply using naming injection for the firewall's name
          no_internet = true #Stops the module from deploying the firwall rule that allows https / http / dns to the internet'
        }

        subnet_objects = [
             {
               name = "use-this-subnet-mgmt" #Because we have the segment "mgmt" In the subnet name, THIS SPECIFIC subnet will be used as the source address for the firewall rule to allow rdp and ssh to spoke vms
               address_prefix = ["192.168.0.0/26"] #Because we did NOT define an "address_spaces"
               #SO in this case, the overall vnet address space is ["192.168.0.0/22"] So if we want to use a custom "address_prefix" We MUST be within this address space
             },
             {
               name = "AzureFirewallSubnet" #Since we deploy the firewall, we must define the firewall subnet
               address_prefix = ["192.168.0.64/26"] #Since we forced all subnets to default CIDR to /27, we must manually create a address prefix of /26, which we do by taking the 2nd possible CIDR of /26 from the vnet address space of ["192.168.0.0/22"]
             },
             {
               name = "custom-subnet2" #Ignoring naming injection
               use_last_subnet = true #Will then use the LAST CIDR block of /27 available from the address space ["192.168.0.0/22"]
               #OBS. Because we MANUALLY defined subnet 1 and 2's address prefix's the module cannot know which CIDR blocks from the original of /22 has been taken
               #Therefor make sure you know whether any subnet address prefixes begin to collide
               #From the 2 subnets defined above´s example, since we manually take the first possible CIDR blok of the "/22" for subnet1 and the 2nd for subnet2, we can still begin to take subnets from the other end of the CIDR block using the bool "use_last_subnet"
               #If we instead used the switch "use_first_subnet" For subnet 2's address prefix, the module would make a collision causing the deployment to fail
             }
           ]
      }
    }

    spoke_objects = [ #We MUST define at least 1 spoke
      {
        rg_name = "custom-spoke-rg" #Ignoring naming injection for the spoke rg
        
        network = {
           #We will use the default address space CIDR block of [10.0.0.0/16]
           vnet_name = "test-vnet-spoke"

           subnet_objects = [
             {
               #Using naming injection for the subnet1's name
               address_prefix = ["10.0.1.0/26"] #Because we did NOT define an "address_spaces" For this spoke, the vnet address space will by default be /24 and the 3rd octect will be the spoke number
               #SO in this case, the overall vnet address space is [10.0.1.0/24] So if we want to use a custom "address_prefix" We MUST be within this address space
             },
             {
               name = "custom-subnet2" #Ignoring naming injection
               use_last_subnet = true #Will then use the LAST CIDR block of /27 available from the address space ["10.0.1.0/24"]
               #OBS. Because we MANUALLY defined subnet 1's address prefix the module cannot know which CIDR blocks from the original of /24 has been taken
               #Therefor make sure you know whether any subnet address prefixes begin to collide
               #From the 2 subnets defined above´s example, since we manually take the first possible CIDR blok of the "/24" for subnet1, we can still begin to take subnets from the other end of the CIDR block using the bool "use_last_subnet"
               #If we instead used the switch "use_first_subnet" For subnet 2's address prefix, the module would make a collision causing the deployment to fail
             },
             {
              #Subnet3 simply using naming injection for the name
              use_last_subnet = true #Using the 2nd last subnet of CIDR "/27" From the address space ["10.0.1.0/24"]
              #The module knows that subnet2 already has the ABSOLUT last subnet, therefor this subnet will take the 2nd last
             }
           ]
        }
      }
    ]
  }
}

//TF plan output (Only most interesting objects are shown)
Plan: 23 to add, 0 to change, 0 to destroy.
//All subnets will use /27 as the attribute "subnet_cidr_notation" Is set in the top level object, but the custom address prefixes will ignore this

 module.control_subnet_used_for_fw_rule_rdp_ssh.azurerm_firewall_network_rule_collection.fw_rule_object["Allow-RDP-SSH-FROM-MGMT-TO-SPOKES"] will be created
  + resource "azurerm_firewall_network_rule_collection" "fw_rule_object" {
      + action              = "Allow"
      + azure_firewall_name = "hub-contoso-fw"
      + id                  = (known after apply)
      + name                = "Allow-RDP-SSH-FROM-MGMT-TO-SPOKES"
      + priority            = 200
      + resource_group_name = "hub-contoso-rg"

      + rule {
          + destination_addresses = [
              + "10.0.1.0/24",
            ]
          + destination_ports     = [
              + "22",
              + "3389",
            ]
          + name                  = "Allow-RDP-SSH-FROM-MGMT-TO-SPOKES"
          + protocols             = [
              + "TCP",
            ]
          + source_addresses      = [
              + "192.168.0.0/26", => (Set by us) //Only the specific subnet "use-this-subnet-mgmt" Because of the "mgmt" But its NOT case sensetive, and it can also be "management" 
            ]
        }
    }

//The internet rule is NOT created - Because we set the flag "no_internet" Under the "firewall" Object
```
[Back to the Examples](#advanced-examples---seperated-on-topics)

[Back to the top](#table-of-contents)

## Examples DIFFERENT SUBS ONLY
This section contains only advanced examples as using the module on different subscriptions by seperating the hub / spoke typology the setup becomes "advanced" Regardless

These examples shall be used on enterprise level Azure environments where landing zones are typically required to be seperated on subscriptions

<b>ATTENTION:</b> In cases where the hub is to be deployed completly seperate from all the spokes it MUST be deployed ahead of time, of the spokes, otherwise the deployment WILL fail

In case one or more spokes are deployed directly in the hub definition module call, these "direct" Spokes will successfully be deployed together with the hub itself, see example 1 for this exact and SPECIFIC behaviour

1. [Deploy hub with Firewall VPN and 1 spoke](#1-deploy-hub-with-firewall-vpn-and-1-spoke)
2. [Deploy hub and spokes in 2 module calls](#2-deploy-hub-and-spokes-in-2-module-calls)
3. [Deploy hub and spokes with custom settings](#3-deploy-hub-and-spokes-with-custom-settings)

### (1) Deploy hub with Firewall VPN and 1 spoke
Below see the code snippet for simply adding the hub and 1 spoke in the same module call.
This makes it possible to create the spoke TOGETHER with the hub.
Please note that even if more than 1 spoke is defined, the same spoke provider will be used on them all.

To create more than 1 spoke in different contexts, you must define them isolated in their own module call and create the appropiate provider configuration. See example 2 on details of that.
```hcl
//We must define a minimum of 2 providers as required by the module
//This behaviour is described in detail in the 'Getting started' Section'

provider "azurerm" {
  features {} //Default provider, will be used for the spoke required provider
}

provider "azurerm" {
  features {} //Will be used by the hub required provider
  alias = "hub"
  subscription_id = "<some sub id for the hub sub>"
}

//The Spoke objects will deploy in the SPOKE provider context parsed and NOT the hub provider context

module "hub_and_spoke_same_module_call" {
  source = "./.." #github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=2.0.0-hub-spoke
  topology_object = {
    env_name = "test" //Custom naming injection via using all 3 naming attributes - If any SPECIFIC names are set on attributes below, they will overrule this
    name_suffix = "contoso" //Custom naming injection via using all 3 naming attributes - If any SPECIFIC names are set on attributes below, they will overrule this
    project_name = "project1" //Custom naming injection via using all 3 naming attributes - If any SPECIFIC names are set on attributes below, they will overrule this
    dns_servers = ["1.1.1.1"] 
    subnets_cidr_notation = "/27" //Forces ALL subnets in the entire topology to be /27 UNLESS a vnet address space or specific subnet prefixes are defined
    
    tags = {
      "something" = "something" //Tags in the root adds the tag to ALL hub & spokes resource groups in the module call
    }

    hub_object = {
      location = "northeurope" //Custom location for the HUB resources ONLY, defaults to westeurope
      rg_name = "ignore-names-hub"
      tags = {
        "hub-something" = "hub-something"
      }

      network = {
        address_spaces = ["172.16.0.0/16"] //Default address space for a hub is ["10.0.0.0/24"]
        firewall = {
          name = "helloworld-fw"
          no_internet = true
          log_name = "tester-diag-log"
          log_diag_name = "tester-diag"
          log_daily_quota_gb = 10 //Per default there is no data limit / quota but we define 1 to be 10gb
          pip_name = "fw-tester"
        }

        vpn = {
          address_space = ["192.168.0.0/20"] //Default VPN address space is: 10.99.0.0/24 BECAUSE we set the hub to be in the 172.16 range.
          //Had we used the default address space from the module, the VPN address space will THEN rotate to 172.16.99.0/24 to avoid possible collisions
          gw_name = "gw-vpn"
        }

        subnet_objects = [
          {
            name = "GatewaySubnet" //Because Use_last_subnet OR use_first_subnet is not defined, the module defaults to "use_first_subnet"
          },
          {
            name = "AzureFirewallSubnet"
            address_prefix = ["172.16.1.0/24"]
          }
        ]
      }
    }

    //We are defining a spoke with 3 subnets DIRECTLY in the hub definition
    //The Spoke objects will deploy in the SPOKE provider context parsed and NOT the hub provider context
    spoke_objects = [
      {
        location = "ukwest"
        rg_name = "spoker"

        tags = {
          "spoker" = "spoker"
        }

        network = {
          address_spaces = ["10.99.0.0/20"]

          subnet_objects = [
            {
              use_last_subnet = true
            },
            {
              use_first_subnet = true
            },
            {
              use_last_subnet = true
            }
          ]
        }
      }
    ]
  }

  //We parse the 2 required providers - The left side of the expression is the names that MUST be parsed
  //The right side is our custom alias provider name defined in the top - to use default provider, we simply parse 'azurerm'
  providers = {
    azurerm.spoke = azurerm
    azurerm.hub = azurerm.hub
  }
}

//PLAN OUTPUT

Plan: 24 to add, 0 to change, 0 to destroy.
```

[Back to Examples](#examples-different-subs-only)

### (2) Deploy hub and spokes in 2 module calls
Different from example 1, we here want to create more than 1 spoke and each spoke must be in different subscriptions - For this, we need to define at least 1 of the 2 spokes in an isolated module call. We can still let the last spoke be created directly with the hub - Safes the amount of code required.

Remember - Because we create an isolated spoke from the hub module call, the hub module block MUST be deployed ahead of time, otherwise it will fail.

The below code snippet is using default values - See example 3 for a more complex example

```hcl
//Creating 3 providers, 2 custom for the spokes, and the hub will use default context

//Will be hub
provider "azurerm" {
  features {} //Will not define alias nor subcription_id for the default provider - Instead we rely on command line az logn for that
}

//Will be spoke 1
provider "azurerm" {
  features {}
  alias = "spoke1"
  subscription_id = "<Subscription_id_1>"
}

//Will be spoke 2
provider "azurerm" {
  features {}
  alias = "spoke1"
  subscription_id = "<Subscription_id_2>"
}

//DEPLOY WHEN HUB AND 1st SPOKE IS ALREADY DEPLOYED
module "hub_and_first_of_2_spokes" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=2.0.0-hub-spoke"
  
  topology_object = {
    
    hub_object = {
      network = {
        //All default
      }
    }

    spoke_objects = [ //All default values for the spoke defined'
      {
        network = {

          subnet_objects = [
            {

            }
          ]
        }
      }
    ]
  }
}

//Parsing the hub and spoke1 provider alias from the top of the code snippet
  providers = {
    azurerm.spoke = azurerm.spoke1
    azurerm.hub = azurerm
  }

  //TF Plan output:
  Apply complete! Resources: 7 added, 0 changed, 0 destroyed.

  //DEFINE SPOKE2 MODULE CALL AND LINK IT TO THE HUB
  module "second_spoke_alone" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=2.0.0-hub-spoke"

  topology_object = {
    
    hub_object = {
      network = {
         vnet_resource_id = values(module.hub_and_first_of_2_spokes.vnet_return_objects)[0].id //The index 0 is ALWAYS the HUB vnet IF the module call creates a hub, which we do
          //By parsing the resource id of the hub vnet already deployed, the module is able to peer the "isolated" Spoke2 to and from the hub
          //NO NEW HUB WILL BE CREATED

          vnet_peering_name = "custom-peering-name-from-hub" //We can create a custom peering name for the peering from the hub to spoke 2
      }
    }

    spoke_objects = [
      {
        network = {
          vnet_peering_name = "custom-peering-name-to-hub" //We can create a custom peering name for the peering from the spoke 2 and to the hub

          subnet_objects = [
            {
              //Default subnet config
            }
          ]
        }
      }
    ]
  }

  providers = {
    azurerm.spoke = azurerm.spoke2 //We use spoke2 provider context here, BUT SAME HUB CONTEXT
    azurerm.hub = azurerm.hub
  }
}

//TF Plan output
Plan: 5 to add, 0 to change, 0 to destroy.

//BOTH spoke 1 and 2 are now linked via peerings to the HUB and ALL 3 elements in the topology are in different subscriptions. EVEN the spoke defined within the hub.
```

[Back to Examples](#examples-different-subs-only)

### (3) Deploy hub and spokes with custom settings
This example expands on the idea from example 2 - The use of multiple module calls to allow the creation of multiple spokes in different subscriptions. 

In this example we focus on a more custom configuration. Its very important to node that all custom configuration when it comes to all the attributes defined in "parameters" Around 95% has the exact same behaviour as for the module in version 1.0.0.

Because we split the module calls, we MUST be aware of the "default behaviour" Of how the module creates all the address spaces / CIDR´s for each virtual vnet, especially the spoke vnets - This will make more sense in the example

We will define both Firewall, VPN and more below.

```hcl
//Creating 3 providers, 2 custom for the spokes, and the hub will use default context

//Will be hub
provider "azurerm" {
  features {} //Will not define alias nor subcription_id for the default provider - Instead we rely on command line az logn for that
}

//Will be spoke 1
provider "azurerm" {
  features {}
  alias = "spoke1"
  subscription_id = "<Subscription_id_1>"
}

//Will be spoke 2
provider "azurerm" {
  features {}
  alias = "spoke1"
  subscription_id = "<Subscription_id_2>"
}

//Deploy the module for the hub ALONE FIRST - NOTICE the larger comment block as its very important information about the module behaviour
module "hub_by_itself" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=2.0.0-hub-spoke"

  topology_object = {
    name_prefix = "contoso" //This naming injection will only affect hub resources and NOT any spokes linked to it OUTSIDE of this module call
    env_name = "prod"
    project_name = "connectivity"

    hub_object = {
      location = "northeurope" //Effects any sub component that are part of the hub

      network = {

        address_spaces = ["172.16.0.0/16"]
        vnet_spoke_address_spaces = ["10.0.1.0/24", "192.168.0.0/16"] //In order for the module during the creation of Firewall rules, it must know the comming spoke vnet address spaces AHEAD of time
        //As the hub must be deployed alone first, its recommended to use static values for this attribute
        //The first address space in the attribute comes from using a default config for spoke 1´s network address spaces, but for spoke 2, we configure a custom address space
        //There is a very important reason for this and it all comes down to how the module behaves when all the automatic subnetting is being done
        //The module always calculates address spaces for spokes (default behaviour) by starting with a /24 and x.x.1.x and then incrementing by 1
        //The problem then arrises when 2 different module calls are made which both create 1 to many spokes - As BOTH SPOKE module calls will SUBNET THE SAME WAY which causes collisions between the vnets when peering
        //To avoid this issue, we can take FULL control over all address spaces being created for the spoke vnets - We can even control each single subnet´s address prefix if we want to

        firewall = {
          no_internet = true //Will stop the module from creating a Firewall rule ALLOW to 443, 80 and 53 for any spoke vnet
          log_diag_name = "custom-diag-settings-name"
          pip_name = "custom-fw-pip-name" //Overwrites naming injection defined in the top level object
          //Rest default
        }

        vpn = {
          pip_name = "gw-custom-pip-name" //Overwrites naming injection defined in the top level object
          //Rest default
        }

        //Because we define both the VPN and FW objects, we must add the required Microsoft defined subnet names
        subnet_objects = [
          {
            name = "AzureFirewallSubnet"
            use_last_subnet = true //Module will CIDR subnet /26 the last possible subnet from the custom defined hub vnet address space of ["172.16.0.0/16"]
          },
          {
            name = "GatewaySubnet" //By not defining an attribute to tell the module how to subnet CIDR this subnet´s address space, the module will use the first possible CIDR of the hub vnet address space
          }
        ]
      }
    }
    //NO SPOKE OBJECTS DEFINED HERE
  }

  providers = {
    azurerm.spoke = azurerm
    azurerm.hub = azurerm.hub
  }
}

//TF Plan output
Plan: 10 to add, 0 to change, 0 to destroy.

//Defining spoke 1 with default values - This means for the attribute in the hub module call the attribute "vnet_spoke_address_spaces" The vnet for the spoke created will be default as in ["10.0.1.0/24"]
//Spoke 1 and spoke 2 can be deployed at the same time - ONLY the hub MUST be deployed ahead of time
module "spoke_1_simple" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=2.0.0-hub-spoke"

  topology_object = {
    
    hub_object = {
      network = {
        vnet_resource_id = values(module.hub_by_itself.vnet_return_objects)[0].id //To link the spoke to the hub
      }
    }

    spoke_objects = [
      {
        network = {
          
          subnet_objects = [
            {
              name = "subnet_1"
              use_last_subnet = true
            },
            {
              name = "subnet_2"
              use_first_subnet = true //Just to showcase that we can easily make the first subnet use the last possible CIDR block and the last subnet use the first
            }
          ]
        }
      }
    ]
  }

  providers = {
    azurerm.spoke = azurerm.spoke1
    azurerm.hub = azurerm.hub
  }
}

//Defining spoke2 - Here we can no longer use the default address spacing created by the module, as the default address spaces are already used by spoke 1
module "spoke_2_advanced" {
  source = "./.."

  topology_object = {
    name_suffix = "fabrikam"
    //Not using env_name and project_name BUT WE CAN ÌF WE WANT TO

    subnets_cidr_notation = "/27" //ALL SUBNETS NOT HAVING AN CUSTOM ADDRESS PREFIX WILL USE THIS CIDR
    //When using a custom CIDR for subnets make sure to know whether the overall address space for the vnets can contain such subnet size
    //Also as it is with subnetting, the smaller CIDR the less amount of subnets can be created
    //If we end up defining more subnets than the CIDR can hold, terraform will fail
    
    hub_object = {
      network = {
        vnet_resource_id = values(module.hub_by_itself.vnet_return_objects)[0].id //To link the spoke to the hub

        fw_resource_id = values(module.hub_by_itself.fw_return_object)[0].id  //Because the connected hub has the Firewall created AND by default an internet opening for spoke vnets are created we MUST parse either the id of the firewall or its private ip
        //If we do not parse this information, the spoke cannot take part of the Firewall rule and will have NO internet
      }
    }

    spoke_objects = [
      {
        network = {

          address_spaces = ["192.168.0.0/16"] //To avoid a CIDR subnetting collision with the spoke 1 defined above with default values
          
          subnet_objects = [
            {
              name = "subnet_1"
              use_last_subnet = true
              address_prefix = ["192.168.99.0/24"] //Ignores ALL automatic subneting EVEN the forced attribute "subnets_cidr_notation" Of /27 defined in the top level object
            },
            {
              name = "subnet_2"
              use_first_subnet = true //Just to showcase that we can easily make the first subnet use the last possible CIDR block and the last subnet use the first
            }
          ]
        }
      }
    ]
  }

  providers = {
    azurerm.spoke = azurerm.spoke2 //Using the 2nd spoke´s provider config
    azurerm.hub = azurerm.hub
  }
}

//TF Plan output of both SPOKE 1 and SPOKE 2 TOGETHER
Plan: 16 to add, 0 to change, 0 to destroy.
```

[Back to Examples](#examples-different-subs-only)

## Known errors
This chapter is all about understanding the different errors that you can encounter while using the module. Use this chapter as a reference to different "error" Codes and their solution

### (1) Resource names has incorrect names, like missing a seperator, having double seperators and missing segments like project name or env name within names

#### Example of this issue:

 In the below incorrect name, we have ONLY filled out the attributes "env_name" And "name_suffix" - As it states in the [Parameters](#parameters) we need to ALSO define a project_name if the 2 other attributes are also filled out
 ```hcl
 name                    = "prodhub-contoso-fw-pip" //Missing "-" Between "prod" And "hub"
 ```

 In the below incorrect name, we are missing the project_name entirely, because we only defined the attributes "name_suffix" And "project_name" But forgetting the attribute "env_name"
 ```hcl
 name                    = "hub-contoso-fw" //Missing the "project_name" From the name
```

### (2) Module fails while deploying subnets due to overlapping address_prefixes
This error can occur in 1 of 2 ways:
1. While creating the list of subnets, either in the hub or any spokes, we mix subnets having address_prefix defined by us, to other subnets in the same vnet using either of the attributes "use_first_subnet" Or "use_last_subnet" To easiliest solve this, lower the complexity of the exact configuration of how each subnet gets a calculated CIDR

2. Creating too many or too large subnets for the vnets address space. Both scenarios can occur together - E.g say you have the following config but recieve the error about overlapping subnets:

```hcl
module "overlap_example" {
   source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=1.0.0-hub-spoke"
   topology object = {
     hub_object = {
        network = {} //Just use all defaults for the hub, not important for the example
     }

     spoke_objects = [
       {
         network = {
           subnet_objects = [
              {
                address_prefix = ["10.0.1.0/27"] //We will use default address block of 10.0.x.0/24 As provided by the module, where x is the spoke number, which is 1 in this case
              },
              {
                use_last_subnet = true //Wont collide
              },
              {
                use_last_subnet = true //Wont collide with anything because the module will take the last possible subnet
              },
              {
                use_last_subnet = true //Will collide with subnet1, for an explanation, see below the error defined
              }

              //NOW, depending on the number of subnets we create now and how large they are, we can create another collision simply by trying to subnet the original /24 more than it can
           ]
         }
       }
     ]
   }
}

   //The above is rather a complex subnetting setup and it actually creates a specific collision on subnet4, which will look like the following in terraform:
    │ Error: creating Subnet (Subscription: "00000000-0000-0000-0000-000000000000"
    │ Resource Group Name: "rg-spoke1"
    │ Virtual Network Name: "vnet-spoke1"
    │ Subnet Name: "subnet4-spoke1"): performing CreateOrUpdate: unexpected status 400 (400 Bad Request) with error: NetcfgSubnetRangesOverlap: Subnet 'subnet4-spoke1' is not valid because its IP address range overlaps with that of an existing subnet in virtual network 'vnet-spoke1'.
    │
    │   with module.overlap_example.azurerm_subnet.subnet_object["subnet4-2-unique-spoke1"],
    │   on .terraform\modules\overlap_example\terraform projects\modules\azurerm-hub-spoke\azurerm-hub-spoke.tf line 293, in resource "azurerm_subnet" "subnet_object":
    │  293: resource "azurerm_subnet" "subnet_object" 

  //This issue comes because we use index 0 of the subnets to reserve a custom address prefix and then on index 1 use the attribute "use_last_subnet"
  //Because we used index 0 of the subnets to define a custom prefix, the module will not be able to use the last possible CIDR block of /26 from the original /24 address space automatically
  //The side effect of this, causes the 4th subnet creation to fail, because it overlaps with our first subnet, simply because we lost the last /26 CIDR subnet
  //To fix this issue, either manually define the 4th (last) subnet manually with the correct CIDR subnetting to reach the last possible subnet in the /24 block
  //Then for the last 2 subnets, contintue to use the attribute "use_last_subnet" This way, the module can once again automically handle the subnetting for the last 2 subnets

  //Notice the change in subnet4 to use a manual address prefix now instead to solve the collision
  //Also - In general I recommend to simply use the attributes "use_last_subnet" And "use_first_subnet" To let the module subnet for you
  //The 2 attributes can easily be mixed with each other - As the module will then have full control over ALL indexes of the subnets

  module "overlap_example" {
   source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=1.0.0-hub-spoke"
   topology object = {
     hub_object = {
        network = {} //Just use all defaults for the hub, not important for the example
     }

     spoke_objects = [
       {
         network = {
           subnet_objects = [
              {
                address_prefix = ["10.0.1.0/27"] //We will use default address block of 10.0.x.0/24 As provided by the module, where x is the spoke number, which is 1 in this case
              },
              {
                use_last_subnet = true //Wont collide with anything, but because subnet1 is manual, and this subnet is index 1, we wont take the last possible of .192 but instead the 2nd last
              },
              {
                use_last_subnet = true //Wont collide with anything because the module will take the last possible subnet, will be closest POSSIBLE subnet to subnet1,
              },
              {
                 address_prefix = ["10.0.1.192/26] //Helping the module by manually taking the last possible subnet, as subnet2 was only able to take the 2nd last /26 CIDR
              }

              //NOW, depending on the number of subnets we create now and how large they are, we can create another collision simply by trying to subnet the original /24 more than it can
           ]
         }
       }
     ]
   }
}
```


[Known errors](#known-errors)

[Back to the top](#table-of-contents)