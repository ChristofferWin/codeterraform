//Simply using the newest available provider versions
terraform {
  required_providers {
    azurerm = {
        source = "hashicorp/azurerm"
    }
    local = {
        source = "hashicorp/local"
    }
  }
}

provider "azurerm" {
  features {
  }
}

//The most simple resource to create is a resource group. We need to specify name and location as minimum
//Because we are just getting started no loops or variables are used for simplicity's sake
resource "azurerm_resource_group" "demo_rg_object" {
  name = "demo-rg01"
  location = "West Europe"

  tags = {
    "Environment" = "Demo"
  }
}

resource "azurerm_log_analytics_workspace" "demo_workspace_object" {
  name = "demo-logspace01"
  location = azurerm_resource_group.demo_rg_object.location //Parse the location directly from the resource group object, notice that the output block does not need to be provided
  resource_group_name = azurerm_resource_group.demo_rg_object.name
}

//More than just creating the resources, we can use the local provider to get a json file provided of the returned objects
//Returning the logspace and using the return values to also directly affect the output filename and adding the content via a function called 'jsonencode'
resource "local_file" "log_analytics_workspace_as_local_json" {
  filename = "${azurerm_log_analytics_workspace.demo_workspace_object.name}.json"
  content =  jsonencode(azurerm_log_analytics_workspace.demo_workspace_object) //Must be converted from a complex return object and into a single string
}