# Azure VM Bundle Terraform Module

## Table of Contents

1. [Description](#description)
2. [Detailed Description](#detailed-description)
3. [Prerequisites](#prerequisites)
4. [Versions](#versions)
5. [Parameters](#parameters)
6. [Return Values](#return-values)
7. [Examples](#examples)

## Description

Welcome to the Azure VM Bundle Terraform module! The "azurerm-vm-bundle" module facilitates the effortless deployment of Azure virtual machines, accommodating both Linux and Windows operating systems across multiple versions. This capability is achieved through the integration of a PowerShell module, available in the same repository at <a href="https://github.com/ChristofferWin/codeterraform/tree/main/powershell%20projects/modules">Get-AzVMSku</a>. This PowerShell module aids in retrieving essential information such as SKU, SKU version, and more.

The module boasts extensive configuration flexibility, supporting a wide array of customization options. For a comprehensive overview of these options, please refer to the detailed description provided.

Furthermore, the module is more than capable of deploying various sub-types that is typically used together with Azure virtual machines. 

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
- Installed Terraform (download [here](https://www.terraform.io/downloads.html))
- Azure CLI installed for authentication

## Versions

The table below outlines the compatibility of the module:

| Terraform Version | Azure Provider Version | Module Version |
| ------------------ | ---------------------- | -------------- |
| 0.15 and above    | 2.0 and above          | 1.0            |

For the latest updates, check the [releases](https://github.com/your-username/azurerm-vm-bundle/releases) page.

## Parameters
If using VScode, make use of the extension for terraform from Hashicorp and thereby getting access to 'Intellisense'

<img src="https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/modules/development/azurerm-vm-bundle/pictures/gifs/Intellisense1.gif"/>

## Return Values
dasdsdasd

## Getting Started


## Examples

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

## Detailed_Description2