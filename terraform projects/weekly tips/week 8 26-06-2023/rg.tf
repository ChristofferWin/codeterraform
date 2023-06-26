resource "azurerm_resource_group" "rg_object" {
  name = var.my_resource_group_name
  location = var.location
}