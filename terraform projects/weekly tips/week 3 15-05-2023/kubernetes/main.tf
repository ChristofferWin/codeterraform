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
  Write_to_screen = "Hello world"
}

output "write_out" {
  value = local.Write_to_screen
}