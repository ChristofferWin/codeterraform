provider "azurerm" {
  features {
  }
}

module "test" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"
  rg_id = var.rg_id
  vnet_resource_id = var.vnet_resource_id
  subnet_bastion_resource_id = var.subnet_bastion_resource_id
  create_bastion = true
  create_public_ip = true
  create_nsg = true
  vm_windows_objects = var.vm_windows_objects
  vm_linux_objects = var.vm_linux_objects
}

data "azurerm_subnet" "data_subnet_object" {
  count = 2
  name = count.index == 0 ? split("/", var.subnet_resource_id)[10] : split("/", var.subnet_bastion_resource_id)[10]
  virtual_network_name = split("/", var.vnet_resource_id)[8]
  resource_group_name = split("/", var.rg_id)[4]
}
 
output "test" {
  value = data.azurerm_subnet.data_subnet_object[0].address_prefixes
}

output "test2" {
  value = [for each in data.azurerm_subnet.data_subnet_object : each if each.name == "AzureBastion"][0].address_prefixes
}