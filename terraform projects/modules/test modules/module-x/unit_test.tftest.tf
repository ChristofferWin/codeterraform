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
  vnet_resource_id = var.vnet_resource_id
  subnet_resource_id = var.subnet_resource_id
  vm_windows_objects = var.vm_windows_objects
  vm_linux_objects = var.vm_linux_objects
}