# Azure VM Bundle Terraform Module

## Table of Contents

1. [Description](#description)
2. [Prerequisites](#prerequisites)
3. [Versions](#versions)
4. [Resources to Deploy](#resources-to-deploy)
5. [Examples](#examples)

## Description

Welcome to the Azure VM Bundle Terraform module! This module, "azurerm-vm-bundle," is designed for the seamless deployment of Azure virtual machines. It supports both Linux and Windows operating systems across various versions.

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

## Resources to Deploy

To deploy Azure VMs using this module, configure the following:

- Resource Group Name
- Virtual Machine Count
- OS Type
- OS Version

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