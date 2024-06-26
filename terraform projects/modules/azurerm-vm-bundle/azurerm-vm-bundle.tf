terraform {
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = ">=3.76.0"
    }
    null = {
      source  = "hashicorp/null"
      version = ">=3.2.1"
    }
    random = {
      source  = "hashicorp/random"
      version = ">=3.5.1"
    }
    local = {
      source  = "hashicorp/local"
      version = ">=2.4.0"
    }
  }
}


################################################################################################
######################################### NOTES ################################################
##                                                                                            ##
##  Date: 17-03-2024                                                                          ## 
##  State: Version 1.3                                                                        ##
##  Missing: Refactoring / Clean up, create more effective variables.                         ##
##  Improvements (1): Solved a huge sub-set of bugs. New logic to further stabilize the code  ##                                                                                
##  =||= (2): Dynamically able to pull information about existing vnet & subnets              ## 
##  =||= (3): N/A                                                                             ##
##  Future improvements: Working on allowing the use of a custom script path                  ##
##  Comments: ALL FEEDBACK IS APPRICIATED - PLEASE READ THE README FILE...                    ##
##                                                                                            ##
## -------------------------------------------------------------------------------------------##
##                                                                                            ##
##  Date: 09-01-2024                                                                          ## 
##  State: Bumped to version 1.2 - skipping 1.1 due to bad release...                         ##
##  Missing: Continue to improve the functionallity of the module...                          ##
##  Improvements (1): Made it possible to create mgmt resources then use them in other calls  ##                                                                                
##  =||= (2): Added a new sub-parameter that enables you to create a nsg with 0 custom rules  ## 
##  =||= (3): Added kv output and improved logic in the subnet object definition              ##
##  Future improvements: Still missing comments...                                            ##
##  Comments: ALL FEEDBACK IS APPRICIATED - PLEASE READ THE README FILE...                    ##
##                                                                                            ##
##--------------------------------------------------------------------------------------------##
##                                                                                            ##                                                                                     
##  Date: 20-11-2023                                                                          ## 
##  State: Ready for production, version 1.0                                                  ##
##  Missing: Being able to parse a kv_resource_id, to use existing key vault for vm secrets   ##
##  Improvements (1): Many small bugs has been fixed during final testing...                  ##                                                                     
##  =||= (2): Multiple different "nice-to-have" features added via object parameters...       ## 
##  =||= (3): N/A                                                                             ##
##  Future improvements: Still missing comments...                                            ##
##  Comments: Module is pretty far at this stage, around 99% done...                          ##
##                                                                                            ##   
##--------------------------------------------------------------------------------------------##
##                                                                                            ##
##  Date: 15-11-2023                                                                          ## 
##  State: Had huge issues with storage accounts today, should now be stable...               ##
##  Missing: Create KV                                                                        ##
##  Improvements (1): Solved many different bugs, module is stable...                         ##                                                                     
##  =||= (2): Need to write the readme, maybe use chatgpt...                                  ## 
##  =||= (3): Extensive tests has been done with many different scenarios...                  ##
##  Future improvements: Clean up the locals block & add comment sections in code...          ##
##  Comments: Module is pretty far at this stage, around 92% done...                          ##
##                                                                                            ##
##--------------------------------------------------------------------------------------------##   
##                                                                                            ##
##  Date: 13-11-2023                                                                          ## 
##  State: Storage account & customized output has been defined...                            ##
##  Missing: Create KV                                                                        ##
##  Improvements (1): Solved many different bugs, module is stable...                         ##                                                                     
##  =||= (2): Need to write the readme, maybe use chatgpt...                                  ## 
##  =||= (3): Extensive tests has been done with many different scenarios...                  ##
##  Future improvements: Clean up the locals block & add comment sections in code...          ##
##  Comments: Module is pretty far at this stage, around 90% done...                          ##
##--------------------------------------------------------------------------------------------##                                                                                           
##                                                                                            ##
##  Date: 09-11-2023                                                                          ## 
##  State: All resources was though to defined and extensive testing has begun, but turns     ##
##  out I need to define all possible resources directly in the module to avoid error:        ##
##  'The "for_each" map includes keys derived from resource attributes...                     ##
##  Missing: Create storage account, kv                                                       ##
##  Improvements (1): Since last update the solution for json payload scenarios are added     ##                                                                     
##  =||= (2): Wont be solved, if TF needs to read a file, let it                              ## 
##  =||= (3): Need to decide whether I should allow custom json payloads                      ##
##  Future improvements: Clean up the locals block & add comment sections in code             ##
##  Comments: Module is pretty far at this stage, around 80% done                             ##      
##--------------------------------------------------------------------------------------------##                                                                                        
##                                                                                            ##
##  Date: 08-11-2023                                                                          ## 
##  State: Linux VMs has been defined & fixed a bug in the PS script attached to the code     ##
##  Missing: Create a resource definition for nsgs to set on subnets & extensive tests        ##
##  Improvements (1): Since last update the solution for json payload scenarios are added     ##                                                                     
##  =||= (2): Wont be solved, if TF needs to read a file, let it                              ## 
##  =||= (3): Need to decide whether I should allow custom json payloads                      ##
##  Future improvements: Clean up the locals block & add comment sections in code             ##
##  Comments: Module is pretty far at this stage, around 80% done                             ##
##                                                                                            ##
##--------------------------------------------------------------------------------------------##                                                                                     
##                                                                                            ##
##  Date: 30-10-2023                                                                          ##    
##  State: All bugs are fixed for all working compontents including creating win vm's         ##
##  Missing: Define the block for linux vm´s and test many different deployment scenarios     ##
##  Improvements (1): Find solution for when JSON payload contain invalid image               ##
##  =||= (2): Make it dynamically possible to not run the file data block                     ## 
##  =||= (3): To (2), make it possible to parse a filepath for custom JSON payloads           ##
##  Future improvements: Clean up the locals block & add comment sections in code             ##
##  Comments: First entry                                                                     ##
##                                                                                            ##                                                                                                                                                       
################################################################################################
################################################################################################

locals {
  //Variable transformation
  rg_object = var.rg_name == null ? { name = split("/", var.rg_id)[4], create_rg = false } : { name = var.rg_name, create_rg = true }

  vnet_object_pre = var.vnet_resource_id == null && var.vnet_object == null ? { for each in [for each in range(1) : {
    name          = var.env_name != null ? "${var.env_name}-vm-vnet" : "vm-vnet"
    address_space = var.env_name == null ? ["192.168.0.0/24", "192.168.99.0/24"] : length(regexall("\\b[pP][rR]?[oO]?[dD]?[uU]?[cC]?[tT]?[iI]?[oO]?[nN]?\\b", var.env_name)) > 0 ? ["10.0.0.0/16", "10.99.0.0/24"] : length(regexall("^\\b[tT][eE]?[sS]?[tT]?[iI]?[nN]?[gG]?\\b$", var.env_name)) > 0 ? ["172.16.0.0/20", "172.16.99.0/24"] : ["192.168.0.0/24", "192.168.99.0/24"]
  }] : each.name => each } : null

  vnet_object_pre2   = local.vnet_object_pre == null && var.vnet_object != null ? { for each in [var.vnet_object] : each.name => each } : null
  vnet_object_helper = can(values(flatten([for each in [local.vnet_object_pre, local.vnet_object_pre2] : each if each != null])[0])[0]) ? values(flatten([for each in [local.vnet_object_pre, local.vnet_object_pre2] : each if each != null])[0])[0] : null

  subnet_creation_count = var.subnet_objects != null && (var.create_bastion || var.bastion_object != null) && var.subnet_resource_id == null || var.subnet_bastion_resource_id == null && var.create_bastion && var.subnet_resource_id == null ? 2 : var.subnet_objects != null || var.subnet_resource_id != null && var.create_bastion || var.subnet_resource_id == null ? 1 : 0
  subnet_data_helper = compact([var.subnet_resource_id, var.subnet_bastion_resource_id])
  vnet_address_space = can(local.vnet_data_object.address_space[0]) ? local.vnet_data_object.address_space[0] : null
  
  subnet_objects = {for each in [for x, y in range(local.subnet_creation_count) : {
    name              = x == 0 && (var.create_bastion || var.bastion_object != null) ? "AzureBastionSubnet" :  var.subnet_resource_id != null ? split("/",var.subnet_resource_id)[10] : !can(var.subnet_objects[x].name) ? "vm-subnet" : length(regexall("bastion",lower(var.subnet_objects[x].name))) == 0 ? var.subnet_objects[x].name : can(var.subnet_objects[x].name) ? var.subnet_objects[x].name : null
    address_prefixes  = can(var.subnet_objects[x].address_prefixes) ? var.subnet_objects[x].address_prefixes : can(cidrsubnet(local.vnet_address_space, 1, 1)) ? [cidrsubnet(local.vnet_address_space, 26 - tonumber(split("/", local.vnet_address_space)[1]), pow(2, 26 - tonumber(split("/", local.vnet_address_space)[1])) -1)] : x == 0 && var.create_bastion ? [cidrsubnet(local.vnet_object_helper.address_space[1], 2, 0)] : [cidrsubnet(local.vnet_object_helper.address_space[0], 1, 0)]
    service_endpoints = x == 0 && (!var.create_bastion || var.bastion_object == null) || x == 1 && (var.create_bastion || var.bastion_object != null) && length([for each in local.merge_objects : each if can(each.boot_diagnostics.storage_account.network_rules)]) > 0 ? ["Microsoft.KeyVault", "Microsoft.Storage.Global"] :  x == 0 && (!var.create_bastion || var.bastion_object == null) || x == 1 && (var.create_bastion || var.bastion_object != null) ? ["Microsoft.KeyVault"] : null
  }] : each.name => each}

  nsg_objects_pre       = !can(length(var.nsg_objects)) && var.create_nsg ? 1 : can(length(var.nsg_objects)) ? length(var.nsg_objects) : 0
  nsg_objects_rules_pre = can(var.nsg_objects.*.security_rules) ? length(flatten(var.nsg_objects.*.security_rules)) : 1

  nsg_objects = {for a in [for b, c in range(local.nsg_objects_pre) : {
    name = can(var.nsg_objects[b].name) ? var.nsg_objects[b].name : var.env_name != null ? "${var.env_name}-vm-nsg" : "vm-nsg"
    subnet_id = can(var.nsg_objects[b].subnet_id) ? var.nsg_objects[b].subnet_id : var.subnet_resource_id != null ? var.subnet_resource_id : var.subnet_objects != null ? [for a in local.subnet_resource_id : a if length(regexall("Bastion", a)) == 0][0] : [for each in local.subnet_resource_id : each if length(regexall("vm", each)[0]) > 0][0]
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
      destination_address_prefix = can(var.nsg_objects[b].security_rules[e].destination_address_prefix) ? var.nsg_objects[b].security_rules[e].destination_address_prefix : var.subnet_resource_id == null ? [for each in local.subnet_return_object : each.address_prefixes[0] if each.name != "AzureBastion"][0] : [for each in data.azurerm_subnet.data_subnet_object : each.address_prefixes[0] if each.name != "AzureBastion"][0]
    }] : uuid() => d} 
  }] : a.name => a}

  pip_objects = can(length(local.merge_objects_pip)) ? { for each in [for each in local.merge_objects_pip : {
    name              = each.name == "bastion" && var.env_name != null ? "${var.env_name}-bastion-pip" : each.name == "bastion" ? "bastion-pip" : each.public_ip != null ? each.public_ip.name : var.env_name != null ? "${var.env_name}-${each.name}-pip" : "${each.name}-pip"
    allocation_method = can(each.public_ip.allocation_method) ? each.public_ip.allocation_method : "Static"
    sku               = can(each.public_ip.sku) ? each.public_ip.sku : "Standard"
    tags              = can(each.public_ip.tags) ? each.public_ip.tags : null
    vm_name           = each.name
    }
  ] : each.name => each } : null

  bastion_object_pre = var.create_bastion && var.bastion_object == null ? { for each in [for x, y in range(1) : {
    name               = var.env_name != null ? "${var.env_name}-bastion-host" : "bastion-host"
    copy_paste_enabled = true
    file_copy_enabled  = true
    sku                = "Standard" //Otherwise file_copy cannot be enabled
    scale_units        = 2
  }] : each.name => each } : null

  bastion_object_pre2 = local.bastion_object_pre == null && var.bastion_object != null ? { for each in [var.bastion_object] : each.name => each } : null
  bastion_object      = local.bastion_object_pre2 != null || local.bastion_object_pre != null ? [for a in flatten([local.bastion_object_pre, local.bastion_object_pre2]) : a if a != null][0] : null

  vm_os_names = distinct(flatten([[for each in local.vm_windows_objects : each.os_name if each.source_image_reference == null], [for each in local.vm_linux_objects : each.os_name if each.source_image_reference == null]]))
  vm_sizes    = can(jsondecode(data.local_file.vmskus_objects[0].content).VMSizes) ? jsondecode(data.local_file.vmskus_objects[0].content).VMSizes : null 

  vm_linux_objects   = var.vm_linux_objects == null ? [] : var.vm_linux_objects
  vm_windows_objects = var.vm_windows_objects == null ? [] : var.vm_windows_objects

  merge_objects         = flatten([local.vm_linux_objects, local.vm_windows_objects])
  logic_bastion         = var.create_bastion || var.bastion_object != null ? [{ name = "bastion" }] : null
  logic_public_ip_false = var.create_public_ip == false && can([for each in [for each in local.merge_objects : each if each.public_ip != null] : each]) ? [for each in [for each in local.merge_objects : each if each.public_ip != null] : each] : null
  logic_public_ip_true  = local.logic_public_ip_false == null && var.create_public_ip ? local.merge_objects : null
  merge_objects_pip     = flatten([for each in [local.logic_bastion, local.logic_public_ip_false, local.logic_public_ip_true] : each if each != null])
  vm_names              = flatten(local.merge_objects.*.name)
  pip_objects_clean     = can([for each in local.pip_objects : each if each.vm_name != "bastion"]) ? [for each in local.pip_objects : each if each.vm_name != "bastion"] : null

  vm_objects_pre = [for x, y in range(length(data.local_file.vmskus_objects)) : {
    publisher      = jsondecode(data.local_file.vmskus_objects[x].content).Publisher
    offer          = jsondecode(data.local_file.vmskus_objects[x].content).Offer
    versions       = jsondecode(data.local_file.vmskus_objects[x].content).Versions
    coresAvailable = jsondecode(data.local_file.vmskus_objects[x].content).CoresAvailable
    coresLimit     = jsondecode(data.local_file.vmskus_objects[x].content).CoresLimit
    os             = split("-", data.local_file.vmskus_objects[x].filename)[0]
    vm_sizes       = jsondecode(data.local_file.vmskus_objects[x].content).VMSizes
  }]

  vm_objects = { for each in [for x, y in local.merge_objects : {
    name                         = local.merge_objects[x].name
    admin_username               = local.merge_objects[x].admin_username != null ? local.merge_objects[x].admin_username : "localadmin"
    admin_password               = local.merge_objects[x].admin_password != null ? local.merge_objects[x].admin_password : random_password.vm_password_object[x].result
    os_disk_name                 = can(local.merge_objects[x].os_disk.name) ? local.merge_objects[x].os_disk.name : "${local.merge_objects[x].name}-os-disk"
    os_disk_caching              = can(local.merge_objects[x].os_disk.caching) ? local.merge_objects[x].os_disk.caching : "ReadWrite"
    os_disk_storage_account_type = can(local.merge_objects[x].storage_account_type) ? local.merge_objects[x].storage_account_type : "StandardSSD_LRS"
    size                         = local.merge_objects[x].size_pattern != null ? [for a in([for b in local.vm_sizes : b if length(regexall((lower(local.merge_objects[x].size_pattern)), lower(b.Name))) > 0]) : a if a.TempDriveSizeInGB > 0][0].Name : local.merge_objects[x].size != null ? local.merge_objects[x].size : local.vm_sizes != null ? [for a in([for b in local.vm_sizes : b if length(regexall((lower("b2ms")), lower(b.Name))) > 0]) : a if a.TempDriveSizeInGB > 0][0].Name : "Standard_B2ms"
    os_disk_size                 = !can(local.merge_objects[x].os_disk.disk_size_gb) && local.vm_sizes != null ? [for a in([for b in local.vm_sizes : b if length(regexall((lower("b2ms")), lower(b.Name))) > 0]) : a if a.TempDriveSizeInGB > 0][0].OSDiskSizeInGB : can(local.merge_objects[x].os_disk.disk_size_gb) ? local.merge_objects[x].os_disk.disk_size_gb : 128
    publisher                    = can([for each in local.vm_objects_pre : each.publisher if lower(each.os) == lower(local.merge_objects[x].os_name)][0]) ? [for each in local.vm_objects_pre : each.publisher if lower(each.os) == lower(local.merge_objects[x].os_name)][0] : local.merge_objects[x].source_image_reference.publisher
    offer                        = can([for each in local.vm_objects_pre : each.offer if lower(each.os) == lower(local.merge_objects[x].os_name)][0]) ? [for each in local.vm_objects_pre : each.offer if lower(each.os) == lower(local.merge_objects[x].os_name)][0] : local.merge_objects[x].source_image_reference.offer
    sku                          = can([for each in local.vm_objects_pre : each.versions[0].SKU if lower(each.os) == lower(local.merge_objects[x].os_name)][0]) ? [for each in local.vm_objects_pre : each.versions[0].SKU if lower(each.os) == lower(local.merge_objects[x].os_name)][0] : local.merge_objects[x].source_image_reference.sku
    version                      = can(length([for each in local.vm_objects_pre : each.versions[0].Versions if lower(each.os) == lower(local.merge_objects[x].os_name)][0])) ? [for each in local.vm_objects_pre : each.versions[0].Versions if lower(each.os) == lower(local.merge_objects[x].os_name)][0] : can(length(local.merge_objects[x].source_image_reference.version)) ? local.merge_objects[x].source_image_reference.version : "latest"
    nic_resource_id              = [for a in local.nic_resource_id : a if length(regexall(([for b in local.nic_objects : b.name if b.vm_name == local.merge_objects[index(local.vm_names, local.merge_objects[x].name)].name][0]), a)) > 0]
  }] : each.name => each }

  nic_objects = { for each in [for x, y in local.merge_objects : {
    name                          = can(local.merge_objects[x].nic.name) ? local.merge_objects[x].nic.name : "${local.merge_objects[x].name}-nic"
    subnet_id                     = can([for each in local.subnet_return_object : each.id if each.name != "AzureBastionSubnet"][0]) ? [for each in local.subnet_return_object : each.id if each.name != "AzureBastionSubnet"][0] : [for each in local.subnet_resource_id : each if !contains(split("/", each), "AzureBastionSubnet")][0]
    dns_servers                   = can(local.merge_objects[x].nic.dns_servers) ? local.merge_objects[x].nic.dns_servers : null
    enable_ip_forwarding          = can(local.merge_objects[x].nic.enable_ip_forwarding) ? local.merge_objects[x].nic.enable_ip_forwarding : null
    edge_zone                     = can(local.merge_objects[x].nic.edge_zone) ? local.merge_objects[x].nic.edge_zone : null
    ip_configuration_name         = can(local.merge_objects[x].nic.ip_configuration.name) ? local.merge_objects[x].nic.ip_configuration.name : "ip-config"
    private_ip_address_version    = can(local.merge_objects[x].nic.ip_configuration.private_ip_address_version) ? local.merge_objects[x].nic.ip_configuration.private_ip_address_version : null
    private_ip_address            = can(local.merge_objects[x].nic.ip_configuration.private_ip_address) ? local.merge_objects[x].nic.ip_configuration.private_ip_address : null
    private_ip_address_allocation = can(local.merge_objects[x].nic.ip_configuration.private_ip_address_allocation) ? local.merge_objects[x].nic.ip_configuration.private_ip_address_allocation : "Dynamic"
    tags                          = can(local.merge_objects[x].nic.tags) ? local.merge_objects[x].nic.tags : null
    pip_resource_id               = can([for a in local.pip_resource_id : a if length(regexall(([for b in local.pip_objects_clean : b.name if b.vm_name == local.merge_objects[index(local.vm_names, local.merge_objects[x].name)].name][0]), a)) > 0][0]) ? [for a in local.pip_resource_id : a if length(regexall(([for b in local.pip_objects_clean : b.name if b.vm_name == local.merge_objects[index(local.vm_names, local.merge_objects[x].name)].name][0]), a)) > 0][0] : null
    vm_name                       = local.merge_objects[x].name 
  }] : each.name => each }

  storage_counter             = length([for each in flatten(local.merge_objects.*.boot_diagnostics) : each if can(length(each))]) != 0 && var.create_diagnostic_settings ? length([for each in flatten(local.merge_objects.*.boot_diagnostics) : each if can(length(each))]) + 1 : var.create_diagnostic_settings ? 1 : length([for each in flatten(local.merge_objects.*.boot_diagnostics) : each if can(length(each))])
  transformed_storage_objects = can([for each in flatten([for each in local.merge_objects.*.boot_diagnostics : each if each != null]) : each]) ? [for each in flatten([for each in local.merge_objects.*.boot_diagnostics : each if each != null]) : each] : []

  storage_account_objects = local.storage_counter > 0 ? { for each in [for a in range(local.storage_counter) : { 
    name                          = can(local.transformed_storage_objects[a].storage_account.name) ? local.transformed_storage_objects[a].storage_account.name : var.env_name != null ? "${var.env_name}vmstorage${substr(uuid(), 0, 5)}" : "vmstorage${substr(uuid(), 0, 5)}"
    vm_name                       = can(length(flatten(local.transformed_storage_objects[a].storage_account.name))) ? [for a in local.merge_objects : a.name if can(length(a.boot_diagnostics.storage_account))][0] : "storage${a}"
    access_tier                   = can(local.transformed_storage_objects[a].storage_account.access_tier) ? local.transformed_storage_objects[a].storage_account.access_tier : "Cool"
    public_network_access_enabled = can(local.transformed_storage_objects[a].storage_account.public_network_access_enabled) ? local.transformed_storage_objects[a].storage_account.public_network_access_enabled : true
    account_tier                  = can(length(local.transformed_storage_objects[a].storage_account.account_tier)) ? local.transformed_storage_objects[a].storage_account.account_tier : "Standard"
    account_replication_type      = can(length(local.transformed_storage_objects[a].storage_account.account_replication_type)) ? local.transformed_storage_objects[a].storage_account.account_replication_type : "LRS"
    account_kind                  = "StorageV2"
    network_rules = can(length(local.transformed_storage_objects[a].storage_account.network_rules)) ? { for a, b in [for c in range(1) : {
      default_action             = can(length(local.transformed_storage_objects[c].storage_account.network_rules.default_action)) ? local.transformed_storage_objects[c].storage_account.network_rules.default_action : "Deny"
      bypass                     = can(length(local.transformed_storage_objects[c].storage_account.network_rules.bypass)) ? local.transformed_storage_objects[c].storage_account.network_rules.bypass : ["Logging", "Metrics", "AzureServices"]
      virtual_network_subnet_ids = can(length(local.transformed_storage_objects[c].storage_account.network_rules.virtual_network_subnet_ids)) ? local.transformed_storage_objects[c].storage_account.network_rules.virtual_network_subnet_ids : [for a in local.subnet_resource_id : a if length(regexall("vm", a)) > 0]
      ip_rules                   = can(length(local.transformed_storage_objects[c].storage_account.network_rules.ip_rules)) ? local.transformed_storage_objects[c].storage_account.network_rules.ip_rules : null
      private_link_access        = can(length(local.transformed_storage_objects[c].storage_account.network_rules.private_link_access)) ? local.transformed_storage_objects[c].storage_account.network_rules.private_link_access : null
    }] : a => b } : {}
  }] : each.vm_name => each } : {}

  kv_object = var.create_kv_for_vms || can(length(var.kv_object)) ? { for each in [for a in range(1) : {
    name                            = can(length(var.kv_object.name)) ? var.kv_object.name : var.env_name != null ? "${var.env_name}vmkv${substr(uuid(), 0, 5)}" : "vmkv${substr(uuid(), 0, 5)}"
    sku_name                        = can(length(var.kv_object.sku_name)) ? var.kv_object.sku_name : "standard"
    enabled_for_deployment          = can(length(var.kv_object.enabled_for_deployment)) ? var.kv_object.enabled_for_deployment : true
    enabled_for_disk_encryption     = can(length(var.kv_object.enabled_for_disk_encryption)) ? var.kv_object.enabled_for_disk_encryption : false
    enabled_for_template_deployment = can(length(var.kv_object.enabled_for_template_deployment)) ? var.kv_object.enabled_for_template_deployment : false
    enable_rbac_authorization       = true //Module will only allow RBAC authorization
    purge_protection_enabled        = can(length(var.kv_object.purge_protection_enabled)) ? var.kv_object.purge_protection_enabled : false
    public_network_access_enabled   = can(length(var.kv_object.public_network_access_enabled)) ? var.kv_object.public_network_access_enabled : true
    tags                            = can(length(var.kv_object.tags)) ? var.kv_object.tags : null
    soft_delete_retention_days      = can(length(var.kv_object.soft_delete_retention_days)) ? var.kv_object.soft_delete_retention_days : 7

    network_acls = can(var.kv_object.network_acls.add_vm_subnet_id) ? { for each in [for a in range(1) : {
      bypass                     = "AzureServices"
      default_action             = "Deny"
      virtual_network_subnet_ids = [for b in local.subnet_resource_id : b if length(regexall("vm", a)) > 0]
    }] : "network_acls" => each } : null
  }] : "kv_object" => each } : null

  script_path = flatten((regexall("^(.*\\/)?([^\\/]+)\\.ps1$",var.script_name)))[0]

  script_commands = length(local.vm_os_names) > 0 ? flatten([for a, b in range(length(local.vm_os_names)) : [
    length([for c in local.merge_objects : c if c.allow_null_version != null && c.os_name == local.vm_os_names[a]]) > 0 ? "${var.script_name} -Location ${var.location} -OS ${local.vm_os_names[a]} -OutputFileName ${local.vm_os_names[a]}-skus.json -AllowNoVersions" : "${var.script_name} -Location ${var.location} -OS ${local.vm_os_names[a]} -OutputFileName ${local.vm_os_names[a]}-skus.json"
  ]]) : null                                                                                                                                                                                                        

  rg_resource_id      = can(azurerm_resource_group.rg_object[0].id) ? azurerm_resource_group.rg_object[0].id : var.rg_id
  vnet_resource_id    = length(azurerm_virtual_network.vnet_object) > 0 ? flatten(values(azurerm_virtual_network.vnet_object))[0].id : var.vnet_resource_id
  subnet_resource_id  = length(azurerm_subnet.subnet_object) > 1 || length(azurerm_subnet.subnet_object) == 1 && var.subnet_resource_id == null || var.subnet_bastion_resource_id == null && var.subnet_resource_id == null ? flatten(values(azurerm_subnet.subnet_object).*.id) : local.subnet_data_helper
  pip_resource_id     = length(azurerm_public_ip.pip_object) > 0 ? flatten(values(azurerm_public_ip.pip_object).*.id) : []
  nic_resource_id     = length(azurerm_network_interface.nic_object) > 0 ? flatten(values(azurerm_network_interface.nic_object).*.id) : []
  nsg_resource_id     = length(azurerm_network_security_group.vm_nsg_object) > 0 ? flatten(values(azurerm_network_security_group.vm_nsg_object).*.id) : []
  storage_resource_id = length(azurerm_storage_account.vm_storage_account_object) > 0 ? flatten(values(azurerm_storage_account.vm_storage_account_object).*.id) : []
  kv_resource_id      = length(azurerm_key_vault.vm_kv_object) > 0 ? flatten(values(azurerm_key_vault.vm_kv_object).*.id)[0] : var.kv_resource_id

  //Return objects
  rg_return_object      = can(azurerm_resource_group.rg_object[0]) ? azurerm_resource_group.rg_object[0] : null
  vnet_return_object    = length(azurerm_virtual_network.vnet_object) > 0 ? azurerm_virtual_network.vnet_object : null
  subnet_return_object  = length(azurerm_subnet.subnet_object) > 0 ? azurerm_subnet.subnet_object : null
  nsg_return_object     = length(azurerm_network_security_group.vm_nsg_object) > 0 ? azurerm_network_security_group.vm_nsg_object : null
  nic_return_object     = length(azurerm_network_interface.nic_object) > 0 ? azurerm_network_interface.nic_object : null
  pip_return_object     = length(azurerm_public_ip.pip_object) > 0 ? azurerm_public_ip.pip_object : null
  windows_return_object = length(azurerm_windows_virtual_machine.vm_windows_object) > 0 ? azurerm_windows_virtual_machine.vm_windows_object : null
  linux_return_object   = length(azurerm_linux_virtual_machine.vm_linux_object) > 0 ? azurerm_linux_virtual_machine.vm_linux_object : null
  storage_return_object = length(azurerm_storage_account.vm_storage_account_object) > 0 ? azurerm_storage_account.vm_storage_account_object : null
  kv_return_object      = length(azurerm_key_vault.vm_kv_object) > 0 ? azurerm_key_vault.vm_kv_object : null

  //Data objects
  vnet_data_object = length(data.azurerm_virtual_network.data_vnet_object) > 0 ? data.azurerm_virtual_network.data_vnet_object[0] : null
  subnet_data_object = length(data.azurerm_subnet.data_subnet_object) > 0 ? data.azurerm_subnet.data_subnet_object : null

  summary_of_deployment = {
    prefix_for_names_used   = var.env_name != null ? true : false
    vnet_deployed           = local.vnet_return_object != null ? true : false
    subnet_deployed         = local.subnet_return_object != null ? true : false
    public_ip_deployed      = local.pip_return_object != null ? true : false
    nsg_deployed            = local.nsg_return_object != null ? true : false
    storage_deployed        = local.storage_return_object != null ? true : false
    bastion_deployed        = length(azurerm_bastion_host.bastion_object) > 0 ? true : false
    windows_vm_deployed     = local.windows_return_object != null ? true : false
    linux_vm_deployed       = local.linux_return_object != null ? true : false
    cpu_cores_total_sub     = length(local.vm_objects_pre) > 0 ? local.vm_objects_pre[0].coresLimit : null
    cpu_cores_available_sub = length(local.vm_objects_pre) > 0 ? local.vm_objects_pre[0].coresAvailable : null

    network_summary = {
      address_space = can(length(local.vnet_object_helper.address_space)) ? local.vnet_object_helper.address_space : null
      vnet_name     = can(length(local.vnet_object_helper.name)) ? local.vnet_object_helper.name : null

      subnets = can(length(local.subnet_objects)) ? [for each in local.subnet_objects : {
        name           = each.name
        address_prefix = each.address_prefixes
      }] : null
    }

    windows_objects = local.windows_return_object != null ? [for each in local.windows_return_object : {
      name           = each.name
      admin_username = [for a in local.vm_objects : a.admin_username if a.name == each.name][0]
      os             = [for a in var.vm_windows_objects : a.os_name if a.name == each.name][0]
      os_sku         = [for a in local.vm_objects : a.sku if a.name == each.name][0]

      size = {
        name      = [for a in local.vm_objects : a.size if a.name == each.name][0]
        memory_gb = length(local.vm_objects_pre) > 0 ? [for a in local.vm_objects_pre[0].vm_sizes : a.MemoryInGB if length(regexall(a.Name, [for a in local.vm_objects : a.size if a.name == each.name][0])) > 0][0] : null
        cpu_cores = length(local.vm_objects_pre) > 0 ? [for a in local.vm_objects_pre[0].vm_sizes : a.CoresAvailable if length(regexall(a.Name, [for a in local.vm_objects : a.size if a.name == each.name][0])) > 0][0] : null
      }

      network_summary = {
        private_ip_address = can(length(local.windows_return_object)) ? [for a in local.windows_return_object : a.private_ip_address if a.name == each.name][0] : null
        public_ip_address  = can(length(local.windows_return_object)) ? [for a in local.windows_return_object : a.public_ip_address if a.name == each.name][0] : null
      }
    }] : null

    linux_objects = local.linux_return_object != null ? [for x, y in var.vm_linux_objects : {
      name           = y.name
      admin_username = [for a in local.vm_objects : a.admin_username if a.name == y.name][0]
      os             = [for a in var.vm_linux_objects : a.os_name if a.name == y.name][0]
      os_sku         = [for a in local.vm_objects : a.sku if a.name == y.name][0]

      # ssh = can(length(local.linux_return_object)) && length([for b in var.vm_linux_objects : b if b.admin_ssh_key != null]) > 0 ? {for a in [for b, c in var.vm_linux_objects : {
      #   connect_string = "${[for d in var.vm_linux_objects : d.os_name if d.name == y.name][0]}@${[for d in local.linux_return_object : d.public_ip_address if d.name == y.name][0]}"
      #  public_key = c.*.public_key
      # }] : y.name => a} : null

      size = {
        name      = [for a in local.vm_objects : a.size if a.name == y.name][0]
        memory_gb = length(local.vm_objects_pre) > 0 ? [for a in local.vm_objects_pre[0].vm_sizes : a.MemoryInGB if length(regexall(a.Name, [for a in local.vm_objects : a.size if a.name == y.name][0])) > 0][0] : null
        cpu_cores = length(local.vm_objects_pre) > 0 ? [for a in local.vm_objects_pre[0].vm_sizes : a.CoresAvailable if length(regexall(a.Name, [for a in local.vm_objects : a.size if a.name == y.name][0])) > 0][0] : null
      }

      network_summary = {
        private_ip_address = can(length(local.linux_return_object)) ? [for a in local.linux_return_object : a.private_ip_address if a.name == y.name][0] : null
        public_ip_address  = can(length(local.linux_return_object)) ? [for a in local.linux_return_object : a.public_ip_address if a.name == y.name][0] : null
      }
    }] : null
  }
}

data "azurerm_virtual_network" "data_vnet_object" {
  count = var.vnet_resource_id != null ? 1 : 0
  name = split("/", var.vnet_resource_id)[8]
  resource_group_name = local.rg_object.name
}

data "azurerm_subnet" "data_subnet_object" {
  count = var.subnet_resource_id != null && var.subnet_bastion_resource_id != null ? 2 : var.subnet_resource_id != null || var.subnet_bastion_resource_id != null ? 1 : 0
  name = split("/", local.subnet_data_helper[count.index])[10]
  virtual_network_name = split("/", local.subnet_data_helper[count.index])[8]
  resource_group_name = local.rg_object.name
}

resource "null_resource" "download_script" {
  count = local.script_commands != null ? 1 : 0
  provisioner "local-exec" {
    command     = <<-EOT
      for($i = 0; $i -le 100; $i++){
        $url = "https://raw.githubusercontent.com/ChristofferWin/codeterraform/main/terraform%20projects/modules/azurerm-vm-bundle/Get-AzVMSKu.ps1"
        $outputPath = "${var.script_name}"
        $content = Invoke-WebRequest -Uri $url
        if($content.count -gt 0){
        while($true){
           try{
            $content.Content | Out-File -path $outputPath -ErrorAction Stop
            exit
           }
           catch{
             Start-Sleep -Seconds 3
           }
         }  
        }
      }
    EOT
    interpreter = ["pwsh", "-Command"]
  }
}

resource "null_resource" "ps_object" {
  count = local.script_commands != null ? length(local.script_commands) : 0
  provisioner "local-exec" {
    command     = local.script_commands[count.index]
    interpreter = ["pwsh", "-Command"]
  }

  depends_on = [null_resource.download_script]
}

data "local_file" "vmskus_objects" {
  count    = length(local.vm_os_names)
  filename = "${local.vm_os_names[count.index]}-skus.json"

  depends_on = [null_resource.ps_object]
}

resource "random_password" "vm_password_object" {
  count   = length(local.merge_objects) //Regardless of whether the user wants to supply own passwords, create a list of passwords ready
  length  = 16
  special = true

  depends_on = [azurerm_key_vault.vm_kv_object]
}

resource "azurerm_resource_group" "rg_object" {
  count    = local.rg_object.create_rg ? 1 : 0
  name     = local.rg_object.name
  location = var.location
  tags     = var.rg_tags
}

resource "azurerm_virtual_network" "vnet_object" {
  for_each            = var.vnet_resource_id == null ? { for each in [local.vnet_object_helper] : each.name => each } : {}
  name                = each.key
  resource_group_name = local.rg_object.name
  location            = var.location
  address_space       = each.value.address_space
  tags                = can(each.value.tags) ? each.value.tags : null

  depends_on = [azurerm_resource_group.rg_object]
}

resource "azurerm_subnet" "subnet_object" {
  for_each             = local.subnet_objects
  name                 = each.value.name
  resource_group_name  = local.rg_object.name
  virtual_network_name = can(local.vnet_object_helper.name) ? local.vnet_object_helper.name : split("/", var.vnet_resource_id)[8]
  address_prefixes     = each.value.address_prefixes
  service_endpoints = each.value.service_endpoints

  depends_on = [azurerm_virtual_network.vnet_object]
}

resource "azurerm_public_ip" "pip_object" {
  for_each            = local.pip_objects
  name                = each.key
  resource_group_name = local.rg_object.name
  location            = var.location
  allocation_method   = each.value.allocation_method
  sku                 = each.value.sku
  tags                = can(each.value.tags) ? each.value.tags : null

  depends_on = [azurerm_subnet.subnet_object]

  lifecycle {
    create_before_destroy = true
  }
}

resource "azurerm_bastion_host" "bastion_object" {
  for_each            = var.create_bastion || var.bastion_object != null ? local.bastion_object : {}
  name                = each.key
  resource_group_name = local.rg_object.name
  location            = var.location
  copy_paste_enabled  = each.value.copy_paste_enabled
  file_copy_enabled   = each.value.file_copy_enabled
  sku                 = each.value.sku
  scale_units         = each.value.scale_units
  tags                = can(each.value.tags) ? each.value.tags : null

  ip_configuration {
    name                 = "ip-config"
    subnet_id            = local.subnet_return_object == null ? [for each in local.subnet_resource_id : each if length(regexall("bastion", lower(each))) > 0][0] : [for each in local.subnet_return_object : each.id if length(regexall("bastion", lower(each.id))) > 0][0]
    public_ip_address_id = [for each in local.pip_resource_id : each if length(regexall("bastion", lower(each))) > 0][0]
  }

  lifecycle {
    ignore_changes = [ip_configuration]
  }

  depends_on = [azurerm_public_ip.pip_object]
}

resource "azurerm_network_interface" "nic_object" {
  for_each             = local.nic_objects
  name                 = each.key
  resource_group_name  = local.rg_object.name
  location             = var.location
  dns_servers          = each.value.dns_servers
  enable_ip_forwarding = each.value.enable_ip_forwarding
  edge_zone            = each.value.edge_zone
  tags                 = each.value.tags

  ip_configuration {
    name                          = each.value.ip_configuration_name
    private_ip_address_allocation = each.value.private_ip_address_allocation
    private_ip_address            = each.value.private_ip_address
    public_ip_address_id          = each.value.pip_resource_id
    subnet_id                     = each.value.subnet_id
  }

  lifecycle {
    ignore_changes = [ip_configuration]
  }
}

resource "azurerm_network_security_group" "vm_nsg_object" {
  for_each            = local.nsg_objects
  name                = each.key
  resource_group_name = local.rg_object.name
  location            = var.location
  tags                = each.value.tags

  dynamic "security_rule" {
    for_each = each.value.security_rules
    content {
      name                       = security_rule.value.name
      priority                   = security_rule.value.priority
      direction                  = security_rule.value.direction
      access                     = security_rule.value.access
      protocol                   = security_rule.value.protocol
      source_port_range          = security_rule.value.source_port_range
      source_port_ranges         = security_rule.value.source_port_ranges
      destination_port_range     = security_rule.value.destination_port_range
      destination_port_ranges    = security_rule.value.destination_port_ranges
      source_address_prefix      = security_rule.value.source_address_prefix
      destination_address_prefix = security_rule.value.destination_address_prefix
    }
  }

  lifecycle {
    ignore_changes = [security_rule]
  }

  depends_on = [azurerm_resource_group.rg_object]
}

resource "azurerm_subnet_network_security_group_association" "vm_nsg_link_object" {
  for_each                  = local.nsg_objects
  subnet_id                 = [for a in local.subnet_resource_id : a if length(regexall("bastion", lower(a))) == 0][0]
  network_security_group_id = [for a in local.nsg_resource_id : a if length(regexall(each.key, a)) > 0][0]

  lifecycle {
    ignore_changes = [network_security_group_id, subnet_id]
  }
}

resource "azurerm_windows_virtual_machine" "vm_windows_object" {
  for_each                                               = var.vm_windows_objects != null ? { for each in var.vm_windows_objects : each.name => each } : {}
  name                                                   = each.key
  resource_group_name                                    = local.rg_object.name
  location                                               = var.location
  network_interface_ids                                  = [for a in local.vm_objects : a.nic_resource_id if a.name == each.key][0]
  size                                                   = [for a in local.vm_objects : a.size if a.name == each.key][0]
  admin_username                                         = [for a in local.vm_objects : a.admin_username if a.name == each.key][0]
  admin_password                                         = [for a in local.vm_objects : a.admin_password if a.name == each.key][0]
  allow_extension_operations                             = can(each.value.allow_extension_operations) ? each.value.allow_extension_operations : null
  availability_set_id                                    = can(each.value.availability_set_id) ? each.value.availability_set_id : null
  bypass_platform_safety_checks_on_user_schedule_enabled = can(each.value.bypass_platform_safety_checks_on_user_schedule_enabled) ? each.value.bypass_platform_safety_checks_on_user_schedule_enabled : null
  capacity_reservation_group_id                          = can(each.value.capacity_reservation_group_id) ? each.value.capacity_reservation_group_id : null
  computer_name                                          = can(each.value.computer_name) ? each.value.computer_name : null
  custom_data                                            = can(each.value.custom_data) ? each.value.custom_data : null
  dedicated_host_id                                      = can(each.value.dedicated_host_id) ? each.value.dedicated_host_id : null
  dedicated_host_group_id                                = can(each.value.dedicated_host_group_id) ? each.value.dedicated_host_group_id : null
  edge_zone                                              = can(each.value.edge_zone) ? each.value.edge_zone : null
  enable_automatic_updates                               = can(each.value.enable_automatic_updates) ? each.value.enable_automatic_updates : null
  eviction_policy                                        = can(each.value.eviction_policy) ? each.value.eviction_policy : null
  extensions_time_budget                                 = can(each.value.extensions_time_budget) ? each.value.extensions_time_budget : null
  hotpatching_enabled                                    = can(each.value.hotpatching_enabled) ? each.value.hotpatching_enabled : null
  license_type                                           = can(each.value.license_type) ? each.value.license_type : null
  max_bid_price                                          = can(each.value.max_bid_price) ? each.value.max_bid_price : null
  patch_assessment_mode                                  = can(each.value.patch_assessment_mode) ? each.value.patch_assessment_mode : null
  patch_mode                                             = can(each.value.patch_mode) ? each.value.patch_mode : null
  platform_fault_domain                                  = can(each.value.platform_fault_domain) ? each.value.platform_fault_domain : null
  priority                                               = can(each.value.priority) ? each.value.priority : null
  provision_vm_agent                                     = can(each.value.provision_vm_agent) ? each.value.provision_vm_agent : null
  proximity_placement_group_id                           = can(each.value.proximity_placement_group_id) ? each.value.proximity_placement_group_id : null
  reboot_setting                                         = can(each.value.reboot_setting) ? each.value.reboot_setting : null
  secure_boot_enabled                                    = can(each.value.secure_boot_enabled) ? each.value.secure_boot_enabled : null
  source_image_id                                        = can(each.value.source_image_id) ? each.value.source_image_id : null
  tags                                                   = can(each.value.tags) ? each.value.tags : null
  timezone                                               = can(each.value.timezone) ? each.value.timezone : null
  user_data                                              = can(each.value.user_data) ? each.value.user_data : null
  virtual_machine_scale_set_id                           = can(each.value.virtual_machine_scale_set_id) ? each.value.virtual_machine_scale_set_id : null 
  vtpm_enabled                                           = can(each.value.vtpm_enabled) ? each.value.vtpm_enabled : null

  dynamic "additional_capabilities" {
    for_each = can(each.value.additional_capabilities.ultra_ssd_enabled) ? { for a in [each.value.additional_capabilities] : uuid() => a } : {}
    content {
      ultra_ssd_enabled = each.value.additional_capabilities.ultra_ssd_enabled
    }
  }

  dynamic "additional_unattend_content" {
    for_each = can(each.value.additional_unattend_content[0]) ? { for a in each.value.additional_unattend_content : uuid() => a } : {}
    content {
      content = additional_unattend_content.value.content
      setting = additional_unattend_content.value.setting
    }
  }

  dynamic "boot_diagnostics" {
    for_each = can(length(local.storage_return_object)) ? { for a in [range(1)] : uuid() => a } : {}
    content {
      storage_account_uri = can(length(each.value.boot_diagnostics)) ? [for a in local.storage_return_object : a.primary_blob_endpoint if length(regexall(each.value.boot_diagnostics.storage_account.name, a.id)) > 0][0] : var.create_diagnostic_settings ? [for a in local.storage_return_object : a.primary_blob_endpoint if length(regexall("vmstorage", a.id)) > 0][0] : null
    }
  }

  dynamic "gallery_application" {
    for_each = can(each.value.gallery_application[0]) ? { for a in each.value.gallery_application : uuid() => a } : {}
    content {
      version_id             = gallery_application.value.version_id
      configuration_blob_uri = can(gallery_application.value.configuration_blob_uri) ? gallery_application.value.configuration_blob_uri : null
      order                  = can(gallery_application.value.order) ? gallery_application.value.order : null
      tag                    = can(gallery_application.value.tag) ? gallery_application.value.tag : null
    }
  }

  dynamic "identity" {
    for_each = can(each.value.identity.type) ? { for a in [each.value.identity] : uuid() => a } : {}
    content {
      type         = identity.value.type
      identity_ids = can(identity.value.identity_ids[0]) ? identity.value.identity_ids : null
    }
  }

  os_disk {
    name                             = [for a in local.vm_objects : a.os_disk_name if a.name == each.key][0]
    caching                          = [for a in local.vm_objects : a.os_disk_caching if a.name == each.key][0]
    storage_account_type             = [for a in local.vm_objects : a.os_disk_storage_account_type if a.name == each.key][0]
    disk_encryption_set_id           = can(each.value.os_disk.disk_encryption_set_id) ? each.value.os_disk.disk_encryption_set_id : null
    disk_size_gb                     = [for a in local.vm_objects : a.os_disk_size if a.name == each.key][0]
    secure_vm_disk_encryption_set_id = can(each.value.os_disk.secure_vm_disk_encryption_set_id) ? each.value.os_disk.secure_vm_disk_encryption_set_id : null
    security_encryption_type         = can(each.value.os_disk.security_encryption_type) ? each.value.os_disk.security_encryption_type : null
    write_accelerator_enabled        = can(each.value.os_disk.write_accelerator_enabled) ? each.value.os_disk.write_accelerator_enabled : null

    dynamic "diff_disk_settings" {
      for_each = can(each.value.os_disk.diff_disk_settings.option) ? { for a in each.value.os_disk.diff_disk_settings : uuid() => a } : {}
      content {
        option    = each.value.os_disk.diff_disk_settings.option
        placement = can(each.value.os_disk.diff_disk_settings.placement) ? each.value.os_disk.diff_disk_settings.placement : null
      }
    }
  }

  dynamic "plan" {
    for_each = can(each.value.plan.name) ? { for a in [each.value.plan] : uuid() => a } : {}
    content {
      name      = plan.name
      product   = plan.product
      publisher = plan.publisher
    }
  }

  dynamic "secret" {
    for_each = can(each.value.secret[0]) ? { for a in each.value.secret : uuid() => a } : {}
    content {
      key_vault_id = secret.value.key_vault_id

      dynamic "certificate" {
        for_each = { for a in flatten(each.value.secret.*.certificate) : uuid() => a }
        content {
          store = certificate.value.store
          url   = certificate.value.url
        }
      }
    }
  }

  dynamic "source_image_reference" {
    for_each = each.value.source_image_id == null ? { for a in [for b in local.vm_objects : b if b.name == each.key] : a.name => a } : {}
    content {
      publisher = source_image_reference.value.publisher
      offer     = source_image_reference.value.offer
      sku       = source_image_reference.value.sku
      version   = source_image_reference.value.version
    }
  }

  dynamic "termination_notification" {
    for_each = can(each.value.termination_notification.enabled) ? { for a in [each.value.termination_notification] : uuid() => a } : {}
    content {
      enabled = termination_notification.value.enabled
      timeout = termination_notification.value.timeout
    }
  }

  dynamic "winrm_listener" {
    for_each = can(each.value.winrm_listener[0]) ? { for a in each.value.winrm_listener : uuid() => a } : {}
    content {
      protocol        = winrm_listener.value.protocol
      certificate_url = winrm_listener.value.certificate_url
    }
  }

  lifecycle {
    ignore_changes = [source_image_reference, boot_diagnostics, admin_password, network_interface_ids, identity]
  }

  depends_on = [ azurerm_storage_account.vm_storage_account_object ]
}

resource "azurerm_linux_virtual_machine" "vm_linux_object" {
  for_each                                               = var.vm_linux_objects != null ? { for each in var.vm_linux_objects : each.name => each } : {}
  name                                                   = each.key
  resource_group_name                                    = local.rg_object.name
  location                                               = var.location
  license_type                                           = can(each.value.license_type) ? each.value.license_type : null
  network_interface_ids                                  = [for a in local.vm_objects : a.nic_resource_id if a.name == each.key][0]
  size                                                   = [for a in local.vm_objects : a.size if a.name == each.key][0]
  admin_username                                         = [for a in local.vm_objects : a.admin_username if a.name == each.key][0]
  admin_password                                         = [for a in local.vm_objects : a.admin_password if a.name == each.key][0]
  allow_extension_operations                             = can(each.value.allow_extension_operations) ? each.value.allow_extension_operations : null
  availability_set_id                                    = can(each.value.availability_set_id) ? each.value.availability_set_id : null
  bypass_platform_safety_checks_on_user_schedule_enabled = can(each.value.bypass_platform_safety_checks_on_user_schedule_enabled) ? each.value.bypass_platform_safety_checks_on_user_schedule_enabled : null
  capacity_reservation_group_id                          = can(each.value.capacity_reservation_group_id) ? each.value.capacity_reservation_group_id : null
  computer_name                                          = can(each.value.computer_name) ? each.value.computer_name : null
  custom_data                                            = can(each.value.custom_data) ? each.value.custom_data : null
  dedicated_host_id                                      = can(each.value.dedicated_host_id) ? each.value.dedicated_host_id : null
  dedicated_host_group_id                                = can(each.value.dedicated_host_group_id) ? each.value.dedicated_host_group_id : null
  disable_password_authentication                        = !can(each.value.disable_password_authentication) ? null : each.value.disable_password_authentication == null ? false : null
  edge_zone                                              = can(each.value.edge_zone) ? each.value.edge_zone : null
  encryption_at_host_enabled                             = can(each.value.encryption_at_host_enabled) ? each.value.encryption_at_host_enabled : null
  eviction_policy                                        = can(each.value.eviction_policy) ? each.value.eviction_policy : null
  extensions_time_budget                                 = can(each.value.extensions_time_budget) ? each.value.extensions_time_budget : null
  patch_assessment_mode                                  = can(each.value.patch_assessment_mode) ? each.value.patch_assessment_mode : null
  patch_mode                                             = can(each.value.patch_mode) ? each.value.patch_mode : null
  max_bid_price                                          = can(each.value.max_bid_price) ? each.value.max_bid_price : null
  platform_fault_domain                                  = can(each.value.platform_fault_domain) ? each.value.platform_fault_domain : null
  priority                                               = can(each.value.priority) ? each.value.priority : null
  provision_vm_agent                                     = can(each.value.provision_vm_agent) ? each.value.provision_vm_agent : null
  proximity_placement_group_id                           = can(each.value.proximity_placement_group_id) ? each.value.proximity_placement_group_id : null
  reboot_setting                                         = can(each.value.reboot_setting) ? each.value.reboot_setting : null
  secure_boot_enabled                                    = can(each.value.secure_boot_enabled) ? each.value.secure_boot_enabled : null
  source_image_id                                        = can(each.value.source_image_id) ? each.value.source_image_id : null
  tags                                                   = can(each.value.tags) ? each.value.tags : null
  user_data                                              = can(each.value.user_data) ? each.value.user_data : null
  vtpm_enabled                                           = can(each.value.vtpm_enabled) ? each.value.vtpm_enabled : null
  virtual_machine_scale_set_id                           = can(each.value.virtual_machine_scale_set_id) ? each.value.virtual_machine_scale_set_id : null
  zone                                                   = can(each.value.zone) ? each.value.zone : null

  dynamic "additional_capabilities" {
    for_each = can(each.value.additional_capabilities.ultra_ssd_enabled) ? { for a in [each.value.additional_capabilities] : uuid() => a } : {}
    content {
      ultra_ssd_enabled = each.value.additional_capabilities.ultra_ssd_enabled
    }
  }

  dynamic "admin_ssh_key" {
    for_each = can(each.value.admin_ssh_key[0]) ? { for a in each.value.admin_ssh_key : uuid() => a } : {}
    content {
      public_key = admin_ssh_key.value.public_key
      username   = admin_ssh_key.value.username
    }
  }

  dynamic "boot_diagnostics" {
    for_each = can(length(local.storage_return_object)) ? { for a in [range(1)] : uuid() => a } : {}
    content {
      storage_account_uri = can(length(each.value.boot_diagnostics)) ? [for a in local.storage_return_object : a.primary_blob_endpoint if length(regexall(each.value.boot_diagnostics.storage_account.name, a.id)) > 0][0] : var.create_diagnostic_settings ? [for a in local.storage_return_object : a.primary_blob_endpoint if length(regexall("vmstorage", a.id)) > 0][0] : null
    }
  }

  dynamic "gallery_application" {
    for_each = can(each.value.gallery_application[0]) ? { for a in each.value.gallery_application : uuid() => a } : {}
    content {
      version_id             = gallery_application.value.version_id
      configuration_blob_uri = can(gallery_application.value.configuration_blob_uri) ? gallery_application.value.configuration_blob_uri : null
      order                  = can(gallery_application.value.order) ? gallery_application.value.order : null
      tag                    = can(gallery_application.value.tag) ? gallery_application.value.tag : null
    }
  }

  dynamic "identity" {
    for_each = can(length(each.value.identity.type)) ? { for a in [each.value.identity] : uuid() => a } : {}
    content {
      type         = identity.value.type
      identity_ids = can(identity.value.identity_ids[0]) ? identity.value.identity_ids : null
    }
  }

  os_disk {
    name                             = [for a in local.vm_objects : a.os_disk_name if a.name == each.key][0]
    caching                          = [for a in local.vm_objects : a.os_disk_caching if a.name == each.key][0]
    storage_account_type             = [for a in local.vm_objects : a.os_disk_storage_account_type if a.name == each.key][0]
    disk_encryption_set_id           = can(each.value.os_disk.disk_encryption_set_id) ? each.value.os_disk.disk_encryption_set_id : null
    disk_size_gb                     = [for a in local.vm_objects : a.os_disk_size if a.name == each.key][0]
    secure_vm_disk_encryption_set_id = can(each.value.os_disk.secure_vm_disk_encryption_set_id) ? each.value.os_disk.secure_vm_disk_encryption_set_id : null
    security_encryption_type         = can(each.value.os_disk.security_encryption_type) ? each.value.os_disk.security_encryption_type : null
    write_accelerator_enabled        = can(each.value.os_disk.write_accelerator_enabled) ? each.value.os_disk.write_accelerator_enabled : null

    dynamic "diff_disk_settings" {
      for_each = can(each.value.os_disk.diff_disk_settings.option) ? { for a in each.value.os_disk.diff_disk_settings : uuid() => a } : {}
      content {
        option    = each.value.os_disk.diff_disk_settings.option
        placement = can(each.value.os_disk.diff_disk_settings.placement) ? each.value.os_disk.diff_disk_settings.placement : null
      }
    }
  }

  dynamic "plan" {
    for_each = can(each.value.plan.name) ? { for a in [each.value.plan] : uuid() => a } : {}
    content {
      name      = plan.name
      product   = plan.product
      publisher = plan.publisher
    }
  }

  dynamic "secret" {
    for_each = can(each.value.secret[0]) ? { for a in each.value.secret : uuid() => a } : {}
    content {
      key_vault_id = secret.value.key_vault_id

      dynamic "certificate" {
        for_each = { for a in flatten(secret.value.*.certificate) : uuid() => a }
        content {
          url = certificate.value.url
        }
      }
    }
  }

  dynamic "source_image_reference" {
    for_each = each.value.source_image_id == null ? { for a in [for b in local.vm_objects : b if b.name == each.key] : a.name => a } : {}
    content {
      publisher = source_image_reference.value.publisher
      offer     = source_image_reference.value.offer
      sku       = source_image_reference.value.sku
      version   = source_image_reference.value.version
    }
  }

  dynamic "termination_notification" {
    for_each = can(each.value.termination_notification.enabled) ? { for a in [each.value.termination_notification] : uuid() => a } : {}
    content {
      enabled = termination_notification.value.enabled
      timeout = termination_notification.value.timeout
    }
  }

  lifecycle {
    ignore_changes = [admin_password, boot_diagnostics, admin_ssh_key, identity]
  }

  depends_on = [ azurerm_storage_account.vm_storage_account_object ]
}

resource "azurerm_storage_account" "vm_storage_account_object" {
  for_each                      = local.storage_account_objects
  name                          = each.value.name
  resource_group_name           = local.rg_object.name
  location                      = var.location
  access_tier                   = each.value.access_tier
  public_network_access_enabled = each.value.public_network_access_enabled
  account_kind                  = each.value.account_kind
  account_replication_type      = each.value.account_replication_type
  account_tier                  = each.value.account_tier

  dynamic "network_rules" {
    for_each = length(each.value.network_rules) > 0 ? { for a in values(each.value.network_rules) : uuid() => a } : {}
    content {
      default_action             = network_rules.value.default_action
      bypass                     = network_rules.value.bypass
      virtual_network_subnet_ids = network_rules.value.virtual_network_subnet_ids
      ip_rules                   = network_rules.value.ip_rules

      dynamic "private_link_access" {
        for_each = can(length(network_rules.value.private_link_access)) ? { for a in network_rules.value.private_link_access : uuid() => a } : {}
        content {
          endpoint_resource_id = private_link_access.value.endpoint_resource_id
          endpoint_tenant_id   = can(private_link_access.value.endpoint_tenant_id) ? private_link_access.value.endpoint_tenant_id : null
        }
      }
    }
  }

  lifecycle {
    ignore_changes = [name, access_tier, network_rules] 
  }
}

data "azurerm_client_config" "current" {}

resource "azurerm_key_vault" "vm_kv_object" { 
  for_each                        = local.kv_object != null ? local.kv_object : {} 
  name                            = each.value.name
  resource_group_name             = local.rg_object.name
  location                        = var.location
  sku_name                        = each.value.sku_name
  tenant_id                       = data.azurerm_client_config.current.tenant_id
  enabled_for_disk_encryption     = each.value.enabled_for_disk_encryption
  enable_rbac_authorization       = each.value.enable_rbac_authorization
  enabled_for_deployment          = each.value.enabled_for_deployment
  enabled_for_template_deployment = each.value.enabled_for_template_deployment
  purge_protection_enabled        = each.value.purge_protection_enabled
  public_network_access_enabled   = each.value.public_network_access_enabled
  soft_delete_retention_days      = each.value.soft_delete_retention_days
  tags                            = each.value.tags

  dynamic "network_acls" {
    for_each = can(length(var.kv_object.network_acls.bypass)) ? { for each in [var.kv_object.network_acls] : uuid()  => each } : can(local.kv_object.network_acls) ? { for each in [local.kv_object.network_acls] : uuid() => each } : {}
    content {
      bypass                     = network_acls.value.bypass
      default_action             = network_acls.value.default_action
      ip_rules                   = network_acls.value.ip_rules
      virtual_network_subnet_ids = network_acls.value.virtual_network_subnet_ids
    }
  }

  dynamic "contact" {
    for_each = can(each.value.contact[0]) ? { for a in each.value.contact : uuid() => a } : {}
    content {
      email = contact.value.email
      name  = can(contact.value.name) ? contact.value.name : null
      phone = can(contact.value.phone) ? contact.value.phone : null
    }
  }

  lifecycle {
    ignore_changes = [tags, name, contact, network_acls]
  }
}

resource "azurerm_role_assignment" "kv_role_assignment_object" {
  for_each             = var.create_kv_role_assignment && local.kv_object != null ? { for each in local.kv_object : "role_assignment_kv" => each } : {}
  principal_id         = data.azurerm_client_config.current.object_id
  scope                = azurerm_key_vault.vm_kv_object["kv_object"].id
  role_definition_name = "Key Vault Administrator"

  depends_on = [azurerm_windows_virtual_machine.vm_windows_object, azurerm_linux_virtual_machine.vm_linux_object, azurerm_key_vault.vm_kv_object]

  lifecycle {
    ignore_changes = [principal_id, scope]
  }
}

resource "azurerm_key_vault_secret" "kv_vm_secret_object" {
  count        = var.create_kv_for_vms || var.kv_object != null || var.kv_resource_id != null ? length(local.vm_objects) : 0
  name         = "${values(local.vm_objects)[count.index].name}-secret"
  value        = values(local.vm_objects)[count.index].admin_password
  key_vault_id = local.kv_resource_id

  depends_on = [azurerm_role_assignment.kv_role_assignment_object]
}