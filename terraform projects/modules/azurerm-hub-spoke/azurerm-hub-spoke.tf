terraform {
  required_providers {
    azurerm = {
      source = "hashicorp/azurerm"
      version = ">=3.99.0"
    }
  }
}

provider "azurerm" {
  features {
  }
  skip_provider_registration = true
}

locals {
  tp_object = var.typology_object
  location = "westeurope" //To be used as a default location
  vnet_cidr_notation = "/24"
  vnet_cidr_block = ["10.0.0.0/${local.vnet_cidr_notation}"]
  subnets_cidr_notation = "/26"
  #multiplicator = local.tp_object.multiplicator != null ? local.tp_object.multiplicator : 1
  rg_count = 1 + length(local.tp_object.spoke_objects) #* local.multiplicator
  env_name = local.tp_object.env_name != null ? local.tp_object.env_name : ""
  customer_name = local.tp_object.customer_name != null ? local.tp_object.customer_name : ""
  name_fix_pre = local.tp_object.name_prefix != null ? true : false
  name_fix = local.name_fix_pre ? local.name_fix_pre : local.tp_object.name_suffix != null ? false : false
  base_name = local.name_fix == null ? null : local.name_fix && local.tp_object.env_name != null ? "${local.tp_object.name_prefix}-${local.customer_name}-open-${local.env_name}" : local.name_fix == false && local.tp_object.env_name != null ? "${local.env_name}-${local.customer_name}-open-${local.tp_object.name_suffix}" : local.name_fix && local.tp_object.env_name == null ? "${local.tp_object.name_prefix}-${local.customer_name}-open" : local.name_fix == false && local.tp_object.env_name == null && local.tp_object.name_suffix != null ? "${local.customer_name}-open-${local.tp_object.name_suffix}" : null
  rg_name = local.name_fix ? "rg-${replace(local.base_name, "-open", "hub")}" : local.base_name != null ? "${replace(local.base_name, "-open", "hub")}-rg" : "rg-hub"
  vnet_base_name = local.name_fix ? "vnet-${replace(local.base_name, "-open", "hub")}" : local.base_name != null ? "${replace(local.base_name, "-open", "hub")}-vnet" : "vnet-hub"

  rg_objects = {for each in [for a, b in range(local.rg_count) : {
    name = replace((a == local.rg_count - 1 && local.tp_object.hub_object.rg_name != null ? local.tp_object.hub_object.rg_name : local.rg_name != null && a == (local.rg_count - 1) ? local.rg_name : local.tp_object.spoke_objects[a].rg_name != null ? local.tp_object.spoke_objects[a].rg_name : replace(local.rg_name, "hub", "spoke${a + 1}")), "^-.+|.+-$", "/")
    location = local.tp_object.location != null ? local.tp_object.location : a == local.rg_count - 1 && local.tp_object.hub_object.location != null ? local.tp_object.hub_object.location : a != local.rg_count - 1 && local.tp_object.spoke_objects[a].location != null ? local.tp_object.spoke_objects[a].location : local.location
    solution_name = a == local.rg_count -1 ? null : can(local.tp_object.spoke_objects[a].solution_name) ? local.tp_object.spoke_objects[a].solution_name : null
    tags = a == local.rg_count - 1 && local.tp_object.hub_object.tags != null ? local.tp_object.hub_object.tags : a != local.rg_count - 1 ? local.tp_object.spoke_objects[a].tags : null
    vnet_name = a == local.rg_count - 1 && can(local.tp_object.hub_object.network.vnet_name) ? local.tp_object.hub_object.network.vnet_name : a == local.rg_count - 1 ? local.vnet_base_name : a != local.rg_count - 1 && can(local.tp_object.spoke_objects[a].network.vnet_name) ? local.tp_object.spoke_objects[a].network.vnet_name : replace(local.vnet_base_name, "hub", "spoke${a + 1}")
  }] : each.name => each}

  vnet_objects_pre = [for a, b in range(local.rg_count) : {
    name = a == 
    is_hub = a == local.rg_count - 1 ? true : false
    spoke_number = a != local.rg_count -1 ? a : null
    address_spaces = a == local.rg_count - 1 && can(local.tp_object.hub_object.network.address_spaces) ? local.tp_object.hub_object.network.address_spaces : a == local.rg_count - 1 ? ["10.0.0.0/24"] : a != local.rg_count - 1 && can(local.tp_object.spoke_objects[a].network.address_spaces) ? local.tp_object.spoke_objects[a].network.address_spaces : a != local.rg_count - 1 ? [cidrsubnet("10.0.0.0/16", 8, a + 1)] : null
    cidr_notation = local.tp_object.cidr_notation != null ? local.tp_object.cidr_notation : a == local.rg_count - 1 && local.tp_object.hub_object.network == null ? local.subnets_cidr_notation : a == local.rg_count -1 && can(local.tp_object.hub_object.network.vnet_cidr_notation) ? local.tp_object.hub_object.network.vnet_cidr_notation : a != local.rg_count - 1 && local.tp_object.spoke_objects[a].network == null ? local.subnets_cidr_notation : a != local.rg_count - 1 && local.tp_object.spoke_objects[a].network.vnet_cidr_notation != null ? local.tp_object.spoke_objects[a].network.vnet_cidr_notation : local.subnets_cidr_notation
    solution_name = a == local.rg_count -1 ? null : can(local.tp_object.spoke_objects[a].solution_name) ? local.tp_object.spoke_objects[a].solution_name : null
    dns_servers = local.tp_object.dns_servers != null ? local.tp_object.dns_servers : a == local.rg_count - 1 && can(local.tp_object.hub_object.network.dns_servers) ? local.tp_object.hub_object.dns_servers : a != local.rg_count - 1 && can(local.tp_object.spoke_objects[a].network.dns_servers) ? local.tp_object.spoke_objects[a].network.dns_servers : null
    tags = local.tp_object.tags != null && can(local.tp_object.hub_object.network.tags) && a == local.rg_count -1 ? merge(local.tp_object.tags, local.tp_object.hub_object.network.tags) : local.tp_object.tags != null && a != local.rg_count -1 && can(local.tp_object.spoke_objects[a].network.tags) ? merge(local.tp_object.tags, local.tp_object.spoke_objects[a].network.tags) : local.tp_object.tags
    subnets = a == local.rg_count -1 && local.tp_object.hub_object.network == null ? null : a == local.rg_count -1 && can(local.tp_object.hub_object.network.subnet_objects) ? local.tp_object.hub_object.network.subnet_objects : a != local.rg_count -1 && local.tp_object.spoke_objects[a].network == null ? null : a != local.rg_count -1 && can(local.tp_object.spoke_objects[a].network.subnet_objects) ? local.tp_object.spoke_objects[a].network.subnet_objects : null
    ddos_protection_plan = can(local.tp_object.spoke_objects[a].network.ddos_protection_plan) ? local.tp_object.spoke_objects[a].network.ddos_protection_plan : null
  }]

  subnet_objects_pre = [for a, b in local.vnet_objects_pre : {
    subnets = b.subnets != null ?  [for c, d in b.subnets : {
      name = d.name != null ? d.name : b.is_hub ? local.vnet_base_name : replace(replace(local.vnet_base_name, "hub", "spoke${b.spoke_number}"), "vnet", "subnet${c + 1}")
      address_prefixes = d.address_prefixes != null ? d.address_prefixes : d.use_first_subnet != null && d.use_last_subnet == null ? [cidrsubnet(b.address_spaces[0], tonumber(replace(local.subnets_cidr_notation, "/", "")) - tonumber(replace(local.vnet_cidr_notation, "/", "")), c)] : [cidrsubnet(b.address_spaces[0], tonumber(replace(local.subnets_cidr_notation, "/", "")) - tonumber(replace(local.vnet_cidr_notation, "/", "")), pow((32 - tonumber(replace(local.subnets_cidr_notation, "/", "")) - (32 - tonumber(replace(local.vnet_cidr_notation, "/", "")))), 2) -1 -c)] 
      delegation = d.delegation != null ? {
      } : null
    }] : null
  }]

  vnet_objects = {for each in local.vnet_objects_pre : each.name => each}


  ##RETURN OBJECTS
  rg_return_helper_objects = local.rg_return_object != {} ? values(local.rg_return_object) : []
  rg_return_object = azurerm_resource_group.rg_object
  vnet_return_objects = null#azurerm_virtual_network.vnet_object
}

resource "azurerm_resource_group" "rg_object" {
  for_each = local.rg_objects
  name = each.value.solution_name == null ? each.key : replace(each.key, "spoke", "${each.value.solution_name}-spoke")
  location = each.value.location
}

resource "azurerm_virtual_network" "vnet_object" {
  for_each = local.vnet_objects
  name = each.value.solution_name == null ? each.key : replace(each.key, "spoke", "${each.value.solution_name}-spoke")
  location = [for a in local.rg_objects : a.location if a.vnet_name == each.key][0]
  resource_group_name = [for a in local.rg_objects : a.name if a.vnet_name == each.key][0]
  address_space = each.value.address_spaces
  dns_servers = each.value.dns_servers
  tags = each.value.tags
  
  dynamic "ddos_protection_plan" {
    for_each = each.value.ddos_protection_plan != null ? {for a in [each.value.ddos_protection_plan] : a.id => a} : {}
    content {
      id = ddos_protection_plan.key
      enable = ddos_protection_plan.value.enable
    }
  }
}
/*
resource "azurerm_subnet" "subnet_object" {
  name = #something
  location = var.location
  resource_group_name = local.rg_name
  virtual_network_name = #LINK FROM VNETS
  address_prefixes = #something
  service_endpoints = #something
  service_endpoint_policy_ids = #something

  dynamic "delegation" {
    for_each = local.subnet_objects
    content {
      name = #sometnhing

      service_delegation {
        name = #something
        actions = #something
      }
    }
  }
}
*/

output "test2" {
  value = [for each in local.subnet_objects_pre : each.subnets if each.subnets != null]
}