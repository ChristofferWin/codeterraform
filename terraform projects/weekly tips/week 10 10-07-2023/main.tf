/*
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

locals {
  rg_names = flatten(azurerm_resource_group.rg_object.*.name)
}

resource "azurerm_resource_group" "rg_object" {
  count = 2
  name = count.index == 0 ? "${var.environment_name}-rg" : "${var.environment_name}-mgmt-rg"
  location = var.location
}

*/