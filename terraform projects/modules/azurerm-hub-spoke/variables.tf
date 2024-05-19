variable "typology_object" {
  description = "a list of objects describing information about the hub and spoke environments"
  type = object({
    customer_name = optional(string)
    location = optional(string)
    name_prefix = optional(string)
    name_suffix = optional(string)
    env_name = optional(string)
    create_tag = optional(bool)
    multiplicator = optional(number)
    dns_servers = optional(list(string))
    tags = optional(map(string))
    address_spaces = optional(list(string))
    subnets_cidr_notation = optional(string)

    hub_object = optional(object({
      rg_name = optional(string)
      location = optional(string)
      tags = optional(map(string))

      network = optional(object({
        vnet_name = optional(string)
        vnet_cidr_notation = optional(string)
        address_spaces = optional(list(string))
        dns_servers = optional(list(string))
        tags = optional(map(string))
        vnet_peering_name = optional(string)
        vnet_peering_allow_virtual_network_access = optional(bool)
        vnet_peering_allow_forwarded_traffic = optional(bool)

        vpn = optional(object({
          gw_name = optional(string)
          address_space = optional(list(string))
          gw_sku = optional(string)
          pip_name = optional(string)
          pip_ddos_protection_mode = optional(string)
        }))

        firewall = optional(object({
          name = optional(string)
          sku_tier = optional(string)
          threat_intel_mode = optional(bool)
          pip_name = optional(string)
          pip_ddos_protection_mode = optional(string)
        }))

        subnet_objects = optional(list(object({
          name = optional(string)
          use_first_subnet = optional(bool)
          use_last_subnet = optional(bool)
          cidr_notation = optional(string)
          address_prefixes = optional(list(string))
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
      }))
    }))

    spoke_objects = list(object({
      rg_name = optional(string)
      location = optional(string)
      tags = optional(map(string))
      solution_name = optional(string)

    network = optional(object({
      vnet_name = optional(string)
      vnet_cidr_notation = optional(string)
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
      
      subnet_objects = optional(list(object({
        name = optional(string)
        use_first_subnet = optional(bool)
        use_last_subnet = optional(bool)
        cidr_notation = optional(string)
        address_prefixes = optional(list(string))
        service_endpoints = optional(set(string))
        service_endpoint_policy_ids = optional(set(string))

        delegation = optional(list(object({
          name = optional(string)
          service_name_pattern = optional(string)
        })))
      })))
    }))
  }))})
  default = {
    multiplicator = 5
    location = "westeurope"
    dns_servers = [ "8.8.8.8" ]
    tags = {
      "hello" = "world"
    }

    hub_object = {
      location = "westus"

      network = { 
        vpn = {}

        subnet_objects = [
          {
            name = "GatewaySubnet"
            use_last_subnet = true
          },
          {
            name = "AzureBastionSubnet"
          }
        ]
      }
    }

    spoke_objects = [ {
      location = "westus"

      network = {
        subnet_objects = [
          {
            
          }
        ]
      }
    },
    {
      location = "eastus"

      network = {
      address_spaces = [ "172.16.0.0/22" ]
        subnet_objects = [
          {
          },
          {
          },
          {
            delegation = [{
              service_name_pattern = "ApiManagement"
            }]
          }
        ]
      }
    }
    ]
  }
}