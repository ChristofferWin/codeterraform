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
  rg_object = var.rg_name == null ? {name = split("/", var.rg_id)[4], create_rg = false} : {
    name = var.rg_name 
    create_rg = true
  }

  vnet_object = var.vnet_resource_id == null && var.vnet_object == null ? {
      name = var.env_name != null ? "${var.env_name}-vm-vnet" : "vm-vnet"
      address_space = var.env_name == null ? ["192.168.0.0/24"] : length(regexall("\\b[pP][rR]?[oO]?[dD]?[uU]?[cC]?[tT]?[iI]?[oO]?[nN]?\\b", var.env_name)) > 0 ? ["10.0.0.0/16"] : length(regexall("^\\b[tT][eE]?[sS]?[tT]?[iI]?[nN]?[gG]?\\b$", var.env_name)) > 0 ? ["172.16.0.0/20"] : ["192.168.0.0/24"]
  
  } : var.vnet_object
  
  subnet_objects = var.subnet_objects!= null && var.vnet_resource_id != null ? {for each in var.subnet_objects : each.name => each} : var.subnet_objects == null && var.create_bastion ? {for each in ([{name = "vm-subnet", address_prefixes = [cidrsubnet(local.vnet_object.address_space[0], 1, 0)]},{name = "AzureBastionSubnet", address_prefixes = [cidrsubnet(local.vnet_object.address_space[0], 1, 1)]}]) : each.name => each} : var.subnet_objects != null && var.create_bastion ? {for each in ([for x, y in range(2) : {
      name = x == 1 ? "AzureBastionSubnet" : var.subnet_objects[x].name
      address_prefixes = x == 1 && !can(cidrsubnet(var.subnet_objects[x].address_prefixes[0], 6, 0)) ? ["${split("/", var.subnet_objects[x].address_prefixes[0])[0]}/${split("/", var.subnet_objects[x].address_prefixes[0])[1] - (6 - (32 - split("/", var.subnet_objects[x].address_prefixes[0])[1]))}"] : var.subnet_objects[x].address_prefixes
    }
  ]) : each.name => each} : null

  //Gotta clean up subnets => Too much IF logic in one line, instead => check if the user has inputted subnet_objects first, if yes, check the bastion part
  //Then do the manual adding for the subnets after in case the user has NOT added any subnet objects

  #var.subnet_objects != null && var.vnet_resource_id != null ? {for each in var.subnet_objects : each.name => each} : var.subnet_objects == null && var.create_bastion == false ? {for each in ([{name = "vm-subnet", address_prefixes = [cidrsubnet(local.vnet_object.address_space[0], 1, 0)]}]) : each.name => each} :

  #vnet_resource_id = azurerm_virtual_network.vnet_object != null || azurerm_virtual_network.vnet_object != [] ? azurerm_virtual_network.vnet_object[0].id : var.vnet_resource_id
  #subnet_resource_id = azurerm_subnet.subnet_object != null || azurerm_subnet.subnet_object != [] ? flatten(azurerm_subnet.subnet_object.*.id) : [var.subnet_resource_id]

  //Return objects
  /*
  rg_return_object = can(azurerm_resource_group.rg_object[0]) ? azurerm_resource_group.rg_object[0] : null
  vnet_return_object = can(azurerm_virtual_network.vnet_object[0]) ? azurerm_virtual_network.vnet_object[0] : null
  subnet_return_object = can(azurerm_subnet.subnet_object[0]) ? azurerm_subnet.subnet_object : null
  */
}
/*
resource "azurerm_resource_group" "rg_object" {
  count = local.rg_object.create_rg ? 1 : 0
  name = local.rg_object.name
  location = var.location  
}

resource "azurerm_virtual_network" "vnet_object"{
  count = var.vnet_resource_id == null ? 1 : 0
  name = var.vnet_object.name
  resource_group_name = azurerm_resource_group.rg_object[0].name
  location = var.location
  address_space = local.address_space
}

*/
output "test" {
  value = local.subnet_objects
}

output "vnet" {
  value = local.vnet_object
}