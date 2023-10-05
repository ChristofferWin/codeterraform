variable "ip_address_spaces" {
  description = "the ip address spaces of the mgmt environments"
  type = list(string)
  default = [ "10.99.0.0/24", "192.168.0.0/24", "172.16.0.0/16", "10.0.0.0/16"]
}

variable "environment_type" { 
  description = "must be exactly one of the following strings: 'dev', 'test', 'prod'"
  type = string

  validation {
    condition = length(regexall("^(dev|test|prod)$", var.environment_type)) > 0
    error_message = "the environment_type provided '${var.environment_type}' did not match any of 'dev||test||prod'"
  }
  default = "dev"
}

variable "use_defaults" {
  description = "switch to determine whether to run in default mode"
  type = bool
  default = true
}

variable "resource_group_name" {
  description = "the resource group to place the resources"
  type = string
  default = null
}

variable "location" {
  description = "where to place the resources"
  type = string
  default = "westeurope"
}

variable "virtual_network_objects" {
  description = "a list of objects representing vnet(s)"
  type = list(object({
    name = string
    address_space = optional(list(string))
    dns_servers = optional(list(string))
  }))
  default = [{
      dns_servers = [ "8.8.8.8", "8.8.4.4", "127.0.0.1"]
      name = "default"
    }
  ]
}

variable "virtual_network_peering_objects" {
  description = "a list of objects representing peering(s)"
  type = list(object({
    name = string
    virtual_network_name = optional(string)
    remote_virtual_network_id = optional(string)
    allow_virtual_network_access = optional(bool)
    allow_forwarded_traffic = optional(bool)
    allow_gateway_transit = optional(bool)
    use_remote_gateways = optional(bool)
  }))
  default = [
    {
      name = "default"
    }
  ]
}

variable "subnet_objects" {
  description = "a list of objects representing subnet(s)"
  type = list(object({
    name = string
    address_prefixes = optional(set(string))
  }))
  default = [
    {
      name = "default"
    }
  ]
}

variable "nsg_objects" {
  description = "a list of objects representing network security group(s)"
  type = list(object({
    name = string
    name_rule = string
    priority = optional(number)
    direction = optional(string)
    access = optional(string)
    protocol = optional(string)
    source_port_ranges = optional(set(string))
    destination_port_ranges = optional(set(string))
    source_address_prefixes = optional(set(string))
    destination_address_prefixes = optional(set(string))
  }))
  default = [ 
    {
      name = "default"
      name_rule = "default"
    } 
  ]
}

variable "bastion_objects" {
  description = "a list of ojects representing bastion instance(s)"
  type = list(object({
    name = string
    copy_paste_enable = optional(bool)
    file_copy_enabled = optional(bool)
    sku = optional(string)
    scale_units = optional(number)
    ip_configuration = optional(list(object({
      name = string
      subnet_id = string
      public_ip_address_id = string
    })))
  }))
  default = [
    {
      name = "default"
    }
  ]
}

variable "public_ip_objects" {
  description = "a list of objects representing public ip(s)"
  type = list(object({
    name = string
    allocation_method = optional(string)
    sku = optional(string)
  }))
  default = [
    {
      name = "bastion-pip"
      allocation_method = "Static"
      sku = "Standard"
    }
  ]
}