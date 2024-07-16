variable "topology_object" {
  description = "a list of objects describing information about the hub and spoke environments"
  type = object({
    project_name = optional(string)
    location = optional(string)
    name_prefix = optional(string)
    name_suffix = optional(string)
    env_name = optional(string)
    #create_tag = optional(bool)
    #multiplicator = optional(number)
    dns_servers = optional(list(string))
    tags = optional(map(string))
    subnets_cidr_notation = optional(string)

    hub_object = object({
      rg_name = optional(string)
      location = optional(string)
      tags = optional(map(string))

      network = object({
        vnet_name = optional(string)
        vnet_cidr_notation = optional(string)
        address_spaces = optional(list(string))
        dns_servers = optional(list(string))
        tags = optional(map(string))
        vnet_resource_id = optional(string)
        vnet_spoke_address_spaces = optional(list(string))
        vnet_peering_name = optional(string)
        vnet_peering_allow_virtual_network_access = optional(bool)
        vnet_peering_allow_forwarded_traffic = optional(bool)
        fw_resource_id = optional(string)
        fw_private_ip = optional(string)

        vpn = optional(object({
          gw_name = optional(string)
          address_space = optional(list(string))
          gw_sku = optional(string)
          pip_name = optional(string)
          pip_ddos_protection_mode = optional(string)
          tags = optional(map(string))
        }))

        firewall = optional(object({
          name = optional(string)
          sku_tier = optional(string)
          threat_intel_mode = optional(bool)
          pip_name = optional(string)
          pip_ddos_protection_mode = optional(string)
          log_name = optional(string)
          log_diag_name = optional(string)
          log_daily_quota_gb = optional(number)
          no_logs = optional(bool)
          no_internet = optional(bool)
          no_rules = optional(bool)
          tags = optional(map(string))
        }))

        subnet_objects = optional(list(object({
          name = optional(string)
          use_first_subnet = optional(bool)
          use_last_subnet = optional(bool)
          address_prefix = optional(list(string))
          service_endpoints = optional(set(string))
          service_endpoint_policy_ids = optional(set(string))

          delegation = optional(list(object({
            name = optional(string)
            service_name_pattern = optional(string)
          })))
        })))

        ddos_protection_plan = optional(object({
          name = optional(string)
          id = optional(string)
          enable = optional(bool)
        }))
      })
    })

    spoke_objects = optional(list(object({
      rg_name = optional(string)
      location = optional(string)
      tags = optional(map(string))

      network = object({
        vnet_name = optional(string)
        address_spaces = optional(list(string))
        dns_servers = optional(list(string))
        tags = optional(map(string))
        vnet_peering_name = optional(string)
        vnet_peering_allow_virtual_network_access = optional(bool)
        vnet_peering_allow_forwarded_traffic = optional(bool)

        ddos_protection_plan = optional(object({
          id = string
          enable = bool
        }))
      
        subnet_objects = list(object({
          name = optional(string)
          use_first_subnet = optional(bool)
          use_last_subnet = optional(bool)
          address_prefix = optional(list(string))
          service_endpoints = optional(set(string))
          service_endpoint_policy_ids = optional(set(string))

          delegation = optional(list(object({
            name = optional(string)
            service_name_pattern = optional(string)
          })))
        }))
      })
  })))})
  /*
  validation {
    condition = var.topology_object.hub_object.network.vnet_spoke_resource_ids != null && var.topology_object.hub_object.network.vnet_resource_id != null
    error_message = "both the 'hub_object.network' attributes: 'vnet_spoke_resource_ids' and 'vnet_resource_id' cannot be set at the same time"
  }

  validation {
    condition = var.topology_object.env_name != null && var.topology_object.project_name == null || var.topology_object.env_name == null && var.topology_object.project_name != null
    error_message = "whenever the root attribute 'env_name' or 'project_name' is set, both attributes must be defined"
  }

  validation {
    condition = var.topology_object.name_prefix != null && var.topology_object.name_suffix != null
    error_message = "both the attributes: 'name_prefix' and 'name_suffix' cannot be set at the same time"
  }
  */
}