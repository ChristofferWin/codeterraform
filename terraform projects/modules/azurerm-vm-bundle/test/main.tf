provider "azurerm" {
  features {
  }
}

module "test" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"
  rg_id = var.rg_id
  location = var.location
  vnet_resource_id = var.vnet_resource_id
  create_public_ip = true
  create_nsg = true
  subnet_objects = var.subnet_object
  vm_windows_objects = var.vm_windows_objects
  vm_linux_objects = var.vm_linux_objects
}