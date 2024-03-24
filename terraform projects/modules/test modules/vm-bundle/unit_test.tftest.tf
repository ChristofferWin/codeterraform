//Define a terraform script file that can be used as "templates" to do multiple unit tests on

//Define required providers (This is actually not necessary for our tests to work as they will automatically be inherrited by the module, but its always good practice to define)
terraform {
  required_providers {
    azurerm = {
      source = "hashicorp/azurerm"
    }
    random = {
      source = "hashicorp/random"
    }
    local = {
      source = "hashicorp/local"
    }
    null = {
      source = "hashicorp/null"
    }
  }
}

provider "azurerm" { //Will use command-line context, typically az cli login 
  features {
  }
}

module "unit_test_1_using_existing_resources" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"   
  rg_id = var.rg_id
  location = var.location
  vnet_resource_id = var.vnet_resource_id
  subnet_resource_id = var.subnet_resource_id
  vm_windows_objects = var.vm_windows_objects_simple
  vm_linux_objects = var.vm_linux_objects_simple
} 

module "unit_and_integration_test_2" { 
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"
  rg_name = var.rg_name
  location = var.location
  vnet_object = var.vnet_object
  subnet_bastion_resource_id = var.subnet_bastion_resource_id
  vm_windows_objects = var.vm_windows_objects_custom_config
  vm_linux_objects = var.vm_linux_objects_custom_config  
}