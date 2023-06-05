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

resource "azurerm_resource_group" "demo_rg_object" {
  name = "demo-rg"
  location = "West Europe"
}

resource "azurerm_virtual_network" "demo_vn_object" {
  name = "demo-vnet"
  location = "West Europe"
  resource_group_name = "demo-rg"
  address_space = ["192.168.0.0/24"]

  depends_on = [ 
        azurerm_resource_group.demo_rg_object
   ]
}