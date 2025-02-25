terraform {
  required_providers {
    azurerm = {
        source = "hashicorp/azurerm"
    }
  }
}

provider "azurerm" {
}

locals {
  location = values(azurerm_resource_group.rg_object)[0].location
  location_mgmt = values(azurerm_resource_group.rg_object)[1].location
  rg_name_mgmt = values(azurerm_resource_group.rg_object)[0].name
  rg_name = values(azurerm_virtual_network.vnet_object)[1].name
  tenant_id = data.azurerm_client_config.client_config.tenant_id
  snet_mgmt_id = azurerm_virtual_network.vnet_object.subnet[0].id
  snet_id = azurerm_virtual_network.vnet_object.vnet_object[1].id

}

data "azurerm_client_config" "client_config" {
}

//Management resources
resource "azurerm_resource_group" "rg_object" {
  for_each = {for each in var.rg_objects : each.name => each}
  name = each.key
  location = each.value.location
  tags = each.value.tags
}

resource "azurerm_key_vault" "kv_object" {
  name = var.kv_object.name
  location = local.location
  resource_group_name = local.rg_name_mgmt
  tenant_id = data.azurerm_client_config.client_config.tenant_id
  sku_name = var.kv_object.sku
  soft_delete_retention_days = var.kv_object.soft_delete_days
  enable_rbac_authorization = true
  network_acls {
    bypass = "AzureServices"
    default_action = "Deny"
    virtual_network_subnet_ids = [azurerm_virtual_network.vnet_object.subnet[0].id]
  }
}

//Networking resources
resource "azurerm_virtual_network" "vnet_object" {
  name = var.vnet_object.name
  location = local.location
  resource_group_name = local.rg_name_mgmt
  address_space = var.vnet.address_space
  subnet = var.vnet_object.subnets
}

resource "azurerm_private_dns_zone" "p_dns_zone" {
  for_each = {for each in var.p_dns_zones : each.name => each}
  name = each.key
  resource_group_name = local.rg_name_mgmt
}

//Web
resource "azurerm_service_plan" "service_plan_object" {
    name = var.web_object.service_plan_name
    resource_group_name = local.rg_name
    location = local.location
    os_type = var.web_object.os_type
    sku_name = var.web_object.sku
}

resource "azurerm_linux_web_app" "linux_web_object" {
  name = "dsa"
  location = "ada"
  resource_group_name = "sadda"
  service_plan_id = azurerm_service_plan.service_plan_object.id
  public_network_access_enabled = false
  virtual_network_subnet_id = local.snet_id
  

  site_config {
    application_stack {
      go_version = "1.19"
    }
  }
}


