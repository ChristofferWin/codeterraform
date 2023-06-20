terraform {
  required_providers {
    azurerm = {
        source = "hashicorp/azurerm"
    }
  }
}

provider "azurerm" {
  features {
  }
}

locals {
  environments = {
    "Dev" = {
        name = "Dev"
        location = "westeurope"
        ip_range_vnet = ["192.168.0.0/24"]
        ip_range_subnets = ["192.168.0.0/28", "192.168.0.16/28", "192.168.0.32/28", "192.168.0.48/28", "192.168.0.64/28", "192.168.0.80/28", "192.168.0.96/28", "192.168.0.112/28", "192.168.0.128/28"]
    },
    "Test" = {
        name = "Test"
        location = "westeurope"
        ip_range_vnet = ["172.16.0.0/16"]
        ip_range_subnets = ["172.16.0.0/24", "172.16.1.0/24", "172.16.2.0/24", "172.16.3.0/24", "172.16.4.0/24", "172.16.5.0/24", "172.16.6.0/24", "172.16.7.0/24", "172.16.8.0/24"]
    },
    "Preprod" = {
        name = "Preprod"
        location = "westeurope"
        ip_range_vnet = ["10.0.0.0/16"]
        ip_range_subnets = ["10.0.0.0/20", "10.0.16.0/20", "10.0.32.0/20", "10.0.48.0/20", "10.0.64.0/20", "10.0.80.0/20", "10.0.96.0/20", "10.0.112.0/20", "10.0.128.0/20"]Rule = ""
      }
    }
  }
}

resource "azurerm_resource_group" "rg_objects" {
  for_each = local.environments
  name = "${each.value.name}-rg" //Can also use each.key here
  location = each.value.location
}

resource "azurerm_virtual_network" "vn_objects" {
  for_each = local.environments
  name = "${each.value.name}-vnet"
  location = each.value.location
  resource_group_name = azurerm_resource_group.rg_objects[each.key].name
  address_space = each.value.ip_range_vnet
}

resource "azurerm_subnet" "subnet_objects_dev" {
  count = length(local.environments.Dev.ip_range_subnets)
  name = "${replace(azurerm_virtual_network.vn_objects["Dev"].name, "vnet", "subnet${count.index}")}"
  resource_group_name = azurerm_resource_group.rg_objects["Dev"].name
  virtual_network_name = azurerm_virtual_network.vn_objects["Dev"].name
  address_prefixes = [local.environments.Dev.ip_range_subnets[count.index]]
}

resource "azurerm_subnet" "subnet_objects_test" {
  count = length(local.environments.Test.ip_range_subnets)
  name = "${replace(azurerm_virtual_network.vn_objects["Test"].name, "vnet", "subnet${count.index}")}"
  resource_group_name = azurerm_resource_group.rg_objects["Test"].name
  virtual_network_name = azurerm_virtual_network.vn_objects["Test"].name
  address_prefixes = [local.environments.Test.ip_range_subnets[count.index]]
}

resource "azurerm_subnet" "subnet_objects_preprod" {
  count = length(local.environments.Preprod.ip_range_subnets)
  name = "${replace(azurerm_virtual_network.vn_objects["Preprod"].name, "vnet", "subnet${count.index}")}"
  resource_group_name = azurerm_resource_group.rg_objects["Preprod"].name
  virtual_network_name = azurerm_virtual_network.vn_objects["Preprod"].name
  address_prefixes = [local.environments.Preprod.ip_range_subnets[count.index]]
}