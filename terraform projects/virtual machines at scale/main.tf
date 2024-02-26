terraform {
  required_providers {
    azurerm = {
      source = "hashicorp/azurerm"
    }
    random = {
      source = "hashicorp/random"
    }
    local = {
      source = "hashicorp/local"
    }
    null = {
      source = "hashicorp/null"
    }
  }
}

locals {
  mgmt_address_space = ["10.10.0.0/24"]
  vnet = module.mgmt_resources.vnet_object
  env_prod = var.environment_objects.env_prod
  env_dev = var.environment_objects.env_dev
}

provider "azurerm" {
  features {
    
  }
}

module "mgmt_resources" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"
  create_bastion = true
  rg_name = "mgmt-rg"
  rg_tags = {
    "demo" = "mgmt"
  }

  vnet_object = {
    name = "mgmt-vnet"
    address_space = local.mgmt_address_space
  }

  subnet_objects = [
    {
      address_prefixes = [cidrsubnet(local.mgmt_address_space[0], 2, 2)]
    }
  ]
}

module "production_resources" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"
  rg_name = "${local.env_prod.name}-rg"
  location = local.env_prod.location
  env_name = local.env_prod.name
  create_kv_for_vms = true
  create_kv_role_assignment = true
  create_diagnostic_settings = true

  rg_tags = {
    "demo" = "prod"
  }

  vnet_object = {
    name = "${local.env_prod.name}-vm-vnet"
    address_space = local.env_prod.address_space
  }

  subnet_objects = [
    {
      name = "${local.env_prod.name}-vm-subnet"
      address_prefixes = [cidrsubnet(local.env_prod.address_space[0], 4, 0)] # /20
    }
  ]

  nsg_objects = [
    {
      name = "${local.env_prod.name}-vm-nsg"
      no_rules = true
    }
  ]

  vm_windows_objects = flatten([for vm in range(local.env_prod.number_of_windows_vms) : [
    {
      name = "${local.env_prod.name}-app-win-${vm + 1}" //To not use index 0 in a name
      admin_username = "${local.env_prod.name}admin"
      size_pattern = local.env_prod.size_pattern
      os_name = vm == 0 ? "server2022" : vm == 1 ? "server2019" : "windows11"
    }
  ]])

  vm_linux_objects = flatten([for vm in range(local.env_prod.number_of_linux_vms) : [
    {
      name = "${local.env_prod.name}-app-linux-${vm + 1}"
      admin_username = "${local.env_prod.name}admin"
      size_pattern = local.env_prod.size_pattern
      os_name = vm == 1 ? "Ubuntu" : "CentOS"
    }
  ]])
}

module "development_resources" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"
  rg_name = "${local.env_dev.name}-rg"
  env_name = local.env_dev.name
  create_kv_for_vms = true
  create_kv_role_assignment = true
  create_diagnostic_settings = true
  create_public_ip = true


  rg_tags = {
    "demo" = "dev"
  }

  vnet_object = {
    name = "${local.env_dev.name}-vm-vnet"
    address_space = local.env_dev.address_space
  }

  subnet_objects = [
    {
      name = "${local.env_dev.name}-vm-subnet"
      address_prefixes = [cidrsubnet(local.env_dev.address_space[0], 1, 0)] # /25
    }
  ]

  nsg_objects = [
    {
      name = "${local.env_dev.name}-vm-nsg"
    }
  ]

  vm_windows_objects = flatten([for vm in range(local.env_dev.number_of_windows_vms) : [
    {
      name = "${local.env_dev.name}-app-win-${vm + 1}" //To not use index 0 in a name
      admin_username = "${local.env_dev.name}admin"
      os_name = vm == 0 ? "server2022" : "server2012r2" //Only 2 Windows servers for Dev
    }
  ]])

  vm_linux_objects = flatten([for vm in range(local.env_dev.number_of_linux_vms) : [
    {
      name = "${local.env_dev.name}-app-linux-${vm + 1}"
      admin_username = "${local.env_dev.name}admin"
      os_name = "debian11" //Only 1 linux VM for Dev
    }
  ]])

  depends_on = [ module.production_resources ]
}

resource "azurerm_virtual_network_peering" "peering_mgmt_env_object" {
  for_each = var.environment_objects
  name = "from-mgmt-to-${each.value.name}"
  resource_group_name = module.mgmt_resources.rg_object.name
  virtual_network_name = values(module.mgmt_resources.vnet_object)[0].name
  remote_virtual_network_id = each.value.name == "prod" ? values(module.production_resources.vnet_object)[0].id : values(module.development_resources.vnet_object)[0].id
  allow_virtual_network_access = true
  allow_forwarded_traffic = false
  allow_gateway_transit = true
}

resource "azurerm_virtual_network_peering" "peering_env_mgmt_object" {
  for_each = var.environment_objects
  name = "from-${each.value.name}-to-mgmt"
  resource_group_name = each.value.name == "prod" ? module.production_resources.rg_object.name : module.development_resources.rg_object.name
  virtual_network_name = each.value.name == "prod" ? values(module.production_resources.vnet_object)[0].name : values(module.development_resources.vnet_object)[0].name 
  remote_virtual_network_id = values(module.mgmt_resources.vnet_object)[0].id
  allow_virtual_network_access = true
  allow_forwarded_traffic = true
  allow_gateway_transit = false
}