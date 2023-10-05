terraform {
  required_providers {
    azurerm = {
      source = "hashicorp/azurerm"
      version = "<=3.64.0"
    }
  }
}

provider "azurerm" {
  features {
  }
}

locals {
  resource_group_names = var.use_defaults ? toset(["mgmt-rg", "${var.environment_type}-rg"]) : var.resource_group_name != null ? toset([var.resource_group_name]) : []
  rg_names = values(azurerm_resource_group.rg_object).*.name

  default_virtual_network_objects = var.use_defaults ? flatten([for i, each in range(2) : [
      {
        name = i == 0 ? "mgmt-vnet" : "${var.environment_type}-vnet"
        address_space = i == 0 ? [var.ip_address_spaces[0]] : var.environment_type == "prod" ? [var.ip_address_spaces[3]] : var.environment_type == "test" ? [var.ip_address_spaces[2]] : [var.ip_address_spaces[1]]
        dns_servers = i == 0 ? null : var.environment_type == "prod" ? [var.virtual_network_objects[0].dns_servers[2]] : [var.virtual_network_objects[0].dns_servers[0], var.virtual_network_objects[0].dns_servers[1]]
        subnets = null //Will be added to later
      }
    ]
  ]) : []
  merge_virtual_network_objects = merge(({for each in local.default_virtual_network_objects : each.name => each}), ({for each in var.virtual_network_objects : each.name => each}))
  virtual_network_objects_without_subnets = [for each in local.merge_virtual_network_objects : each if each.name != "default"]

  default_subnet_objects = var.use_defaults ? flatten([for i, each in range(4) : [
      {
        name = i == 0 ? "bastion-subnet" : i == 1 ? "lan-subnet" : i == 2 ? "serv-subnet" : "dmz-subnet"
        address_prefixes = i == 0 ? cidrsubnet(local.virtual_network_objects[0].address_space[0], 2, 0) : i == 1 && var.environment_type != "dev" ? cidrsubnet(local.virtual_network_objects[1].address_space[0], 8, 128) : i == 1 && var.environment_type == "dev" ? cidrsubnet(local.virtual_network_objects[1].address_space[0], 2, 1) : i == 2 && var.environment_type != "dev" ? cidrsubnet(local.virtual_network_objects[1].address_space[0], 6, 0) : i == 2 && var.environment_type == "dev" ? cidrsubnet(local.virtual_network_objects[1].address_space[0], 2, 0) : i == 3 && var.environment_type != "dev" ? cidrsubnet(local.virtual_network_objects[1].address_space[0], 8, 254) : cidrsubnet(local.virtual_network_objects[1].address_space[0], 2, 3)
      }
    ]
  ]) : []
  merge_subnet_objects = merge(({for each in local.default_subnet_objects : each.name => each}), ({for each in var.subnet_objects : each.name => each}))
  subnet_objects = [for each in local.merge_subnet_objects : each if each.name != "default"]


  virtual_network_objects = [for i, each in local.virtual_network_objects_without_subnets : [
    {
      name = each.name
      address_space = each.address_space
      dns_servers = each.dns_servers
      subnets = null
    }
  ]]
/*
  virtual_network_ids = var.use_defaults ? [for i, each in range(2) : [ //The resource_group_name
    "/subscriptions/${data.azurerm_client_config.my_config.subscription_id}/resourceGroups/${flatten(local.resource_group_names)[i]}/providers/Microsoft.Network/virtualNetworks/${local.virtual_network_objects[i].name}"
  ]] : []

  default_virtual_network_peering_objects = var.use_defaults ? flatten([for i, each in range(2) : [
      {
        name = i == 0 ? "from-${var.environment_type}-to-mgmt" : "from-mgmt-to-${var.environment_type}"
        virtual_network_name = i == 0 ? local.virtual_network_objects[0].name : local.virtual_network_objects[1].name
        remote_virtual_network_id = i == 0 ? local.virtual_network_ids[1] : local.virtual_network_ids[0]
        allow_virtual_network_access = i == 0 ? false : true
      }
    ]
  ]) : [] 
  merge_virtual_network_peering_objects = merge(({for each in local.default_virtual_network_peering_objects : each.name => each}), ({for each in var.virtual_network_peering_objects : each.name => each}))
  virtual_network_peering_objects = [for each in local.merge_virtual_network_peering_objects : each if each.name != "default"]

  default_nsg_objects = var.use_defaults ? flatten([for i, each in range(4) : [
      {
        name = i == 0 ? "mgmt-nsg" : i == 1 ? "lan-nsg" : i == 2 ? "serv-nsg" : "dmz-nsg"
        security_role_objects =  i == 2 || i == 3 ? flatten([for e, each in range(4) : [
            {
              name_rule = e == 0 && i == 2 ? "ALLOW-SQL-FROM-DMZ" : e == 0 && i == 3 ? "ALLOW-SQL-TO-SERV" : e == 1 && i == 2 ? "ALLOW-BASTION-FROM-MGMT" : e == 1 && i == 3 ? "ALLOW-HTTPS-FROM-INTERNET" : e == 2 && i == 2 ? "ALLOW-AD-FROM-LAN" : e == 2 && i == 3 ? "ALLOW-FTP-FROM-INTERNET" : e == 3 ? "DENY-ALL" : null
              priority = e == 0 ? 100 : e == 1 ? 200 : e == 2 ? 300 : 400
              direction = e == 0 && i == 3 ? "Outbound" : "Inbound"
              access = e != 3 ? "Allow" : "Deny"
              protocol = "Tcp"
              source_port_ranges = "*"
              destination_port_ranges = e == 0 ? [1433] : e == 1 ? [443] : e == 2 && i == 2 ? [53, 88, 135, 137, 138, 389, 445, 464, 636, 3268, 3269] : e == 2 && i == 3 ? [20, 21] : null
              source_address_prefixes = e == 0 && i == 2 ? ["${local.subnet_objects[3].address_prefixes}"] : e == 0 && i == 3 ? ["${local.subnet_objects[2].address_prefixes}"] : e == 1 && i == 2 ? ["${local.subnet_objects[0].address_prefixes}"] : e == 1 && i == 2 ? ["*"] : e == 2 && i == 2 ? ["${local.subnet_objects[1].address_prefixes}"] : e == 2 || e == 3 ? ["*"] : null
              destination_address_prefixes = ["*"]
            }
          ]
        ]) : null
      }
    ] 
  ]) : []
  merge_nsg_objects  = merge(({for each in local.default_nsg_objects : each.name => each}), ({for each in var.nsg_objects : each.name => each}))
  nsg_objects = [for each in local.merge_nsg_objects : each if each.name != "default"]

  test = {for each in local.subnet_objects : each.name => each}
  /*
  bastion_objects = use_defaults ? [
    {
      name = "${var.environment_type}-bastion"
      copy_paste_enabled = var.environment_type != "prod" ? true : false
      file_copy_enabled = var.environment_type != "prod" ? true : false
      sku = var.environment_type != "dev" ? "Standard" : "Basic"
      scale_units = var.environment_type != "dev" ? 50 : null
      ip_configuration = [
        {
          name = "bastion"
          subnet_id = azurerm_subnet.subnet_object["${local.subnet_objects[0].name}"].id
          public_ip_address = azurerm_public_ip.pip_object["${var.public_ip_objects[0].name}"].id
        }
      ]
    }
  ] : []
  merge_bastion_objects = merge(({for each in local.default_bastion_objects : each.name => each}), ({for each in var.bastion_objects : each.name => each}))
  #bastion_objects = [for each in local.merge_bastion_objects : each if each.name != "default"]
  */
}

data "azurerm_client_config" "my_config"{}

resource "azurerm_resource_group" "rg_object" {
  for_each = local.resource_group_names
  name = each.key
  location = var.location
}
/*
resource "azurerm_virtual_network" "vnet_object" {
  for_each = {for each in local.virtual_network_objects : each.name => each} 
  name = each.key
  location = var.location
  resource_group_name = length(local.resource_group_names) == 1 ? flatten(local.resource_group_names)[0] : length(regexall("mgmt", each.key)) > 0 ? local.rg_names[1] : local.rg_names[0]
  address_space = each.value.address_space
  dns_servers = each.value.dns_servers

  dynamic "subnet" {
    for_each = var.use_defaults ? virtual_network_objects
    content {
      name = subnet.key
      address_prefix = subnet.value.address_prefix
    }
  }
}
*/
output "tesy" {
  value = local.virtual_network_objects
}