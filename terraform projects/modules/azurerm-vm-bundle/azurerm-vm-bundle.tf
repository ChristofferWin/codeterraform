terraform {
  required_providers {
    azurerm = {
        source = "hashicorp/azurerm"
        version = ">=3.76.0"
    }
    null = {
        source = "hashicorp/null"
        version = ">=3.2.1"
    }
    random = {
        source = "hashicorp/random"
        version = ">=3.5.1"
    }
    local = {
        source = "hashicorp/local"
        version = ">=2.4.0"
    }
  }
}

provider "azurerm" {
  features {
    
  }
}

locals {
  //Variable transformation
  rg_object = var.rg_name == null ? {name = split("/", var.rg_id)[4], create_rg = false} : {name = var.rg_name, create_rg = true}

  vnet_object_pre = var.vnet_resource_id == null && var.vnet_object == null ? {for each in [for each in range(1) : {
      name = var.env_name != null ? "${var.env_name}-vm-vnet" : "vm-vnet"
      address_space = var.env_name == null ? ["192.168.0.0/24", "192.168.99.0/24"] : length(regexall("\\b[pP][rR]?[oO]?[dD]?[uU]?[cC]?[tT]?[iI]?[oO]?[nN]?\\b", var.env_name)) > 0 ? ["10.0.0.0/16", "10.99.0.0/24"] : length(regexall("^\\b[tT][eE]?[sS]?[tT]?[iI]?[nN]?[gG]?\\b$", var.env_name)) > 0 ? ["172.16.0.0/20", "172.16.99.0/24"] : ["192.168.0.0/24", "192.168.99.0/24"]
  }] : each.name => each} : var.vnet_resource_id != null && var.vnet_object == null ? null : {for each in var.vnet_object : each.name => each}

  vnet_object_helper = values(local.vnet_object_pre)[0]

  subnet_objects_pre = var.subnet_objects!= null && var.vnet_resource_id != null && var.create_bastion == false ? {for each in var.subnet_objects : each.name => each} : var.subnet_objects != null && var.create_bastion ? {for each in ([for x, y in range(2) : {
      name = x == 1 ? "AzureBastionSubnet" : var.subnet_objects[x].name
      address_prefixes = x == 1 && !can(cidrsubnet(var.subnet_objects[x].address_prefixes[0], 6, 0)) && can(var.subnet_objects[x].address_prefixes) ? ["${split("/", var.subnet_objects[x].address_prefixes[0])[0]}/${split("/", var.subnet_objects[x].address_prefixes[0])[1] - (6 - (32 - split("/", var.subnet_objects[x].address_prefixes[0])[1]))}"] : {for each in var.subnet_objects : each.name => each}
    }
  ]) : each.name => each} : null

  subnet_objects = local.subnet_objects_pre == null && var.create_bastion ? {for each in ([{name = "vm-subnet", address_prefixes = [cidrsubnet(local.vnet_object_helper.address_space[0], 1, 0)]},{name = "AzureBastionSubnet", address_prefixes = ["${split("/", local.vnet_object_helper.address_space[0])[0]}/${split("/", local.vnet_object_helper.address_space[0])[1] - (6 - (32 - split("/", local.vnet_object_helper.address_space[0])[1]))}]}])"]}]) : each.name => each} : {for each in [{name = "vm-subnet", address_prefixes = [cidrsubnet(local.vnet_object_helper.address_space[0], 1, 0)]}] : each.name => each}

  pip_objects = var.create_public_ip ? {for each in [for x, y in range(local.vm_counter) : {
    name = x == local.vm_counter && var.env_name != null && var.create_bastion ? "${var.env_name}-bastion-pip" : x == local.vm_counter && var.create_bastion ? "bastion-pip" : can(local.merge_objects[x].public_ip.name) ?  local.merge_objects[x].public_ip.name : "${local.merge_objects[x].name}-pip"
    allocation_method = can(local.merge_objects[x].public_ip.allocation_method) ? local.merge_objects[x].public_ip.allocation_method : "Static"
    sku = can(local.merge_objects[x].public_ip.sku) ? local.merge_objects[x].public_ip.sku : "Standard"
    tags = can(local.merge_objects[x].nic.tags) ? local.merge_objects[x].nic.tags : null
    vm_name = local.merge_objects[x].name
  }] : each.name => each} : {}

  pip_helper_objects = var.create_bastion ? {for each in (flatten([{name = "bastion-pip"},[for each in local.merge_objects : each if each.public_ip != null]])) : each.name => each} : {for each in ([for each in local.merge_objects : each if each.public_ip != null]) : each.name => each}
/*
  pip_objects = local.pip_objects_pre == null ? {for each in [for x, y in range(length(local.merge_objects) - length(local.vm_counter) + sum([for each in local.merge_objects : 1 if each.public_ip != null])) : {
    name = x == 0 && var.env_name != null && var.create_bastion ? "${var.env_name}-bastion-pip" : x == 0 && var.create_bastion ? "bastion-pip" : can(local.merge_objects[x].public_ip.name) ? local.merge_objects[x].public_ip.name : null
    allocation_method = can(local.merge_objects[x].public_ip.allocation_method)

  }] : each.name => each} : local.pip_objects_pre
*/
  bastion_object = var.create_bastion && var.bastion_object == null ? {for each in [for x, y in range(1) : {
      name = var.env_name != null ? "${var.env_name}-bastion-host" : "bastion-host"
      copy_paste_enabled = true
      file_copy_enabled = true
      sku = "Standard" //Otherwise file_copy cannot be enabled
      scale_units = 2
  }] : each.name => each} : var.create_bastion && var.bastion_object != null ? {for each in var.bastion_object : each.name => each} : null

  vm_counter_windows = can(length(var.vm_windows_objects)) ? length(var.vm_windows_objects) : 0
  vm_counter_linux = can(length(var.vm_windows_objects)) ? length(var.vm_windows_objects) : 0
  vm_counter = var.create_bastion ? length((local.vm_objects)) + 1 : length((local.vm_objects))
  vm_os_names = var.vm_windows_objects != null && var.vm_linux_objects != null ? distinct(flatten([[for each in var.vm_windows_objects : each.os_name if each.source_image_reference == null], [for each in var.vm_linux_objects : each.os_name if each.source_image_reference == null]])) : null
  
  vm_objects_pre = [for x, y in range(length(data.local_file.vmskus_objects)) : {
      publisher = jsondecode(data.local_file.vmskus_objects[x].content).Publisher
      offer = jsondecode(data.local_file.vmskus_objects[x].content).Offer
      versions = jsondecode(data.local_file.vmskus_objects[x].content).Versions
      coresAvailable = jsondecode(data.local_file.vmskus_objects[x].content).CoresAvailable
      coresLimit = jsondecode(data.local_file.vmskus_objects[x].content).CoresLimit
      os = split("-", data.local_file.vmskus_objects[x].filename)[0]
    }
  ]

  merge_objects = flatten([var.vm_windows_objects, var.vm_linux_objects])

  vm_objects = {for each in [for x, y in local.merge_objects : {
    name = local.merge_objects[x].name
    publisher = [for each in local.vm_objects_pre : each.publisher if each.os == lower(local.merge_objects[x].os_name)][0]
    offer = [for each in local.vm_objects_pre : each.offer if each.os == lower(local.merge_objects[x].os_name)][0]
    sku = [for each in local.vm_objects_pre : each.versions[0].SKU if each.os == lower(local.merge_objects[x].os_name)][0]
    version = [for each in local.vm_objects_pre : each.versions[0].Versions if each.os == lower(local.merge_objects[x].os_name)][0]
  } if local.merge_objects[x].source_image_reference == null] : each.name => each}

  nic_objects = {for each in [for x, y in local.merge_objects : {
    name = can(local.merge_objects[x].nic.name) ? local.merge_objects[x].nic.name : "${local.merge_objects[x].name}-nic"
    dns_servers = can(local.merge_objects[x].nic.dns_servers) ? local.merge_objects[x].nic.dns_servers : null
    enable_ip_forwarding = can(local.merge_objects[x].nic.enable_ip_forwarding) ? local.merge_objects[x].nic.enable_ip_forwarding : null
    edge_zone = can(local.merge_objects[x].nic.edge_zone) ? local.merge_objects[x].nic.edge_zone : null
    ip_configuration_name = can(local.merge_objects[x].nic.ip_configuration.name) ? local.merge_objects[x].nic.ip_configuration.name : "ip-config"
    private_ip_address_version = can(local.merge_objects[x].nic.ip_configuration.private_ip_address_version) ? local.merge_objects[x].nic.ip_configuration.private_ip_address_version : null
    private_ip_address = can(local.merge_objects[x].nic.ip_configuration.private_ip_address) ? local.merge_objects[x].nic.ip_configuration.private_ip_address : null
    private_ip_address_allocation = can(local.merge_objects[x].nic.ip_configuration.private_ip_address_allocation) ? local.merge_objects[x].nic.ip_configuration.private_ip_address_allocation : "Dynamic"
    tags = can(local.merge_objects[x].nic.tags) ? local.merge_objects[x].nic.tags : null
    vm_name = local.merge_objects[x].name
  }] : each.name => each}
  
  script_name = var.script_name != null && can(file(var.script_name)) ? var.script_name : var.script_name == null ? "${path.module}/Get-AzVMSku.ps1" : null

  rg_resource_id = azurerm_resource_group.rg_object != null || azurerm_resource_group.rg_object != [] ? azurerm_resource_group.rg_object[0].id : var.rg_id
  vnet_resource_id = azurerm_virtual_network.vnet_object != null || azurerm_virtual_network.vnet_object != [] ? flatten(values(azurerm_virtual_network.vnet_object))[0].id : var.vnet_resource_id
  subnet_resource_id = azurerm_subnet.subnet_object != null || azurerm_subnet.subnet_object != [] ? flatten(values(azurerm_subnet.subnet_object).*.id) : [var.subnet_resource_id]
  pip_resource_id = azurerm_public_ip.pip_object != null || azurerm_public_ip.pip_object != [] ? flatten(values(azurerm_public_ip.pip_object).*.id) : null

  //Return objects
  rg_return_object = can(azurerm_resource_group.rg_object[0]) ? azurerm_resource_group.rg_object[0] : null
  vnet_return_object = azurerm_virtual_network.vnet_object != null || azurerm_virtual_network.vnet_object != {} ? azurerm_virtual_network.vnet_object : null
  subnet_return_object = azurerm_subnet.subnet_object != null || azurerm_subnet.subnet_object != {} ? azurerm_subnet.subnet_object : null
}

resource "null_resource" "ps_object" {
  count = can(length(local.vm_os_names)) && local.script_name != null ? length(local.vm_os_names) : 0
  provisioner "local-exec" {
        command = "${path.module}/${local.script_name} -Location ${var.location} -OS ${local.vm_os_names[count.index]} -OutputFileName ${local.vm_os_names[count.index]}-skus.json"
        interpreter = ["pwsh.exe","-Command"]
  }
}

data "local_file" "vmskus_objects" {
  count = length(local.vm_os_names)
  filename = "${local.vm_os_names[count.index]}-skus.json"

  depends_on = [ null_resource.ps_object ]
}

resource "random_password" "vm_password_object" {
  count = local.vm_counter - 1 //Regardless of whether the user wants to supply own passwords, create a list of passwords ready, in case not
  length           = 16
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

resource "azurerm_resource_group" "rg_object" {
  count = local.rg_object.create_rg ? 1 : 0
  name = local.rg_object.name
  location = var.location
}

resource "azurerm_virtual_network" "vnet_object"{
  for_each = var.vnet_resource_id == null ? local.vnet_object_pre : {}
  name = each.key
  resource_group_name = local.rg_object.name
  location = var.location
  address_space = each.value.address_space
  tags = can(each.value.tags) ? each.value.tags : null

  depends_on = [ azurerm_resource_group.rg_object ]
}

resource "azurerm_subnet" "subnet_object" {
  for_each = var.subnet_resource_id == null ? local.subnet_objects : {}
  name = each.key
  resource_group_name = local.rg_object.name
  virtual_network_name = local.vnet_object_helper.name
  address_prefixes = each.value.address_prefixes

  depends_on = [ azurerm_virtual_network.vnet_object ]
}

resource "azurerm_public_ip" "pip_object" {
  for_each = local.pip_objects
  name = each.key
  resource_group_name = local.rg_object.name
  location = var.location
  allocation_method = each.value.allocation_method
  sku = each.value.sku
  tags = can(each.value.tags) ? each.value.tags : null

  depends_on = [ azurerm_subnet.subnet_object ]
}

resource "azurerm_bastion_host" "bastion_object" {
  for_each = var.create_bastion ? local.bastion_object : {}
  name = each.key
  resource_group_name = local.rg_object.name
  location = var.location
  copy_paste_enabled = each.value.copy_paste_enabled
  file_copy_enabled = each.value.file_copy_enabled
  sku = each.value.sku
  tags = can(each.value.tags) ? each.value.tags : null

  ip_configuration {
    name = "ip-config"
    subnet_id = [for each in local.subnet_resource_id : each if length(regexall("Bastion", each)) > 0][0]
    public_ip_address_id = [for each in local.pip_resource_id : each if length(regexall("bastion", each)) > 0][0]
  }

  depends_on = [ azurerm_public_ip.pip_object ]
}

resource "azurerm_network_interface" "nic_object" {
  for_each = local.nic_objects
  name = each.key
  resource_group_name = local.rg_object.name
  location = var.location
  dns_servers = each.value.dns_servers
  enable_ip_forwarding = each.value.enable_ip_forwarding
  edge_zone = each.value.edge_zone
  tags = each.value.tags
  
  ip_configuration {
    name = each.value.ip_configuration_name
    private_ip_address_allocation = each.value.private_ip_address_allocation
    private_ip_address = each.value.private_ip_address
    #public_ip_address_id = local.pip_objects != {} ? [for each in local.pip_resource_id : each if length(regexall(([for each in local.pip_objects : each.name if each.value.vm_name == each.vm_name][0]), each)) > 0][0] : null
  }
}
/*
resource "azurerm_windows_virtual_machine" "vm_windows_object" {
  for_each = can{for each in var.vm_windows_objects : each.name => each} ? for each in var.vm_windows_objects : each.name => each : {}
  name = each.key
  resource_group_name = local.rg_object.name
  location = var.location
  
}
*/
output "counter" {
  value = local.vm_counter
}

output "subnet_objects" {
  value = local.pip_helper_objects
}