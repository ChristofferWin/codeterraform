# Azure VM Bundle Terraform Module

## Table of Contents

1. [Description](#description)
2. [Detailed Description](#detailed-description)
3. [Prerequisites](#prerequisites)
4. [Versions](#versions)
5. [Parameters](#parameters)
6. [Return Values](#return-values)
7. [Examples simple](#examples-simple)
8. [Examples advanced](#examples-advanced)

## Description

Welcome to the Azure VM Bundle Terraform module! The "azurerm-vm-bundle" module facilitates the effortless deployment of Azure virtual machines, accommodating both Linux and Windows operating systems across multiple versions. This capability is achieved through the integration of a PowerShell module, available in the same repository at <a href="https://github.com/ChristofferWin/codeterraform/tree/main/powershell%20projects/modules">Get-AzVMSku</a>. This PowerShell module aids in retrieving essential information such as SKU, SKU version, and more.

The module boasts extensive configuration flexibility, supporting a wide array of customization options. For a comprehensive overview of these options, please refer to the detailed description provided.

Furthermore, the module is more than capable of deploying various subtypes that is typically used together with Azure virtual machines.

<b>Example deployment of 2 Linux machines with public ips, NSG and ssh setup:</b>
</br>
</br>
<img src="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/development/azurerm-vm-bundle/pictures/gifs/SSH-Demo.gif"/>


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
    - vm subnet = /25 (123 host addresses)
    - bastion subnet = /26 (as per required by Azure)
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

## Prerequisites

Before using this module, make sure you have the following:

- Active Azure Subscription
  - Must either have RBAC roles:
    - Contributor (Module wont be able to assign kv rbac role)
    - Contributor + User Access Administrator
    - Owner
- Installed Terraform (download [here](https://www.terraform.io/downloads.html))
- Azure CLI installed for authentication (download [here](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli))
- PowerShell Core installed for intergration with PS module (download [here](https://learn.microsoft.com/en-us/powershell/scripting/install/installing-powershell-on-windows?view=powershell-7.3))
  - Its possible to run module without it, see <a href="https://github.com/ChristofferWin/codeterraform/tree/main/terraform%20projects/modules/azurerm-vm-bundle#examples">Examples</a> for details
- Have local admin permissions on the machine executing

## Versions
The table below outlines the compatibility of the module:

Please take note of the 'Azure Provider Version' among the various providers utilized by the module. Keep in mind that there WILL be a required minimum version, and this requirement can vary with each module version.

<b>Module version 1.0.0 requires the following provider versions:<b>

| Provider name | Provider url | Minimum version |
| ------------------ | ---------------------- | -------------- |
| azurerm | <a href="https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs">hashicorp/azurerm</a>  | 3.76.0 |
| null | <a href="https://registry.terraform.io/providers/hashicorp/null/latest/docs">hashicorp/null</a> | 3.2.1 |
| random | <a href="https://registry.terraform.io/providers/hashicorp/random/latest/docs">hashicorp/random</a> | 3.5.1 |
local | <a href="https://registry.terraform.io/providers/hashicorp/random/latest/docs">hashicorp/local</a> | 2.4.0

For the latest updates of the terraform module, check the [releases](https://github.com/your-username/azurerm-vm-bundle/releases) page.

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
│ Could not retrieve the list of available versions for provider hashicorp/azurerm: no available releases match the given constraints 3.64.0, >= 3.76.0
```
To solve it, simply remove the version parameter OR use a version that is the minimum requirement from <a href="https://github.com/ChristofferWin/codeterraform/tree/main/terraform%20projects/modules/azurerm-vm-bundle#versions">Versions</a>:
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
- Installed hashicorp/azurerm v3.81.0 (signed by HashiCorp)

Terraform has been successfully initialized!
```
Please see the <a href="https://github.com/ChristofferWin/codeterraform/tree/main/terraform%20projects/modules/azurerm-vm-bundle#parameters">Parameters</a> section for a better understanding of what the module can take as inputs


## Parameters
If you're using VSCode, leverage the Terraform extension from HashiCorp to benefit from 'Intellisense.' Note that, in some cases, you may need to clone the repository as the HashiCorp Terraform extension might encounter difficulties resolving parameters through a remote module.

(Intellisense might need a local copy of the repository, clone <a href="https://github.com/ChristofferWin/codeterraform.git">codeterraform</a>)

<img src="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/development/azurerm-vm-bundle/pictures/gifs/Intellisense1.gif"/>

The below lists showcases all possible parameters. For default values go to <a href="https://github.com/ChristofferWin/codeterraform/tree/main/terraform%20projects/modules/azurerm-vm-bundle#detailed-description">Detailed Description</a>


### resource_id (to avoid that the module must create the resoruce type)
1. rg_id = resource id of a resource group to deploy resources to
2. vnet_resource_id = resource id of virtual network to deploy subnet to
3. subnet_resource_id = resource id of the vm subnet where the module shall deploy vms to
4. kv_resource_id = resource id of the key vault to add vm admin password secret to

#### Example of each resource id type
```hcl
rg_id = /subscriptions/<sub id>/resourceGroups/<rg name>
vnet_resource_id = /subscriptions/<sub id>/resourceGroups/<rg name>/providers/Microsoft.Network/virtualNetworks/<vnet name>
subnet_resource_id = /subscriptions/<sub id>/resourceGroups/<rg name>/providers/Microsoft.Network/virtualNetworks/<vnet name>/subnets/<subnet name>
kv_resource_id = /subscriptions/<sub id>/resourceGroups/<rg name>/providers/Microsoft.KeyVault/vaults/<key vault name>

//Remember in most cases these ids can be retrieved directly from resource definitions like:

resource "azurerm_resource_group" "rg_object" {
  name = "rg-test"
  location = "westeurope"
}

//Which gives us the rg_id as such:
azurerm_resource_group.rg_object.id //Which can be used directly in the module resource call. OBS. all resources parsed as resource_ids MUST be deployed ahead of time of the module. See the examples for a detailed explanation.
```

### create_ statement, switches used to tell the module to create specific sub resources
1. create_bastion = Creates a bastion host to be used by any VM on the same vnet
2. create_nsg = Creates 1 nsg for the vm subnet
3. create_public_ip = Creates a public ip for each vm specified
4. create_diagnostic_settings = Creates a storage account to hold boot diagnostics information from all vms defined
5. create_kv_for_vms = Creates a kv to be used to store all vm admin passwords as secrets
6. create_kv_role_assignment = Creates RBAC role assignment for the principal in the current Azure context

#### Example of create statements
```hcl
create_bastion = true //Not defining it means not deploying it
create_nsg = true //Not defining it means not deploying it
create_public_ip = true //Not defining it means not deploying it
create_diagnostic_settings = true //Not defining it means not deploying it
create_kv_for_vms = true //Not defining it means not deploying it
create_kv_role_assignment = false //If create_kv_for_vms is set to true, this will automatically be true. The parameter can be used to overwrite the module and not allowing it to create the role assignment "Key Vault Administrator" On the kv. 
```

### mgmt parameters used to define the most backbone pieces of information for the module
1. rg_name = Define a name for the resource group to be deployed
2. location = Define the Azure location of which to deploy to
3. env_name = Define an env to use as prefix on resources being deployed
  - The module can track a multitude of env names as it uses complex regex expressions
  - Also, using the env_name will effect the ip ranges, see <a href="https://github.com/ChristofferWin/codeterraform/tree/main/terraform%20projects/modules/azurerm-vm-bundle#detailed-description">Detailed Description</a> for more information, under "Virtual network"
4. script_name = *Warning* This parameter is experimental and currently not utilized for any purpose

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
    - The structure must be created in a specific order, see the examples for an explanation
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
        - 
    - For all other attributes, see the <a href="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/azurerm-vm-bundle/variables.tf">expanded defintion</a>, search for variable 'kv_object'
    - *Warning* Its only recommended to use this parameter in case a network security rule is to be customized - See the advanced examples for guidance

#### Example of defining custom sub resource objects
```hcl
//We will only define some simple object configurations here. For more information, see the advanced examples
vnet_object = {
  address_space = ["192.168.0.0/20"]
  name = "custom-vnet"
  tags = {
    "environment" = "prod"
  }
}

//You need define 2 subnets in case 'create_bastion = true' The module will always use index 0 for the vm's
//Name is not required and for the bastion subnet it will always be 'AzureBastionSubnet' Regardless of user defined name
subnet_objects = [
  {
    name = "custom-vm-subnet"
    address_prefixes = ["192.168.0.0/22"]
  },
  {
    address_prefixes = ["192.168.10.0/24"]
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

## Return Values
dasdsdasd

## Getting Started
Remember to have read the chapter <a href="https://github.com/ChristofferWin/codeterraform/tree/main/terraform%20projects/modules/azurerm-vm-bundle#prerequisites">Prerequisites</a> before getting started.

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
module "my_first_vm" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=1.0.0" //Always use a specific version of the module

  rg_name = "vm-rg" //Creating a new rg

  vm_linux_objects = [
    {
      name = "ubuntu-vm"
      os_name = "ubuntu"
    }
  ]

  // VNet and VM subnet will also be created.
  // Required dependencies for the vm will also be created.
  // Due to no public subtypes enabled, the VM will only be accessible via its private IP.
  // Refer to the examples section for many more combinations of configurations.
}
```
5. Run terraform init & terraform apply
```hcl
terraform init
terraform apply

//Plan output
Plan: 8 to add, 0 to change, 0 to destroy.

────────────────────────────────────────────────────────────────────────────────── 

//press yes
yes

//apply output
Apply complete! Resources: 8 added, 0 changed, 0 destroyed.
```

6. How it looks in Azure
<img src="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/development/azurerm-vm-bundle/pictures/first-vm-black.png"/>

7. To easily establish a connection, include the following code in your module.
```hcl
module "my_first_vm" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=1.0.0" //Always use a specific version of the module

  rg_name = "vm-rg" //Creating a new rg

  vm_linux_objects = [
    {
      name = "ubuntu-vm"
      os_name = "ubuntu"
    }
  ]

  create_public_ip = true
  create_nsg = true

  // VNet and VM subnet will also be created.
  // Required dependencies for the vm will also be created.
  // Due to no public subtypes enabled, the VM will only be accessible via its private IP.
  // Refer to the examples section for many more combinations of configurations.
}
```
8. Run terraform apply again
```hcl
//Skipping confirm
terraform apply --auto-approve=true

//apply output

Apply complete! Resources: 3 added, 0 changed, 0 destroyed.
```
9. What has been added to Azure
<img src="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/development/azurerm-vm-bundle/pictures/second-vm-black.png" />

10. There is a ton more to explore with the module, see the <a href="https://github.com/ChristofferWin/codeterraform/tree/main/terraform%20projects/modules/azurerm-vm-bundle#examples">Examples</a> for details

## Examples
<b>This section is split into 2 different sub sections:</b>

- <a href="https://github.com/ChristofferWin/codeterraform/tree/main/terraform%20projects/modules/azurerm-vm-bundle#simple-examples---separated-on-topics">Simple examples</a> = Meant to be useful for deployments using default values or for deploying vms where some or all dependencies are already deployed and is instead simply referenced using resource_ids. If in any doubt, please see the <a href="https://github.com/ChristofferWin/codeterraform/tree/main/terraform%20projects/modules/azurerm-vm-bundle#parameters">Parameters</a> section
- Advanced examples = Meant to showcase different combination of resources to deploy with vms

### Simple examples - Separated on topics
1. [How to retrieve required information like os_name](#1-how-to-retrieve-required-information-like-os_name)
2. [A few vms and bastion](#2-a-few-vms-and-bastion)
3. [Using existing virtual vnet and subnet](#3-using-existing-virtual-vnet-and-subnet)


### (1) how to retrieve required information like 'os_name'
```hcl
module "azure_vm_bundle" {
  source                 = "path/to/azurerm-vm-bundle"
  resource_group_name   = "myResourceGroup"
  virtual_machine_count  = 3
  os_type                = "linux"
  os_version             = "Ubuntu 20.04 LTS"
  // Add more configuration as needed
}
```

### (2) A few vms and bastion
```hcl

```

### (3) Using existing Virtual vnet and subnet
```hcl

```

### Advanced examples

#### Deploy 2 windows vms, one needs a specific public ip config, + a few other features
