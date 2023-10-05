terraform {
  required_providers {
    azuread = {
        source = "hashicorp/azuread"
    }
    azurerm = {
        source = "hashicorp/azurerm"
    }
  }
}

provider "azuread" {
}

provider "azurerm" {
  features {
    
  }
}

resource "azuread_application" "test" {
  display_name = "test"
}

data "azurerm_client_config" "example" {
}

resource "azurerm_role_assignment" "test" {
  role_definition_name = "contributor"
  principal_id = "decc441b-42ab-44ff-bc42-3dd6ffc8dabc"
  scope = "/subscriptions/7c97469d-50f5-4060-b931-772d59bee884"
  
  depends_on = [ azuread_application.test ]
}