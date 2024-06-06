output "return_vnet_and_subnet_objects" {
  value = values(data.azurerm_virtual_network.data_vnet_objects)
}