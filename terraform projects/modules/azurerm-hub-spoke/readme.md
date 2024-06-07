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

Just below here, two different visual examples of types of hub-spokes can be seen. Both can be directly deployed with the module, see the [Examples](#examples) for the actual code.

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

Please take note of the 'Azure Provider Version' among the various providers utilized by the module. Keep in mind that there WILL be a required minimum version, and this requirement can vary with each module version.

<b>Module version 1.0.0 requires the following provider versions:</b>

| Provider name | Provider url | Minimum version |
| -------------- | ------------ | ---------------- |
| azurerm        | [hashicorp/azurerm](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs) | 3.99.0 |
|                |                          |        |

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
If you're using VSCode, leverage the Terraform extension from HashiCorp to benefit from 'Intellisense.' Note that, in some cases, you may need to clone the repository as the HashiCorp Terraform extension might encounter difficulties resolving parameters through a remote module.

(Intellisense might need a local copy of the repository, clone <a href="https://github.com/ChristofferWin/codeterraform.git">codeterraform</a>)

<img src="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-vm-bundle/pictures/gifs/Intellisense1.gif"/>

The below lists showcases all possible parameters. For default values go to <a href="https://github.com/ChristofferWin/codeterraform/tree/main/terraform%20projects/modules/azurerm-hub-spoke#detailed-description">Detailed Description</a>

### Attributes on the "top" Level of the "typology_object"
1. customer_name = (optional) A string defining the name of the customer. Will be injected into the overall resource names. OBS. Using this variable requires both either "name_prefix" OR "name_suffix" AND "env_name" to be provided as well

2. location = (optional) A string defining the location of ALL resources deployed (overwrites ANY lower set location)

3. name_prefix = (optional) A string to inject a prefix into all resource names - This variable makes it so names follow a naming standard: \<resource abbreviation>\-<name_prefix>\-\<Identier, either "hub" or "spoke">

4. name_suffix = (optional) A string to inject a suffix into all resource names - This variable also makes names follow a naming standard: <Identifier, either "hub" or "spoke">\-\<name_suffix>\-\<resource abbreviation>

5. env_name = (optional) A string defining an environment name to inject into all resource names. OBS. Using this variable requires both either "name_prefix" OR "name_suffix" AND "customer_name" To be provided as well

6. dns_servers = (optional) A list of strings defining DNS server IP adresses to set for ALL vnets in the typology (overwrites ANY lower set DNS servers)

7. tags = (optional) A map of strings defining any tags to set on ALL vnets and resource groups (Any tags set lower will be appended to these tags set here)

8. subnets_cidr_notation = (optional) A string defining what specific subnet size that ALL subnets should have - Defaults to "/26"

### Attributes on the "hub" Level of the "typology_object"
1. rg_name = (optional) A string defining the specific name of the hub resource group resource (Overwrites any name injection defined in the top level attributes)

2. location = (optional) A string defining the location of which to deploy the hub to (If the top level location is set, this will be overwritten)

3. tags = (optional) A map og strings defining any tags to set on the hub resources

4. network = (required) an object 

#### Example of values to use for mgmt parameters
```hcl
rg_name = "some-rg"
location = "northeurope" Must be a valid Microsoft location, use the powershell module 'Get-AzVMSKu' With the switch '-ShowLocations' To see all valid regions
env_name_1 = "p" = prod
env_name_2 = "prod" = prod
env_name_3 = "prd" = prod
env_name_4 = "pd" = prod
env_name_5 = "t" = test
env_name_6 = "test" = test
env_name_7 = "tst" = test

//Many more names can be used for the environments, these are simply a slice of them
```

### object defined parameters
1. vm_windows_objects & vm_linux_objects = a list of objects defining:
    - name
    - admin_username
    - admin_password
    - size (vm)
    - size_pattern (use a pattern to find a vm size, like e.g. 'A1')
      - size will then be decided by the module using specific logic - See the advanced examples section for more information
    - boot_diagnostics which is an object defining:
      - storage_account, see the <a href="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-vm-bundle/variables.tf">expanded defintion</a>, search for variable 'storage_account'
    - os_disk which is an object defining:
      - name
      - disk_size_gb
      - see the <a href="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-vm-bundle/variables.tf">expanded defintion</a> for all other attributes to set, search for variable 'os_disk'
    - source_image_reference which is an object defining:
      - offer
      - publisher
      - sku
      - version
      - *Warning* Only utilize this option when a custom SKU or version is necessary. To obtain this information, employ the PowerShell module 'Get-AzVmSku' with parameters '-Location <location> -OperatingSystem <os_name>' to retrieve all the necessary details.
    - nic which is an object defining:
      - name
      - dns_servers
      - enable_ip_forwarding
      - ip_configuration, see the <a href="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-vm-bundle/variables.tf">expanded defintion</a> for all other attributes to set, search for variable 'nic'
    - public_ip which is an object defining:
      - name
      - allocation_method
      - sku
      - tags
    - admin_ssh_key which is a list of objects defining (Only Linux vms):
      - public_key
      - username
    - Many other attributes, they can all be found at <a href="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-vm-bundle/variables.tf">expanded defintion</a>
2. vnet_object = an object defining:
    - name
    - address_space
    - tags
2. subnet_objects = a list of objects defining:
    - name
    - address_prefixes (bastion must be at least /26)
    - The order of defining the objects are defining the bastion subnet first, then the vm subnet - See the examples for a more detailed explanation
3. bastion_object = an object defining:
    - name
    - copy_paste_enabled
    - file_copy_enabled
    - sku 
    - scale_units
    - tags
    - *Warning* Its only recommended to use this parameter in case the number of 'scale_units' is to be customized  See the advanced examples for guidance
4. nsg_objects = a list of objects defining:
    - name
    - subnet_id
    - tags
    - security_rule, see the <a href="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-vm-bundle/variables.tf">expanded defintion</a>, search for variable 'nsg_objects'
    - *Warning* Its only recommended to use this parameter in case the security rule is to be customized - See the advanced examples for guidance
5. kv_object = an object defining:
    - name (must be globally unique)
    - network_acls = an object defining:
      - Define a custom network security ruleset for the kv
    - For all other attributes, see the <a href="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-vm-bundle/variables.tf">expanded defintion</a>, search for variable 'kv_object'
    - *Warning* Its only recommended to use this parameter in case a network security rule is to be customized - See the advanced examples for guidance

#### Example of defining custom sub resource objects
```hcl
//We will only define some simple object configurations here. For more information, see the advanced examples
vnet_object = {
  address_space = ["172.16.0.0/16"]
  name = "custom-vnet"
  tags = {
    "environment" = "prod"
  }
}

//You need define 2 subnets in case 'create_bastion = true' The module will always use index 0 for bastion
//Name is not required and for the bastion subnet it will always be 'AzureBastionSubnet' Regardless of user defined name
subnet_objects = [
 {
  name = "WILL BE REPLACED BY 'AzureBastionSubnet'"
  address_prefixes = ["172.16.99.0/26"]
 },
 {
  name = "myvm-subnet"
  address_prefixes = ["172.16.0.0/24"]
 }
]

create_bastion = true
```

#### Example 2 of defining custom sub resource objects
```hcl
//In this example we want a custom bastion config, but rest shall be default. Notice how we do not need to specify the create_bastion switch as the object is already defined by us
bastion_object = {
  copy_paste_enabled = false
  file_copy_enabled = false
  name = "my-custom-bastion"
  scale_units = 6
  sku = "Standard"
}

//With no other configuration than the required, a custom bastion will be created with default vnet and default subnets
```
[Back to the top](#table-of-contents)
## Detailed Description
First off, these are all potential subtypes of resources available for deployment, with the option to deploy any number of virtual machines:

The below list also contain resource types default value in case the user adds any of the 'create' <a href="https://github.com/ChristofferWin/codeterraform/tree/main/terraform%20projects/modules/azurerm-vm-bundle#parameters">parameters</a>

1. VM(s), both Windows & Linux
  - Any amount can be created, this is only limitted by the subscriptions internal quota for CPU cores. The module can return this information, see <a href="https://github.com/ChristofferWin/codeterraform/tree/main/terraform%20projects/modules/azurerm-vm-bundle#return-values">outputs</a>
    - admin_username = localadmin
    - admin_password = <random 16 length password with special chars>
    - os_disk_caching = Read/Write
    - os_disk_size = 128 GB
    - size = Standard_B2ms
2. Resource group
3. Virtual network
  - The address space depends on the environment, with the first space designated for VM subnets and the second reserved for bastion and other management resources.:
    - prod = ["10.0.0.0/16", "10.99.0.0/24"]
    - test = ["172.16.0.0/20", "172.16.99.0/24]
    - any other environment name = ["192.168.0.0/24", "192.168.99.0/24"] (can be used for environments like dev)
4. Subnet(s)
  - The address prefixes of each subnet, bastion subnet wont be created unless the resource is to be deployed
    - bastion subnet = /26 (as per required by Azure)
    - vm subnet = /25 (123 host addresses)
5. Bastion
  - Configured to work for most use-cases
    - copy_paste_enabled = true
    - file_copy_enabled = true
    - sku = Standard
    - scale_units = 2
6. Public ip(s)
  - Either one per vm or one for each specific vm(s)
    - sku_name = "standard"
    - allocation_method = "Static"
7. Network Security group
 - Add rules to vm subnet
    - ALLOW ports 22/3389 from ANY to VM SUBNET (ssh & rdp)
8. Storage Account
 - Either one per vm or one total for all vms. Used for boot-diagnostic settings
    - access_tier = "Cool"
    - public_network_access_enabled = true
    - account_tier = "Standard"
    - account_kind = "StorageV2"
    - account_replication_type = "LRS"
9. Key Vault & secrets
  - One total to store all vm password secrets in
    - sku_name = "standard"
    - enabled_for_deployment = true
    - enabled_for_disk_encryption = false
    - enabled_for_template_deployment = false
    - enable_rbac_authorization = true
    - purge_protection_enabled = true
    - public_network_access_enabled = true
    - soft_delete_retention_days = 7

[Back to the top](#table-of-contents)
## Return Values
Its important to state that almost all values returned from the module is of type map. This can either be used to our advantage by making our variable references more type-safe
or we can simply use a function like 'values' to make the return value a list of object instead, where we can then simply use int index-based references like [0]

See below list of possible return values:
 - summary_object = simpel object, can call attributes directly
    - <general information> even how many CPU cores are left in terms of quota on the subscription
    - network_summary
    - windows_objects
      - passwords are NOT 
    - linux_objects
 - rg_object = simpel object, can call attributes directly
    - id
    - name
    - location
 - vnet_object = map of object, call specific key, or use values()
    - id
    - name
    - address_space
    - See vnet <a href="https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/virtual_network#attributes-reference">Hashicorp docs / attributes references</a>
 - subnet_object = map of object, call specific key, or use values()
    - id
    - name
    - virtual_network_name
    - address_prefixes
 - nsg_object = map of object, call specific key, or use values()
    - id
    - name
    - See nsg
 - nic_object = map of object, call specific key, or use values()
    - id
    - name
    - See nic <a href="https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_interface#attributes-reference">Hashicorp docs / attribute references</a>
 - pip_object = map of object, call specific key, or use values()
    - id
    - name
    - ip_address
    - fqdn
 - windows_object = map of object, call specific key, or use values()
    - id
    - name
    - See windows vm <a href="https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/windows_virtual_machine#attributes-reference">Hashicorp docs / attribute references</a>
 - linux_object = map of object, call specific key or use values()
    - id
    - name
    - See linux vm <a href="https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/linux_virtual_machine#attributes-reference">Hashicorp docs / attribute references</a>
 - storage_object = map of object, call specific key or use values()
    - id
    - name
    - See storage <a href="https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/storage_account#attributes-reference">Hashicorp docs / attribute references</a>

[Back to the top](#table-of-contents)
## Examples
<b>This section is split into 2 different sub sections:</b>

- <a href="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-vm-bundle/readme.md#simple-examples---separated-on-topics">Simple examples</a> = Meant to showcase the easiest ways to deploy vms with its dependencies
- <a href="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-vm-bundle/readme.md#advanced-examples---seperated-on-topics">Advanced examples</a> = Meant to showcase different combinations of resources to deploy with vms

[Back to the top](#table-of-contents)

### Simple examples - Separated on topics
1. [How to retrieve required information like os_name](#1-how-to-retrieve-required-information-like-os_name)
2. [A few vms and bastion](#2-a-few-vms-and-bastion)
3. [Using existing virtual vnet and subnet](#3-using-existing-virtual-vnet-and-subnet)
4. [Use attributes like size_pattern and defining a custom os_disk configuration](#4-use-attributes-like-size_pattern-and-defining-a-custom-os_disk-configuration)
5. [Avoid using PowerShell 7 entirely when deploying with the module](#5-avoid-using-powershell-7-entirely-when-deploying-with-the-module)


### (1) how to retrieve required information like 'os_name'
```powershell
#Make sure to have the PowerShell module 'Get-AzVMSku' Installed
#Must be in administrator mode to install
Install-module Get-AzVMSku -Force

#Run the show command for different information that you may require to run the terraform module

#Retrive name needed for parameter 'os_name' of the terraform module
Get-AzVmSku -ShowVMOperatingSystems

#Sample output
server2008
server2012
server2012r2
....

#Retrieve valid Azure locations required by parameter 'location'
#Requires an Azure context
Login-AzAccount #Interactive browser prompt
Get-AzVmSku -ShowLocations #Use the 'ShortName' output

#Sample output
ShortName          LongName
---------          --------
eastus             East US
eastus2            East US 2
westus             West US
centralus          Central US
northcentralus     North Central US
....

#We can also retrieve a specific os and use this information for other parameters
$VMObject = Get-AzVMSKU -Location "westeurope" -OperatingSystem "windows11"

#Sample output required in case you want to deploy vms with specific version and sku
$VMObject | Select-Object Publisher, Offer

Publisher               Offer
---------               -----
MicrosoftWindowsDesktop Windows-11

$VMObject.SKUs #Sample of windows11 skus

win11-21h2-avd
win11-21h2-ent
win11-21h2-entn
win11-21h2-pro

#In summary, the PowerShell module can be used interactively before executing the Terraform module to gather necessary information for deployment. 
#For most applications, obtaining specific OS information is unnecessary, as the module can handle this automatically. 
#However, in cases where #a particular SKU or SKU version is required for any operating system, the information obtained from the last output needs to be provided as input to the module.
```
For more information about how to use the PowerShell module, please visit the <a href="https://github.com/ChristofferWin/codeterraform/blob/main/powershell%20projects/modules/Get-AzVMSku/Examples.md">readme</a> where a lot of examples are shown

[Back to the Examples](#examples)
### (2) A few vms and bastion
```hcl
//Boilerplate

provider "azurerm" {
  features{}
}

module "simple_vms" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"

  rg_name = "simple-vms-rg"
  create_public_ip = true
  create_nsg = true //With publicIP true, nsg should also be created otherwise we cant connect via the public ip
  create_diagnostic_settings = true //Will create us a storage account and link both vms to it

  vm_windows_objects = [
    {
      name = "simple-win-vm"
      os_name = "windows10"
    }
  ]

  vm_linux_objects = [
    {
      name = "simple-linux-vm"
      os_name = "centos"
    }
  ]
}

//Create a simple output to see our deployment results
output "deployment_results" {
  value = module.simple_vms.summary_object
}

//Sample output
/*
"linux_objects" = [
    {
      "admin_username" = "localadmin"
      "name" = "simple-linux-vm"
      "network_summary" = {
        "private_ip_address" = "192.168.0.5"
        "public_ip_address" = "20.126.18.32"
      }
      "os" = "centos"
      "os_sku" = "8_5-gen2"
      "size" = {
        "cpu_cores" = 2
        "memory_gb" = 8
        "name" = "Standard_B2ms"
      }
    },
  ]
  */
```
How it looks in Azure:
<img src="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-vm-bundle/pictures/3rd-vm-black.png" />

[Back to the Examples](#examples)
### (3) Using existing virtual vnet and subnet
```hcl
//The resource group, virtual network & subnet must be created in advance
//Using reference from example 2, adding 1 new windows vm to the environment and a public ip for it
module "existing_resources_vm" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"
  rg_id = module.simple_vms.rg_object.id
  vnet_resource_id = module.simple_vms.vnet_object["vm-vnet"].id
  subnet_resource_id = module.simple_vms.subnet_object["vm-subnet"].id
  create_public_ip = true //So we can connect to it

  vm_windows_objects = [
    {
      name = "windows-vm01"
      os_name = "windows11"
    }
  ]
}

//Create a simple output to see our deployment results
output "existing_resources_vm_result" {
  value = module.existing_resources_vm.summary_object
}

//Sample output
/*
windows_objects" = [
    {
      "admin_username" = "localadmin"
      "name" = "windows-vm01"
      "network_summary" = {
        "private_ip_address" = "192.168.0.6"
        "public_ip_address" = "40.118.59.198"
      }
      "os" = "windows11"
      "os_sku" = "win11-23h2-pron"
      "size" = {
        "cpu_cores" = 2
        "memory_gb" = 8
        "name" = "Standard_B2ms"
      }
    },
  ]
*/
```
How it looks in Azure:
<img src="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-vm-bundle/pictures/4th-vm-black.png" />

[Back to the Examples](#examples)
### (4) Use attributes like 'size_pattern' and defining a custom 'os_disk' configuration 
```hcl
module "vm_specific_config" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=1.3.0"

  rg_name = "vm-specific-config-rg"
  
  create_public_ip = true
  create_nsg = true //With publicIP true, nsg should also be created otherwise we cant connect via the public ip
  create_diagnostic_settings = true //Will create us a storage account and link both vms to it

  vm_windows_objects = [
    {
      name = "simple-win-vm" //Required
      os_name = "windows10" //Required
      admin_username = "mycustomuser" //Optional
      admin_password = "ShowCasedONLYFORDEMO!" //Optional
      //See the parameters section or use Intellisense to see the rest of the possible attributes to set
      
      os_disk = {
        name = "custom-os-disk" //Optional
        disk_size_gb = 256 //Optional
        caching = "ReadWrite" //Required
        //See the parameters section or use Intellisense to see the rest of the possible attributes to set
      }
    },
    {
      name = "default-vm"
      os_name = "windows11"
      size_pattern = "DS4" //Module retrieves the closest vm sized matched
      //See the parameters section or use Intellisense to see the rest of the possible attributes to set
    }
  ]
}

//Output
output "vm_specific_config_result" {
  value = module.vm_specific_config.summary_object
}

//Sample (Notice how the size of the vm became 'Standard_DS4_v2' and we only wrote 'DS4' in the 'size_patteren')
/*
"windows_objects" = [
    {
      "admin_username" = "localadmin"
      "name" = "default-vm"
      "network_summary" = {
        "private_ip_address" = "192.168.0.4"
        "public_ip_address" = "13.94.244.99"
      }
      "os" = "windows11"
      "os_sku" = "win11-23h2-pron"
      "size" = {
        "cpu_cores" = 8
        "memory_gb" = 28
        "name" = "Standard_DS4_v2" 
      }
    },
]
/*
```
How it looks in Azure:

<img src="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-vm-bundle/pictures/5th-vm-black.png" />

[Back to the Examples](#examples)
### (5) Avoid using PowerShell 7 entirely when deploying with the module
```hcl
module "avoid_using_powershell" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"

  rg_name = "avoid-using-powershell-rg"

  create_public_ip = true
  create_nsg = true

  //We can define any attribute, but because we have statically defined the 'source_image_reference' PowerShell will NOT be executed by the module
  vm_linux_objects = [
    {
      name = "custom-sku-ubuntu"
      os_name = "ubuntu"

      //used the PS module 'Get-AzVMSku' to retrieve the below information, only parsing 'Get-AzVmsku -Location westeurope -OperatingSystem ubuntu -NewestSKUsVersions'
      source_image_reference = {
        offer = "UbuntuServer"
        publisher = "Canonical"
        sku = "14.04.5-DAILY-LTS"
        version = "14.04.201911070"
      }
    }
  ]
}

output "avoid_using_powershell" {
  value = module.avoid_using_powershell.summary_object
}

//Sample output
/*
"linux_objects" = [
    {
      "admin_username" = "localadmin"
      "name" = "custom-sku-ubuntu"
      "network_summary" = {
        "private_ip_address" = "192.168.0.4"
        "public_ip_address" = "13.81.201.156"
      }
      "os" = "ubuntu"
      "os_sku" = "14.04.5-DAILY-LTS"
      "size" = {
        "cpu_cores" = null
        "memory_gb" = null
        "name" = "Standard_B2ms"
      }
    },
  ]
/*
```
How it looks in Azure:

<img src="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-vm-bundle/pictures/6th-vm-black.png" />

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