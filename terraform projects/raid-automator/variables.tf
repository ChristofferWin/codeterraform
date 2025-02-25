variable "rg_objects" {
  type = tuple([ object({
    name = string
    location = string
    tags = map(any)
  }) ])

  default = [ {
    name = "rg-raidautomator-ne-mgmt"
    location = "northeurope"
    tags = {
      "environment" = "mgmt"
      "shared_resources" = true
    }
  },
  {
    name = "rg-raidautomator-ne-prod"
    location = "northeurope"
    tags = {
        "environment" = "prod"
        "shared_resources" = false
    }
  } ]
}

variable "vnet_object" {
  type = object({
    name = string
    address_space = set(string)
    subnets = tuple([ object({
      name = string
      address_prefix = set(string) //Will be calculated at run-time
    }) ])
  })
  default = {
    name = "vnet-raidautomator-ne-mgmt"
    address_space = [ "172.16.0.0/16" ]
    subnets = [ {
      name = "snet-pe"
    },
    {
        name = "snet-web"
    } ]
  }
}

variable "p_dns_zones" {
  type = tuple([ string ])
  default = [ "privatelink.azurewebsites.net", "scm.privatelink.azurewebsites.net" ]
}

variable "kv_object" {
  type = object({
    name = string
    sku = string
    soft_delete_days = number
  })
  default = {
    name = "kvraidautomatorneprod"
    sku = "standard"
    soft_delete_days = 7
  }
}

variable "web_object" {
  type = object({
    service_plan_name = string
    sku = string
    os_type = string

    app = object({
      name = string
      
      site_config = object({
        name = ""
      })
    })
  })
}