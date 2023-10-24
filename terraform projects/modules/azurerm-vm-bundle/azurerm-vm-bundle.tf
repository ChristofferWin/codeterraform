terraform {
  required_providers {
    azurerm = {
        source = "hashicorp/azurerm"
        version = ">=3.76.0"
    }
    null = {
        source = "hashicorp/null"
        version = ">=3.2.1"
    }
    random = {
        source = "hashicorp/random"
        version = ">=3.5.1"
    }
    local = {
        source = "hashicorp/local"
        version = ">=2.4.0"
    }
  }
}

provider "azurerm" {
  features {
    
  }
}

locals {
  //Variable transformation
  rg_object = var.rg_name == null ? {name = split("/", var.rg_id)[4], create_rg = false} : {name = var.rg_name, create_rg = true}

  vnet_object = var.vnet_resource_id == null && var.vnet_object == null ? {
      name = var.env_name != null ? "${var.env_name}-vm-vnet" : "vm-vnet"
      address_space = var.env_name == null ? ["192.168.0.0/24"] : length(regexall("\\b[pP][rR]?[oO]?[dD]?[uU]?[cC]?[tT]?[iI]?[oO]?[nN]?\\b", var.env_name)) > 0 ? ["10.0.0.0/16"] : length(regexall("^\\b[tT][eE]?[sS]?[tT]?[iI]?[nN]?[gG]?\\b$", var.env_name)) > 0 ? ["172.16.0.0/20"] : ["192.168.0.0/24"]
  
  } : var.vnet_object
  
  subnet_objects_pre = var.subnet_objects!= null && var.vnet_resource_id != null && var.create_bastion == false ? {for each in var.subnet_objects : each.name => each} : var.subnet_objects != null && var.create_bastion ? {for each in ([for x, y in range(2) : {
      name = x == 1 ? "AzureBastionSubnet" : var.subnet_objects[x].name
      address_prefixes = x == 1 && !can(cidrsubnet(var.subnet_objects[x].address_prefixes[0], 6, 0)) && can(var.subnet_objects[x].address_prefixes) ? ["${split("/", var.subnet_objects[x].address_prefixes[0])[0]}/${split("/", var.subnet_objects[x].address_prefixes[0])[1] - (6 - (32 - split("/", var.subnet_objects[x].address_prefixes[0])[1]))}"] : can(cidrsubnet(var.subnet_objects[x].address_prefixes[0], 6, 0)) ? [var.subnet_objects[x].address_prefixes[x]] : ["${split("/", local.vnet_object.address_space[0])[0]}/${split("/", local.vnet_object.address_space[0])[1] - (6 - (32 - split("/", local.vnet_object.address_space[0])[1]))}"]
    }
  ]) : each.name => each} : null

  //Figure out a better way to create the automatic bastion subnet in case the user has not defined any manual subnets. as of now, the expression simply splits the current network in 2 which causes the bastion subnet to be way too big in most cases. should always be minimum /26 (if auto)
  subnet_objects = local.subnet_objects_pre == null ? merge(local.subnet_objects_pre, var.subnet_objects == null && var.vnet_resource_id == null && var.create_bastion ? {for each in ([{name = "vm-subnet", address_prefixes = [cidrsubnet(local.vnet_object.address_space[0], 1, 0)]},{name = "AzureBastionSubnet", address_prefixes = ["${split("/", local.vnet_object.address_space[0])[0]}/${split("/", local.vnet_object.address_space[0])[1] - (6 - (32 - split("/", local.vnet_object.address_space[0])[1]))}]}])"]}]) : each.name => each} : {for each in var.subnet_objects : each.name => each}) : local.subnet_objects_pre

  pip_objects = var.create_public_ip && var.pip_objects == null ? {for each in [for x, y in range(local.vm_counter) : {
    //Figure out how to smartest make the name automatically - Please make one for bastion in case
    name = x == 0 && var.env_name != null && var.create_bastion ? "${var.env_name}-bastion-pip" : x == 0 && var.create_bastion ? "bastion-pip" : "${values(local.vm_objects)[x].name}-pip"
    allocation_method = "Static"
    sku = "Standard"
  }] : each.name => each} : !can({for each in var.pip_objects : each.name => each}) ? null : length({for each in var.pip_objects : each.name => each}) - local.vm_counter == 0 ? {for each in var.pip_objects : each.name => each} : {}

  bastion_object = 

  vm_counter_windows = can(length(var.vm_windows_objects)) ? length(var.vm_windows_objects) : 0
  vm_counter_linux = can(length(var.vm_windows_objects)) ? length(var.vm_windows_objects) : 0
  vm_counter = var.create_bastion ? (local.vm_objects) + 1 : (local.vm_objects)

  vm_objects = {
    "test" = {
      name = "test"
    },
    "test2" = {
      name = "test2"
    }
  }

  #rg_resource_id = azurerm_resource_group.rg_object != null || azurerm_resource_group.rg_object != [] ? azurerm_resource_group.rg_object[0].id : var.rg_id
  vnet_resource_id = azurerm_virtual_network.vnet_object != null || azurerm_virtual_network.vnet_object != [] ? azurerm_virtual_network.vnet_object[0].id : var.vnet_resource_id
  subnet_resource_id = azurerm_subnet.subnet_object != null || azurerm_subnet.subnet_object != [] ? flatten(values(azurerm_subnet.subnet_object).*.id) : [var.subnet_resource_id]

  //Return objects
  rg_return_object = can(azurerm_resource_group.rg_object[0]) ? azurerm_resource_group.rg_object[0] : null
  vnet_return_object = can(azurerm_virtual_network.vnet_object[0]) ? azurerm_virtual_network.vnet_object[0] : null
  subnet_return_object = can(azurerm_subnet.subnet_object[0]) ? azurerm_subnet.subnet_object : null
}

resource "azurerm_resource_group" "rg_object" {
  count = local.rg_object.create_rg ? 1 : 0
  name = local.rg_object.name
  location = var.location
}

resource "azurerm_virtual_network" "vnet_object"{
  count = var.vnet_resource_id == null ? 1 : 0
  name = local.vnet_object.name
  resource_group_name = local.rg_object.name
  location = var.location
  address_space = local.vnet_object.address_space
}

resource "azurerm_subnet" "subnet_object" {
  for_each = var.subnet_resource_id == null ? local.subnet_objects : {}
  name = each.key
  resource_group_name = local.rg_object.name
  virtual_network_name = local.vnet_object.name
  address_prefixes = each.value.address_prefixes
}

resource "azurerm_public_ip" "pip_object" {
  for_each = var.create_public_ip ? local.pip_objects : {}
  name = each.key
  resource_group_name = local.rg_object.name
  location = var.location
  allocation_method = each.value.allocation_method
  sku = each.value.sku
}

resource "azurerm_bastion_host" "bastion_object" {
  count = var.create_bastion ? 1 : 0
  name = 
}

output "test" {
  value = local.vm_counter
}