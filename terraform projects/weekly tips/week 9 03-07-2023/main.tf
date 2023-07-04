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

locals {
  name_prefix = "${var.environment_type}-${var.name_prefix}"
  #rg_name = length(azurerm_resource_group.rg_object) > 1 ? [azurerm_resource_group.rg_object[0].name] : azurerm_resource_group.rg_object.*.name
  split = replace(var.ip_address_space[0], "^[0-9]+(?=\\.[0-9]+\\/)$", "99")
}
/*
resource "azurerm_resource_group" "rg_object" {
  count = var.environment_type == "prod" ? 2 : 1
  name = count.index == 1 ? "${local.name_prefix}-mgmt-rg" : "${local.name_prefix}-${var.environment_type}-rg"
  location = var.location
}

resource "azurerm_log_analytics_workspace" "logspace_object" {
  name                = "${local.name_prefix}-logspace"
  location            = var.location
  resource_group_name = local.rg_name[0]
  sku                 = "PerGB2018"
  allow_resource_only_permissions = var.environment_type == "prod" ? true : false
  local_authentication_disabled = var.environment_type == "prod" ? true : false
  retention_in_days = var.environment_type == "prod" ? 120 : 90
  daily_quota_gb = var.environment_type != "prod" ? 5 : null
  tags = {
    environment = var.environment_type
  }
}
/*
resource "azurerm_virtual_network" "vn_objects" {
  count = var.environment_type == "prod" ? 2 : 1
  name = count.index == 1 ? "${local.name_prefix}-mgmt-rg" : "${local.name_prefix}-${var.environment_type}-vnet"
  resource_group_name = local.rg_name[count.index]
  address_space = count.index == 1 ? s
}

/*
resource "azurerm_virtual_network_gateway" "gw_object" {
  count = var.environment_type == "prod" ? 1 : 0
  name = "${local.name_prefix}-gw"
  location = var.location
  resource_group_name = local.rg_name[0]
  sku = "Basic"
  type = "Vpn"
  private_ip_address_enabled = true

  ip_configuration {
    name = "ip_config"
    private_ip_address_allocation = "Static"
    subnet_id = 
  }
}
*/
output "test" {
 value = local.split
}