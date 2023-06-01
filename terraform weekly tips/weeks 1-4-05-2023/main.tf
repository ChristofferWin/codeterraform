terraform {
  required_providers {
    azurerm = {
        source = "hashicorp/azurerm" 
    }
    local = {
        source = "hashicorp/local" 
    }
    random = {
        source = "hashicorp/random" 
    }
  }
}

provider "azurerm" {
  features {
  }
  client_id = "<app id>"
  client_secret = var.client_secret
  subscription_id = "<sub id>"
  tenant_id = "<tenant id>"
}

locals {
  location = azurerm_resource_group.demo_rg_object.location
  base_resource_name = split("-", azurerm_resource_group.demo_rg_object.name)[0]
}

resource "azurerm_resource_group" "demo_rg_object" {
  name = "demo-rg"
  location = "west europe"
}

resource "random_string" "storage_random_string_object" {
    length = 3
    min_numeric = 3
}

resource "azurerm_storage_account" "demo_storage_object" {
  name = "${local.base_resource_name}${random_string.storage_random_string_object.result}storage"
  location = local.location
  account_tier = "Standard"
  account_replication_type = "LRS"
  resource_group_name = azurerm_resource_group.demo_rg_object.name
}