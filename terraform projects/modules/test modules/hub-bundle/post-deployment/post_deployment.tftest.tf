provider "azurerm" {
  features {
  }
}

data "azurerm_virtual_network" "data_vnet_objects" {
  for_each = var.vnet_objects
  name = each.key
  resource_group_name = each.value.resource_group_name
}