//Defining the provider 'azurerm' that we are going to use
//By not defining the version attribute terraform will always pull down the latest
terraform {
  required_providers {
    azurerm = {
        source = "hashicorp/azurerm"
    }
  }
}

//Because we are using the most simpel form of authentication, only the features attribute (required) is added
provider "azurerm" {
  features {
  }
}

//The most simple resource to create is a resource group. We need to specify name and location as minimum
//Because we are just getting started no loops or variables are used for simplicity's sake
resource "azurerm_resource_group" "my_first_rg" {
  name = "my-first-rg"
  location = "West Europe"
}

//If we dont provide an output definition terraform will not return any output after the plan has been provided other than a status of the deployment
output "rg" {
  value = azurerm_resource_group.my_first_rg.name
}