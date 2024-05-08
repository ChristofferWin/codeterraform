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
    cidr_notation = optional(string)
    tags = optional(map(string))

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

        subnet_objects = optional(list(object({
          name = optional(string)
          use_first_subnet = optional(bool)
          use_last_subnet = optional(bool)
          cidr_notation = optional(string)
          address_prefixes = optional(list(string))

      delegation = optional(list(object({
        name = optional(string)
        service_name = optional(string)
        actions = optional(list(string))
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
      vnet_name = string
      vnet_cidr_notation = optional(string)
      address_spaces = optional(list(string))
      dns_servers = optional(list(string))
      tags = optional(map(string))

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

      delegation = optional(list(object({
        name = optional(string)
        service_name = optional(string)
        actions = optional(list(string))
      })))
    })))
      }))
    }))
  })
  default = {
    multiplicator = 5
    dns_servers = [ "7.7.7.7" ]
    tags = {
      "hello" = "world"
    }

    hub_object = {
      location = "westus"

      network = {
        subnet_objects = [
          {
            name = "tester1337"
          }
        ]
      }
    }

    spoke_objects = [ {
      location = "westus"
    },
    {
      location = "eastus"
      solution_name = "solution"

      network = {
        vnet_name = "test123"
        address_spaces = ["192.168.0.0/24"]

        subnet_objects = [
          {
            use_first_subnet = true
          }
        ]

        ddos_protection_plan = {
          id = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/group1/providers/Microsoft.Network/ddosProtectionPlans/testddospplan"
          enable = true
        }
      }
    },
    {
      location = "northeurope"
      }
    ]
  }
}

variable "subnet_objects" {
  description = "a list of objects describing subnets to create - this list can contain both subnets for the hub and any spoke vnet"
  type = list(object({
    name = optional(string)
    address_prefixes = optional(list(string))
    solution_name = optional(string)
    use_first_subnet = optional(bool)
    use_last_subnet = optional(bool)

    delegation = optional(list(object({
      name = optional(string)
      service_name = optional(string)
      actions = optional(list(string))
    })))
  }))
  default = null
}