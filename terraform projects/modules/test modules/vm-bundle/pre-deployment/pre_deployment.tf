terraform {
  required_providers {
    azurerm = {
      source = "hashicorp/azurerm"
    }
  }
}

provider "azurerm" { //In line authentication
  features {
  }
}

module "pre_deployment_vnet_subnet" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"
  rg_name = var.rg_name
  location = var.location
  vnet_object = var.vnet_object
  subnet_objects = var.subnet_objects
}

module "pre_deployment_mgmt_resources" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"
  rg_name = var.rg_name
  location = var.location
  vnet_object = var.vnet_object
  subnet_objects = var.subnet_bastion_resource_id   
}