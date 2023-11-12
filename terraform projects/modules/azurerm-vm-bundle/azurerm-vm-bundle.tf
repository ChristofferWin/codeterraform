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

##############################################################################################
######################################### NOTES ##############################################
##                                                                                          ##
##  Date: 30-10-2023                                                                        ## 
##  State: All bugs are fixed for all working compontents including creating win vm's       ##
##  Missing: Define the block for linux vm´s and test many different deployment scenarios   ##
##  Improvements (1): Find solution for when JSON payload contain invalid image             ##
##  =||= (2): Make it dynamically possible to not run the file data block                   ## 
##  =||= (3): To (2), make it possible to parse a filepath for custom JSON payloads         ##
##  Future improvements: Clean up the locals block & add comment sections in code           ##
##  Comments: First entry                                                                   ##
##------------------------------------------------------------------------------------------##
##                                                                                          ##
##  Date: 08-11-2023                                                                        ## 
##  State: Linux VMs has been defined & fixed a bug in the PS script attached to the code   ##
##  Missing: Create a resource definition for nsgs to set on subnets & extensive tests      ##
##  Improvements (1): Since last update the solution for json payload scenarios are added   ##                                                                     
##  =||= (2): Wont be solved, if TF needs to read a file, let it                            ## 
##  =||= (3): Need to decide whether I should allow custom json payloads                    ##
##  Future improvements: Clean up the locals block & add comment sections in code           ##
##  Comments: Module is pretty far at this stage, around 80% done                           ##
##------------------------------------------------------------------------------------------##
##                                                                                          ##
##  Date: 09-11-2023                                                                        ## 
##  State: All resources was though to defined and extensive testing has begun, but turns   ##
##  out I need to define all possible resources directly in the module to avoid error:      ##
##  'The "for_each" map includes keys derived from resource attributes...                   ##
##  Missing: Create storage account, kv      ##
##  Improvements (1): Since last update the solution for json payload scenarios are added   ##                                                                     
##  =||= (2): Wont be solved, if TF needs to read a file, let it                            ## 
##  =||= (3): Need to decide whether I should allow custom json payloads                    ##
##  Future improvements: Clean up the locals block & add comment sections in code           ##
##  Comments: Module is pretty far at this stage, around 80% done                           ##      ## // ADD COOL OUTPUT / FORMAT SO ITS EASY TO CONNECT AFTER RESOURCES ARE DONE
##                                                                                          ##
##############################################################################################

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

  subnet_objects = local.subnet_objects_pre == null && var.create_bastion ? {for each in ([{name = "vm-subnet", address_prefixes = [cidrsubnet(local.vnet_object_helper.address_space[0], 1, 0)]},{name = "AzureBastionSubnet", address_prefixes = [local.vnet_object_helper.address_space[1]]}]) : each.name => each} : {for each in [{name = "vm-subnet", address_prefixes = [cidrsubnet(local.vnet_object_helper.address_space[0], 1, 0)]}] : each.name => each}

  nsg_objects_pre = !can(length(var.nsg_objects)) && var.create_nsg ? 1 : can(length(var.nsg_objects)) ? length(var.nsg_objects) : 0
  nsg_objects_rules_pre = can(var.nsg_objects.*.security_rules) ? length(flatten(var.nsg_objects.*.security_rules)) : 1
  
  nsg_objects = local.nsg_objects_pre > 0 ? {for a in [for b, c in range(local.nsg_objects_pre) : {
    name = can(var.nsg_objects[b].name) ? var.nsg_objects[b].name : var.env_name != null ? "${var.env_name}-vm-nsg" : "vm-nsg"
    subnet_id = can(var.nsg_objects[b].subnet_id) ? var.nsg_objects[b].subnet_id : var.subnet_resource_id != null ? var.subnet_resource_id : [for each in local.subnet_resource_id : each if length(regexall("vm", each)[0]) > 0][0]
    tags = can(var.nsg_objects[b].tags) ? var.nsg_objects[b].tags : null

    security_rules = {for d in [for e, f in range(local.nsg_objects_rules_pre) : { //
      name = can(var.nsg_objects[b].security_rules[e].name) ? var.nsg_objects[b].security_rules[e].name : "ALLOW-3389_22-INBOUND-FROM-ANY"
      priority = can(var.nsg_objects[b].security_rules[e].priority) ? var.nsg_objects[b].security_rules[e].priority : 100
      direction = can(var.nsg_objects[b].security_rules[e].direction) ? var.nsg_objects[b].security_rules[e].direction : "Inbound"
      access = can(var.nsg_objects[b].security_rules[e].access) ? var.nsg_objects[b].security_rules[e].access : "Allow"
      protocol = can(var.nsg_objects[b].security_rules[e].protocol) ? var.nsg_objects[b].security_rules[e].protocol : "Tcp"
      source_port_range = can(var.nsg_objects[b].security_rules[e].source_port_range) ? var.nsg_objects[b].security_rules[e].source_port_range : "*"
      source_port_ranges = can(var.nsg_objects[b].security_rules[e].source_port_ranges) ? var.nsg_objects[b].security_rules[e].source_port_ranges : null
      destination_port_range = can(var.nsg_objects[b].security_rules[e].destination_port_range) ? var.nsg_objects[b].security_rules[e].destination_port_range : null
      destination_port_ranges = can(var.nsg_objects[b].security_rules[e].destination_port_ranges) ? var.nsg_objects[b].security_rules[e].destination_port_ranges : [22, 3389]
      source_address_prefix = can(var.nsg_objects[b].security_rules[e].source_address_prefix) ? var.nsg_objects[b].security_rules[e].source_address_prefix : "*"
      destination_address_prefix = can(var.nsg_objects[b].security_rules[e].destination_address_prefix) ? var.nsg_objects[b].security_rules[e].destination_address_prefix : "*"
    }] : uuid() => d} 
  }] : a.name => a} : {}
  
  pip_objects = can(length(local.merge_objects_pip)) ? {for each in [for each in local.merge_objects_pip : {
      name = each.name == "bastion" && var.env_name != null ? "${var.env_name}-bastion-pip" : each.name == "bastion" ? "bastion-pip" : each.public_ip != null ? each.public_ip.name : var.env_name != null ? "${var.env_name}-${each.name}-pip" : "${each.name}-pip"
      allocation_method = can(each.public_ip.allocation_method) ? each.public_ip.allocation_method : "Static"
      sku = can(each.public_ip.sku) ? each.public_ip.sku : "Standard"
      tags = can(each.public_ip.tags) ? each.public_ip.tags : null
      vm_name = each.name
    }
  ] : each.name => each} : null

  bastion_object = var.create_bastion && var.bastion_object == null ? {for each in [for x, y in range(1) : {
      name = var.env_name != null ? "${var.env_name}-bastion-host" : "bastion-host"
      copy_paste_enabled = true
      file_copy_enabled = true
      sku = "Standard" //Otherwise file_copy cannot be enabled
      scale_units = 2
      public_ip = local.pip_resource_id
  }] : each.name => each} : var.create_bastion && var.bastion_object != null ? {for each in var.bastion_object : each.name => each} : null

  vm_counter = var.create_public_ip && var.create_bastion ? length(local.merge_objects) + 1 : var.create_bastion == false && var.create_public_ip == false && can(length([for each in local.merge_objects : each if each.public_ip != null])) ? length([for each in local.merge_objects : each if each.public_ip != null]) : var.create_bastion ? length([for each in local.merge_objects : each if each.public_ip != null]) + 1 : 0
  vm_os_names = distinct(flatten([[for each in local.vm_windows_objects : each.os_name if each.source_image_reference == null], [for each in local.vm_linux_objects : each.os_name if each.source_image_reference == null]]))
  vm_sizes = jsondecode(data.local_file.vmskus_objects[0].content).VMSizes

  vm_linux_objects = var.vm_linux_objects == null ? [] : var.vm_linux_objects
  vm_windows_objects = var.vm_windows_objects == null ? [] : var.vm_windows_objects

  merge_objects = flatten([local.vm_linux_objects, local.vm_windows_objects])
  logic_bastion = var.create_bastion ? [{name = "bastion"}] : null
  logic_public_ip_false = var.create_public_ip == false && can([for each in [for each in local.merge_objects : each if each.public_ip != null] : each]) ? [for each in [for each in local.merge_objects : each if each.public_ip != null] : each] : null
  logic_public_ip_true = local.logic_public_ip_false == null && var.create_public_ip ? local.merge_objects : null
  merge_objects_pip = flatten([for each in [local.logic_bastion, local.logic_public_ip_false, local.logic_public_ip_true] : each if each != null])
  vm_names = flatten(local.merge_objects.*.name)
  pip_objects_clean = can([for each in local.pip_objects : each if each.vm_name != "bastion"]) ? [for each in local.pip_objects : each if each.vm_name != "bastion"] : null

  vm_objects_pre = [for x, y in range(length(data.local_file.vmskus_objects)) : {
    publisher = jsondecode(data.local_file.vmskus_objects[x].content).Publisher
    offer = jsondecode(data.local_file.vmskus_objects[x].content).Offer
    versions = jsondecode(data.local_file.vmskus_objects[x].content).Versions
    coresAvailable = jsondecode(data.local_file.vmskus_objects[x].content).CoresAvailable
    coresLimit = jsondecode(data.local_file.vmskus_objects[x].content).CoresLimit
    os = split("-", data.local_file.vmskus_objects[x].filename)[0]
  }]

  vm_objects = {for each in [for x, y in local.merge_objects : {
    name = local.merge_objects[x].name
    admin_username = local.merge_objects[x].admin_username != null ? local.merge_objects[x].admin_username  : "localadmin"
    admin_password = local.merge_objects[x].admin_password != null ? local.merge_objects[x].admin_password : random_password.vm_password_object[x].result
    os_disk_name = can(local.merge_objects[x].os_disk.name) ? local.merge_objects[x].os_disk.name : "${local.merge_objects[x].name}-os-disk"
    os_disk_caching = can(local.merge_objects[x].os_disk.caching) ? local.merge_objects[x].os_disk.caching : "ReadWrite"
    os_disk_storage_account_type = can(local.merge_objects[x].storage_account_type) ? local.merge_objects[x].storage_account_type : "StandardSSD_LRS"
    size = local.merge_objects[x].size_pattern != null ? [for a in ([for b in local.vm_sizes : b if length(regexall((lower(local.merge_objects[x].size_pattern)), lower(b.Name))) > 0]) : a if a.TempDriveSizeInGB > 0][0].Name : local.merge_objects[x].size != null ? local.merge_objects[x].size : [for a in ([for b in local.vm_sizes : b if length(regexall((lower("b2ms")), lower(b.Name))) > 0]) : a if a.TempDriveSizeInGB > 0][0].Name
    os_disk_size = !can(local.merge_objects[x].os_disk.disk_size_gb) ? [for a in ([for b in local.vm_sizes : b if length(regexall((lower("b2ms")), lower(b.Name))) > 0]) : a if a.TempDriveSizeInGB > 0][0].OSDiskSizeInGB : local.merge_objects[x].os_disk.disk_size_gb
    publisher = can([for each in local.vm_objects_pre : each.publisher if lower(each.os) == lower(local.merge_objects[x].os_name)][0]) ? [for each in local.vm_objects_pre : each.publisher if lower(each.os) == lower(local.merge_objects[x].os_name)][0] : local.merge_objects[x].source_image_reference.publisher
    offer = can([for each in local.vm_objects_pre : each.offer if lower(each.os) == lower(local.merge_objects[x].os_name)][0]) ? [for each in local.vm_objects_pre : each.offer if lower(each.os) == lower(local.merge_objects[x].os_name)][0] : local.merge_objects[x].source_image_reference.offer 
    sku = can([for each in local.vm_objects_pre : each.versions[0].SKU if lower(each.os) == lower(local.merge_objects[x].os_name)][0]) ? [for each in local.vm_objects_pre : each.versions[0].SKU if lower(each.os) == lower(local.merge_objects[x].os_name)][0] : local.merge_objects[x].source_image_reference.sku
    version = can([for each in local.vm_objects_pre : each.versions[0].Versions if lower(each.os) == lower(local.merge_objects[x].os_name)][0]) ? [for each in local.vm_objects_pre : each.versions[0].Versions if lower(each.os) == lower(local.merge_objects[x].os_name)][0] : local.merge_objects[x].source_image_reference.version
    nic_resource_id = [for a in local.nic_resource_id : a if length(regexall(([for b in local.nic_objects : b.name if b.vm_name == local.merge_objects[index(local.vm_names, local.merge_objects[x].name)].name][0]), a)) > 0]
  }] : each.name => each}

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
    pip_resource_id = can([for a in local.pip_resource_id : a if length(regexall(([for b in local.pip_objects_clean : b.name if b.vm_name == local.merge_objects[index(local.vm_names, local.merge_objects[x].name)].name][0]), a)) > 0][0]) ? [for a in local.pip_resource_id : a if length(regexall(([for b in local.pip_objects_clean : b.name if b.vm_name == local.merge_objects[index(local.vm_names, local.merge_objects[x].name)].name][0]), a)) > 0][0] : null
    vm_name = local.merge_objects[x].name
  }] : each.name => each}

  storage_counter = length([for each in flatten(local.merge_objects.*.boot_diagnostics) : each if can(length(each))]) != 0 && var.create_diagnostic_settings ? length([for each in flatten(local.merge_objects.*.boot_diagnostics) : each if can(length(each))]) + 1 : var.create_diagnostic_settings ? 1 : length([for each in flatten(local.merge_objects.*.boot_diagnostics) : each if can(length(each))])
  transformed_storage_objects = [for each in [for each in local.merge_objects.*.boot_diagnostics : each if can(length(each))] : each if each != null]

  storage_account_objects = local.storage_counter > 0 ? {for each in [for a in range(local.storage_counter) : {
    name = can(local.transformed_storage_objects[a].storage_account.name) ? local.transformed_storage_objects[a].storage_account.name : var.env_name != null ? "${var.env_name}$vmstorage${substr(uuid(), 0, 5)}" : "vmstorage${substr(uuid(), 0, 5)}"
    vm_name = can(length(flatten(local.transformed_storage_objects[a].storage_account.name))) ? [for a in local.merge_objects : a.name if can(length(a.boot_diagnostics.storage_account))][0] : "placehoholder${a}"
    access_tier = can(local.transformed_storage_objects[a].storage_account.access_tier) ? local.transformed_storage_objects[a].storage_account.access_tier : "Cool"
    public_network_access_enabled = can(local.transformed_storage_objects[a].storage_account.public_network_access_enabled) ? local.transformed_storage_objects[a].storage_account.public_network_access_enabled : true
    account_tier = can(length(local.transformed_storage_objects[a].storage_account.account_tier)) ? local.transformed_storage_objects[a].storage_account.account_tier : "Standard"
    account_replication_type = can(length(local.transformed_storage_objects[a].storage_account.account_replication_type)) ? local.transformed_storage_objects[a].storage_account.account_replication_type : "LRS"
    account_kind = "StorageV2" 
    network_rules = can(length(local.transformed_storage_objects[a].storage_account.network_rules)) ? [for c, d in local.transformed_storage_objects[a].storage_account.network_rules : [
      {
        default_action = can(length(c.default_action)) ? c.default_action : "Deny"
        bypass = can(length(c.bypass)) ? c.bypass : ["Logging", "Metrics", "AzureServices"] 
        virtual_network_subnet_ids = can(length(c.virtual_network_subnet_ids)) ? c.virtual_network_subnet_ids : [for a in local.subnet_resource_id : a if length(regexall("vm", a)) > 0]
        ip_rules = can(length(c.ip_rules)) ? c.ip_rules : null
        private_link_access = can(length(c.private_link_access)) ? c.private_link_access : null
      } 
    ]] : null
  }] : each.vm_name => each} : {}
  
  script_name = var.script_name != null && can(file(var.script_name)) ? var.script_name : var.script_name == null ? "Get-AzVMSku.ps1" : null

  rg_resource_id = can(azurerm_resource_group.rg_object[0].id) ? azurerm_resource_group.rg_object[0].id : var.rg_id
  vnet_resource_id = can(length(azurerm_virtual_network.vnet_object)) ? flatten(values(azurerm_virtual_network.vnet_object))[0].id : var.vnet_resource_id
  subnet_resource_id = can(length(azurerm_subnet.subnet_object)) ? flatten(values(azurerm_subnet.subnet_object).*.id) : [var.subnet_resource_id]
  pip_resource_id =  can(length(azurerm_public_ip.pip_object)) ? flatten(values(azurerm_public_ip.pip_object).*.id) : []
  nic_resource_id =  can(length(azurerm_network_interface.nic_object)) ? flatten(values(azurerm_network_interface.nic_object).*.id) : []
  nsg_resource_id = can(length(azurerm_network_security_group.vm_nsg_object)) ? flatten(values(azurerm_network_security_group.vm_nsg_object).*.id) : []
  storage_resource_id = can(length(azurerm_storage_account.vm_storage_account_object)) ? flatten(values(azurerm_storage_account.vm_storage_account_object).*.id) : []

  //Return objects
  rg_return_object = can(azurerm_resource_group.rg_object[0]) ? azurerm_resource_group.rg_object[0] : null
  vnet_return_object = can(length(azurerm_virtual_network.vnet_object)) ? azurerm_virtual_network.vnet_object : null
  subnet_return_object = can(length(azurerm_subnet.subnet_object)) ? azurerm_subnet.subnet_object : null
  nsg_return_object = can(length(azurerm_network_security_group.vm_nsg_object)) ? azurerm_network_security_group.vm_nsg_object : null
  nic_return_object = can(length(azurerm_network_interface.nic_object)) ?  azurerm_network_interface.nic_object : null
  pip_return_object = can(length(azurerm_public_ip.pip_object)) ? azurerm_public_ip.pip_object : null
  windows_return_object = can(length(azurerm_windows_virtual_machine.vm_windows_object))  ? azurerm_windows_virtual_machine.vm_windows_object : null
  linux_return_object = can(length(azurerm_linux_virtual_machine.vm_linux_object)) ? azurerm_linux_virtual_machine.vm_linux_object : null
  storage_return_object = can(length(azurerm_storage_account.vm_storage_account_object)) ? azurerm_storage_account.vm_storage_account_object : null
}

resource "null_resource" "ps_object" {
  count = can(length(local.vm_os_names)) && local.script_name != null ? length(local.vm_os_names) : 0
  provisioner "local-exec" {
        command = "${path.module}/${local.script_name} -Location ${var.location} -OS ${local.vm_os_names[count.index]} -OutputFileName ${local.vm_os_names[count.index]}-skus.json"
        interpreter = ["pwsh","-Command"]
  }
}

data "local_file" "vmskus_objects" {
  count = length(local.vm_os_names)
  filename = "${local.vm_os_names[count.index]}-skus.json"

  depends_on = [ null_resource.ps_object ]
}

resource "random_password" "vm_password_object" {
  count = length(local.merge_objects) //Regardless of whether the user wants to supply own passwords, create a list of passwords ready
  length           = 16
  special          = true
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

  lifecycle {
    create_before_destroy = true
  }
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
    subnet_id = [for each in local.subnet_resource_id : each if length(regexall("Bastion", each)) == 0][0]
    private_ip_address_allocation = each.value.private_ip_address_allocation
    private_ip_address = each.value.private_ip_address
    public_ip_address_id = each.value.pip_resource_id
  }

  lifecycle {
    ignore_changes = [ ip_configuration ]
  }
}

resource "azurerm_network_security_group" "vm_nsg_object" {
  for_each = local.nsg_objects
  name = each.key
  resource_group_name = local.rg_object.name
  location = var.location
  tags = each.value.tags

  dynamic "security_rule" {
    for_each = each.value.security_rules
    content {
      name = security_rule.value.name
      priority = security_rule.value.priority
      direction = security_rule.value.direction
      access = security_rule.value.access
      protocol = security_rule.value.protocol
      source_port_range = security_rule.value.source_port_range
      source_port_ranges = security_rule.value.source_port_ranges
      destination_port_range = security_rule.value.destination_port_range
      destination_port_ranges = security_rule.value.destination_port_ranges
      source_address_prefix = security_rule.value.source_address_prefix
      destination_address_prefix = security_rule.value.destination_address_prefix
    }
  }

  lifecycle {
    ignore_changes = [security_rule]
  }
}

resource "azurerm_subnet_network_security_group_association" "vm_nsg_link_object" {
  for_each = local.nsg_objects
  subnet_id = each.value.subnet_id
  network_security_group_id = [for a in local.nsg_resource_id : a if length(regexall(each.key, a)) > 0][0]

  lifecycle {
    ignore_changes = [ network_security_group_id ]
  }
}

resource "azurerm_windows_virtual_machine" "vm_windows_object" {
  for_each = var.vm_windows_objects != null ? {for each in var.vm_windows_objects : each.name => each} : {}
  name = each.key
  resource_group_name = local.rg_object.name
  location = var.location
  network_interface_ids = [for a in local.vm_objects : a.nic_resource_id if a.name == each.key][0]
  size = [for a in local.vm_objects : a.size if a.name == each.key][0]
  admin_username = [for a in local.vm_objects : a.admin_username if a.name == each.key][0]
  admin_password = [for a in local.vm_objects : a.admin_password if a.name == each.key][0]
  allow_extension_operations = can(each.value.allow_extension_operations) ? each.value.allow_extension_operations : null
  availability_set_id = can(each.value.availability_set_id) ? each.value.availability_set_id : null
  bypass_platform_safety_checks_on_user_schedule_enabled = can(each.value.bypass_platform_safety_checks_on_user_schedule_enabled) ? each.value.bypass_platform_safety_checks_on_user_schedule_enabled : null
  capacity_reservation_group_id = can(each.value.capacity_reservation_group_id) ? each.value.capacity_reservation_group_id : null
  computer_name = can(each.value.computer_name) ? each.value.computer_name : null
  custom_data = can(each.value.custom_data) ? each.value.custom_data : null
  dedicated_host_id = can(each.value.dedicated_host_id) ? each.value.dedicated_host_id : null
  dedicated_host_group_id = can(each.value.dedicated_host_group_id) ? each.value.dedicated_host_group_id : null
  edge_zone = can(each.value.edge_zone) ? each.value.edge_zone : null
  enable_automatic_updates = can(each.value.enable_automatic_updates) ? each.value.enable_automatic_updates : null
  eviction_policy = can(each.value.eviction_policy) ? each.value.eviction_policy : null
  extensions_time_budget = can(each.value.extensions_time_budget) ? each.value.extensions_time_budget : null
  hotpatching_enabled = can(each.value.hotpatching_enabled) ? each.value.hotpatching_enabled : null
  license_type = can(each.value.license_type) ? each.value.license_type : null
  max_bid_price = can(each.value.max_bid_price) ? each.value.max_bid_price : null
  patch_assessment_mode = can(each.value.patch_assessment_mode) ? each.value.patch_assessment_mode : null
  patch_mode = can(each.value.patch_mode) ? each.value.patch_mode : null
  platform_fault_domain = can(each.value.platform_fault_domain) ? each.value.platform_fault_domain : null
  priority = can(each.value.priority) ? each.value.priority : null
  provision_vm_agent = can(each.value.provision_vm_agent) ? each.value.provision_vm_agent : null
  proximity_placement_group_id = can(each.value.proximity_placement_group_id) ? each.value.proximity_placement_group_id : null
  reboot_setting = can(each.value.reboot_setting) ? each.value.reboot_setting : null
  secure_boot_enabled = can(each.value.secure_boot_enabled) ? each.value.secure_boot_enabled : null
  source_image_id = can(each.value.source_image_id) ? each.value.source_image_id : null
  tags = can(each.value.tags) ? each.value.tags : null
  timezone = can(each.value.timezone) ? each.value.timezone : null
  user_data = can(each.value.user_data) ? each.value.user_data : null
  virtual_machine_scale_set_id = can(each.value.virtual_machine_scale_set_id) ? each.value.virtual_machine_scale_set_id : null
  vtpm_enabled = can(each.value.vtpm_enabled) ? each.value.vtpm_enabled : null

  dynamic "additional_capabilities" {
    for_each = can(each.value.additional_capabilities.ultra_ssd_enabled) ? {for a in [each.value.additional_capabilities] : uuid() => a} : {}
    content {
      ultra_ssd_enabled = each.value.additional_capabilities.ultra_ssd_enabled
    }
  }

  dynamic "additional_unattend_content" {
    for_each = can(each.value.additional_unattend_content[0]) ? {for a in each.value.additional_unattend_content : uuid() => a} : {}
    content {
      content = additional_unattend_content.value.content
      setting = additional_unattend_content.value.setting
    }
  }
  
  dynamic "boot_diagnostics" {
    for_each = can(length(local.storage_resource_id)) ? {for a in [range(1)] : uuid() => a} : {}
    content {
      storage_account_uri = var.create_diagnostic_settings && !can(length(each.value.boot_diagnostics)) ? [for a in local.storage_return_object : a.primary_blob_endpoint if length(regexall("vmstorage", a.name)) > 0][0] : can(length(each.value.boot_diagnostics)) ? [for a in local.storage_return_object : a.primary_blob_endpoint if length(regexall(each.value.boot_diagnostics.storage_account.name, a.name)) > 0][0] : null
    }
  }

  dynamic "gallery_application" {
    for_each = can(each.value.gallery_application[0]) ? {for a in each.value.gallery_application : uuid() => a} : {}
    content {
      version_id = gallery_application.value.version_id
      configuration_blob_uri = can(gallery_application.value.configuration_blob_uri) ? gallery_application.value.configuration_blob_uri : null
      order = can(gallery_application.value.order) ? gallery_application.value.order : null
      tag = can(gallery_application.value.tag) ? gallery_application.value.tag : null
    }
  }
  
  dynamic "identity" {
    for_each = can(each.value.identity.type) ? {for a in [each.value.identity] : uuid() => a} : {}
    content {
      type = identity.value.type
      identity_ids = can(identity.value.identity_ids[0]) ? identity.value.identity_ids : null
    }
  }

  os_disk {
    name = [for a in local.vm_objects : a.os_disk_name if a.name == each.key][0]
    caching = [for a in local.vm_objects : a.os_disk_caching if a.name == each.key][0]
    storage_account_type = [for a in local.vm_objects : a.os_disk_storage_account_type if a.name == each.key][0]
    disk_encryption_set_id = can(each.value.os_disk.disk_encryption_set_id) ? each.value.os_disk.disk_encryption_set_id : null
    disk_size_gb = [for a in local.vm_objects : a.os_disk_size if a.name == each.key][0]
    secure_vm_disk_encryption_set_id = can(each.value.os_disk.secure_vm_disk_encryption_set_id) ? each.value.os_disk.secure_vm_disk_encryption_set_id : null
    security_encryption_type = can(each.value.os_disk.security_encryption_type) ? each.value.os_disk.security_encryption_type : null
    write_accelerator_enabled = can(each.value.os_disk.write_accelerator_enabled) ? each.value.os_disk.write_accelerator_enabled : null

    dynamic "diff_disk_settings" {
      for_each = can(each.value.os_disk.diff_disk_settings.option) ? {for a in each.value.os_disk.diff_disk_settings : uuid() => a} : {}
      content {
        option = each.value.os_disk.diff_disk_settings.option
        placement = can(each.value.os_disk.diff_disk_settings.placement) ? each.value.os_disk.diff_disk_settings.placement : null
      }
    }
  }

  dynamic "plan" {
    for_each = can(each.value.plan.name) ? {for a in [each.value.plan] : uuid() => a} : {}
    content {
      name = plan.name
      product = plan.product
      publisher = plan.publisher
    }
  }

  dynamic "secret" {
    for_each = can(each.value.secret[0]) ? {for a in each.value.secret : uuid() => a} : {}
    content {
      key_vault_id = secret.value.key_vault_id

      dynamic "certificate" {
        for_each = {for a in flatten(each.value.secret.*.certificate) : uuid() => a}
        content {
          store = certificate.value.store
          url = certificate.value.url
        }
      }
    }
  }

  dynamic "source_image_reference" {
    for_each = each.value.source_image_id == null ? {for a in [for b in local.vm_objects : b if b.name == each.key] : a.name => a} : {}
    content {
      publisher = source_image_reference.value.publisher
      offer = source_image_reference.value.offer
      sku = source_image_reference.value.sku
      version = source_image_reference.value.version
    }
  }

  dynamic "termination_notification" {
    for_each = can(each.value.termination_notification.enabled) ? {for a in [each.value.termination_notification] : uuid() => a} : {}
    content {
      enabled = termination_notification.value.enabled
      timeout = termination_notification.value.timeout
    }
  }

  dynamic "winrm_listener" {
    for_each = can(each.value.winrm_listener[0]) ? {for a in each.value.winrm_listener : uuid() => a} : {}
    content {
      protocol = winrm_listener.value.protocol
      certificate_url = winrm_listener.value.certificate_url
    }
  }

  lifecycle {
    ignore_changes = [ source_image_reference, boot_diagnostics ]
  }
}

resource "azurerm_linux_virtual_machine" "vm_linux_object" {
  for_each = var.vm_linux_objects != null ? {for each in var.vm_linux_objects : each.name => each} : {}
  name = each.key
  resource_group_name = local.rg_object.name
  location = var.location
  license_type = can(each.value.license_type) ? each.value.license_type : null
  network_interface_ids = [for a in local.vm_objects : a.nic_resource_id if a.name == each.key][0]
  size = [for a in local.vm_objects : a.size if a.name == each.key][0]
  admin_username = [for a in local.vm_objects : a.admin_username if a.name == each.key][0]
  admin_password = [for a in local.vm_objects : a.admin_password if a.name == each.key][0]
  allow_extension_operations = can(each.value.allow_extension_operations) ? each.value.allow_extension_operations : null
  availability_set_id = can(each.value.availability_set_id) ? each.value.availability_set_id : null
  bypass_platform_safety_checks_on_user_schedule_enabled = can(each.value.bypass_platform_safety_checks_on_user_schedule_enabled) ? each.value.bypass_platform_safety_checks_on_user_schedule_enabled : null
  capacity_reservation_group_id = can(each.value.capacity_reservation_group_id) ? each.value.capacity_reservation_group_id : null
  computer_name = can(each.value.computer_name) ? each.value.computer_name : null
  custom_data = can(each.value.custom_data) ? each.value.custom_data : null
  dedicated_host_id = can(each.value.dedicated_host_id) ? each.value.dedicated_host_id : null
  dedicated_host_group_id = can(each.value.dedicated_host_group_id) ? each.value.dedicated_host_group_id : null
  disable_password_authentication = !can(each.value.disable_password_authentication) ? null : each.value.disable_password_authentication == null ? false : null
  edge_zone = can(each.value.edge_zone) ? each.value.edge_zone : null
  encryption_at_host_enabled = can(each.value.encryption_at_host_enabled) ? each.value.encryption_at_host_enabled : null
  eviction_policy = can(each.value.eviction_policy) ? each.value.eviction_policy : null
  extensions_time_budget = can(each.value.extensions_time_budget) ? each.value.extensions_time_budget : null
  patch_assessment_mode = can(each.value.patch_assessment_mode) ? each.value.patch_assessment_mode : null
  patch_mode = can(each.value.patch_mode) ? each.value.patch_mode : null
  max_bid_price = can(each.value.max_bid_price) ? each.value.max_bid_price : null
  platform_fault_domain = can(each.value.platform_fault_domain) ? each.value.platform_fault_domain : null
  priority = can(each.value.priority) ? each.value.priority : null
  provision_vm_agent = can(each.value.provision_vm_agent) ? each.value.provision_vm_agent : null
  proximity_placement_group_id = can(each.value.proximity_placement_group_id) ? each.value.proximity_placement_group_id : null
  reboot_setting = can(each.value.reboot_setting) ? each.value.reboot_setting : null
  secure_boot_enabled = can(each.value.secure_boot_enabled) ? each.value.secure_boot_enabled : null
  source_image_id = can(each.value.source_image_id) ? each.value.source_image_id : null
  tags = can(each.value.tags) ? each.value.tags : null
  user_data = can(each.value.user_data) ? each.value.user_data : null
  vtpm_enabled = can(each.value.vtpm_enabled) ? each.value.vtpm_enabled : null
  virtual_machine_scale_set_id = can(each.value.virtual_machine_scale_set_id) ? each.value.virtual_machine_scale_set_id : null
  zone = can(each.value.zone) ? each.value.zone : null

  dynamic "additional_capabilities" {
    for_each = can(each.value.additional_capabilities.ultra_ssd_enabled) ? {for a in [each.value.additional_capabilities] : uuid() => a} : {}
    content {
      ultra_ssd_enabled = each.value.additional_capabilities.ultra_ssd_enabled
    }
  }

  dynamic "admin_ssh_key" {
    for_each = can(each.value.admin_ssh_key[0]) ? {for a in each.value.admin_ssh_key : uuid() => a} : {}
    content {
      public_key = admin_ssh_key.value.public_key
      username = admin_ssh_key.value.username
    }
  }

  dynamic "boot_diagnostics" {
    for_each = can(length(local.storage_resource_id)) ? {for a in [range(1)] : uuid() => a} : {}
    content {
      storage_account_uri = var.create_diagnostic_settings && !can(length(each.value.boot_diagnostics)) ? [for a in local.storage_resource_id : a if length(regexall("vmstorage", a)) > 0][0] : can(length(each.value.boot_diagnostics)) ? [for a in local.storage_resource_id : a if length(regexall(each.value.boot_diagnostics.storage_account.name, a) > 0)] : null
    }
  }

  dynamic "gallery_application" {
    for_each = can(each.value.gallery_application[0]) ? {for a in each.value.gallery_application : uuid() => a} : {}
    content {
      version_id = gallery_application.value.version_id
      configuration_blob_uri = can(gallery_application.value.configuration_blob_uri) ? gallery_application.value.configuration_blob_uri : null
      order = can(gallery_application.value.order) ? gallery_application.value.order : null
      tag = can(gallery_application.value.tag) ? gallery_application.value.tag : null
    }
  }

  dynamic "identity" {
  for_each = can(each.value.identity.type) ? {for a in [each.value.identity] : uuid() => a} : {}
  content {
    type = identity.value.type
    identity_ids = can(identity.value.identity_ids[0]) ? identity.value.identity_ids : null
    }
  }

  os_disk {
    name = [for a in local.vm_objects : a.os_disk_name if a.name == each.key][0]
    caching = [for a in local.vm_objects : a.os_disk_caching if a.name == each.key][0]
    storage_account_type = [for a in local.vm_objects : a.os_disk_storage_account_type if a.name == each.key][0]
    disk_encryption_set_id = can(each.value.os_disk.disk_encryption_set_id) ? each.value.os_disk.disk_encryption_set_id : null
    disk_size_gb = [for a in local.vm_objects : a.os_disk_size if a.name == each.key][0]
    secure_vm_disk_encryption_set_id = can(each.value.os_disk.secure_vm_disk_encryption_set_id) ? each.value.os_disk.secure_vm_disk_encryption_set_id : null
    security_encryption_type = can(each.value.os_disk.security_encryption_type) ? each.value.os_disk.security_encryption_type : null
    write_accelerator_enabled = can(each.value.os_disk.write_accelerator_enabled) ? each.value.os_disk.write_accelerator_enabled : null

    dynamic "diff_disk_settings" {
      for_each = can(each.value.os_disk.diff_disk_settings.option) ? {for a in each.value.os_disk.diff_disk_settings : uuid() => a} : {}
      content {
        option = each.value.os_disk.diff_disk_settings.option
        placement = can(each.value.os_disk.diff_disk_settings.placement) ? each.value.os_disk.diff_disk_settings.placement : null
      }
    }
  }

  dynamic "plan" {
    for_each = can(each.value.plan.name) ? {for a in [each.value.plan] : uuid() => a} : {}
    content {
      name = plan.name
      product = plan.product
      publisher = plan.publisher
    }
  }

   dynamic "secret" {
    for_each = can(each.value.secret[0]) ? {for a in each.value.secret : uuid() => a} : {}
    content {
      key_vault_id = secret.value.key_vault_id

      dynamic "certificate" {
        for_each = {for a in flatten(secret.value.*.certificate) : uuid() => a}
        content {
          url = certificate.value.url
        }
      }
    }
  }

  dynamic "source_image_reference" {
    for_each = each.value.source_image_id == null ? {for a in [for b in local.vm_objects : b if b.name == each.key] : a.name => a} : {}
    content {
      publisher = source_image_reference.value.publisher
      offer = source_image_reference.value.offer
      sku = source_image_reference.value.sku
      version = source_image_reference.value.version
    }
  }

  dynamic "termination_notification" {
    for_each = can(each.value.termination_notification.enabled) ? {for a in [each.value.termination_notification] : uuid() => a} : {}
    content {
      enabled = termination_notification.value.enabled
      timeout = termination_notification.value.timeout
    }
  }

  lifecycle {
    ignore_changes = [admin_ssh_key, admin_password, boot_diagnostics]
  }
}

resource "azurerm_storage_account" "vm_storage_account_object" {
  for_each = local.storage_account_objects
  name = each.value.name
  resource_group_name = local.rg_object.name
  location = var.location
  access_tier = each.value.access_tier
  public_network_access_enabled = each.value.public_network_access_enabled
  account_kind = each.value.account_kind
  account_replication_type = each.value.account_replication_type
  account_tier = each.value.account_tier

  dynamic "network_rules" {
    for_each = can(length(each.value.network_rules)) ? {for a in [each.value.network_rules] : uuid() => a} : {}
    content {
      default_action = network_rules.value.default_action
      bypass = network_rules.value.bypass
      virtual_network_subnet_ids = network_rules.value.virtual_network_subnet_ids
      ip_rules = can(length(network_rules.value.ip_rules)) ? network_rules.value.ip_rules : null
      
      dynamic "private_link_access" {
        for_each = can(length(network_rules.value.private_link_access)) ? {for a in network_rules.value.private_link_access : uuid() => a} : {}
        content {
          endpoint_resource_id = private_link_access.value.endpoint_resource_id
          endpoint_tenant_id = can(private_link_access.value.endpoint_tenant_id) ? private_link_access.value.endpoint_tenant_id : null
        }
      }
    }
  }
  
  lifecycle {
    ignore_changes = [ name ]
  }
}