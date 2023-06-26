terraform {
  required_providers {
    azurerm = {
        source = "hashicorp/azurerm"
    }
    local = {
        source = "hashicorp/local"
    }
  }
}

provider "azurerm" {
  features {
  }
}

resource "azurerm_resource_group" "rg01" {
  name = "rg01"
  location = "West Europe"
}

resource "azurerm_virtual_network" "vnet_objects" {
  count = 5
  name = "vnet-${count.index}"
  location = "West Europe"
  address_space = ["10.0.${count.index}.0/24"]
  resource_group_name = azurerm_resource_group.rg01.name

  subnet {
    name = "subnet01"
    address_prefix = "10.0.${count.index}.0/26"
  }

  subnet {
    name = "subnet02"
    address_prefix = "10.0.${count.index}.64/26"
  }

  subnet {
    name = "subnet03"
    address_prefix = "10.0.${count.index}.128/26"
  }

  subnet {
    name = "subnet04"
    address_prefix = "10.0.${count.index}.192/26"
  }
}

output "vnet_objects_raw" {
  value = azurerm_virtual_network.vnet_objects
}

output "all_subnets_from_all_vnets" {
  value = azurerm_virtual_network.vnet_objects.*.subnet
}

output "using_flatten_to_remove_sets" {
  value = flatten(azurerm_virtual_network.vnet_objects.*.subnet)
}

output "combine_all_the_above" {
  value = flatten(azurerm_virtual_network.vnet_objects.*.subnet).*.address_prefix
}

resource "local_file" "vnet_objects_to_json" {
  filename = "vnets_configuration.json"
  content = jsonencode(azurerm_virtual_network.vnet_objects)
}