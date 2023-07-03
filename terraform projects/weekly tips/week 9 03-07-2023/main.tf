terraform {
  required_providers {
    azurerm = {
        source = "hashicorp/azurerm"
    }
    azuread = {
        source = "hashicorp/azuread"
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
}