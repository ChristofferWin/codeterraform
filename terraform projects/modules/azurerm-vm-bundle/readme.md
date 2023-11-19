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

Maybe you want a simple test environment with a few virtual machines and what about all their dependencies?

<b>Example deployment of 2 Linux machines with public ips and ssh setup:</b>
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
| azurerm | <a href="https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs">hashicorp/azurerm</a>  | 3.76.0            |
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
To solve it, simply remove the version parameter:
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
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=0.9.0-beta" //Always use a specific version of the module

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
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=0.9.0-beta" //Always use a specific version of the module

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

## Examples simple

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

## Examples advanced
This is the section for any advanced examples using the module