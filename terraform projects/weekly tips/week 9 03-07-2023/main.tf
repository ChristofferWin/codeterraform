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
  name_prefix = "${var.environment_type}-${var.name_prefix}"
  rg_name = azurerm_resource_group.rg_object.*.name
  calculate_ip_address_space_prod = var.environment_type == "prod" ? [cidrsubnet(var.ip_address_space[0], 8, 1), cidrsubnet(var.ip_address_space[0], 8, 99)]: null
  calculate_ip_address_space_nonprod = var.environment_type == "test" ? [var.ip_address_space[1]] : [var.ip_address_space[2]]
  ip_address_space = local.calculate_ip_address_space_prod != null ? local.calculate_ip_address_space_prod : local.calculate_ip_address_space_nonprod
}

resource "azurerm_resource_group" "rg_object" {
  count = var.environment_type == "prod" ? 2 : 1
  name = count.index == 1 ? "${local.name_prefix}-mgmt-rg" : "${local.name_prefix}-rg"
  location = var.location

  tags = {
    environment = var.environment_type
  }
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

resource "azurerm_virtual_network" "vn_objects" {
  count = var.environment_type == "prod" ? 2 : 1
  name = count.index == 1 ? "${local.name_prefix}-mgmt-vnet" : "${local.name_prefix}-vnet"
  resource_group_name = count.index == 1 ? local.rg_name[1] : local.rg_name[0]
  location = var.location
  address_space = [local.ip_address_space[count.index]]

  dynamic "subnet" {
    for_each = var.environment_type == "prod" ? {for ip in [local.ip_address_space[count.index]] : ip => ip} : {}
    content {
      name = count.index == 0 ? "DMZ" : "GatewaySubnet"
      address_prefix = "${cidrsubnet(subnet.key, 1, 0)}"
    }
  }

  tags = {
    environment = var.environment_type
  }
}

resource "azurerm_public_ip" "pip_object" {
  count = var.environment_type == "prod" ? 1 : 0
  name = "${local.name_prefix}-gw-pip"
  location = var.location
  resource_group_name = local.rg_name[1]
  sku = "Standard"
  allocation_method = "Static"

  tags = {
    environment = var.environment_type
  }
}

resource "azurerm_virtual_network_gateway" "gw_object" {
  count = var.environment_type == "prod" ? 1 : 0
  name = "${local.name_prefix}-gw"
  location = var.location
  resource_group_name = local.rg_name[1]
  sku = "Basic"
  type = "Vpn"
  private_ip_address_enabled = true

  ip_configuration {
    name = "ip_config"
    private_ip_address_allocation = "Static"
    subnet_id = flatten([for each in flatten(azurerm_virtual_network.vn_objects.*.subnet) : each if each.name == "GatewaySubnet"])[count.index].id
    public_ip_address_id = azurerm_public_ip.pip_object[count.index].id
  }

  tags = {
    environment = var.environment_type
  }
}