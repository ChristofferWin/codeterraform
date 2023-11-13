variable "rg_name" {
  description = "the name of resource group to put the vm bundle in. use this variable to create a new resource group to put the vm bundle in. either rg_id or this variable must be specified"
  type = string
  default = null
}

variable "rg_id" {
  description = "the resource id of the resource group of which to put the vm bundle in. either rg_name or this variable must be specified."
  type = string
  default = null
}

variable "location" {
  description = "name of the location to put the resources. defaults to 'westeurope'"
  type = string
  default = "westeurope"
}

variable "env_name" {
  description = "the name of the environment. can be any string value, will be used as a prefix in every resource name that does not have an explicit name defined"
  type = string
  default = null
}

variable "create_bastion" {
  description = "switch to determine whether the module shall deploy bastion"
  type = bool
  default = false
}

variable "create_nsg" {
  description = "switch to determine whether the module shall deploy default nsgs for vm subnet"
  type = bool
  default = false
}

variable "create_public_ip" {
  description = "switch to determine whether the module shall deploy a public ip for the vm"
  type = bool
  default = false
}

variable "create_diagnostic_settings" {
  description = "switch to determine whether the module shall deploy a storage account and configure diag settings"
  type = bool
  default = false
}

variable "vnet_object" {
  description = "an object defining the vnet address spaces in format [x.x.x.x/x] and its name. must be at least /24 in case bastion or vpn is also enabled"
  type = object({
    name = string
    address_space = list(string)
    tags = optional(map(string))
  })
  default = null
}

variable "vnet_resource_id" {
  description = "in case the module is not to create a new vnet, specify the id of vnet of which to use. if this is specified, the subnet id must also be specified"
  type = string
  default = null
}

variable "subnet_objects" {
  description = "define up to 2 subnets. 1 for the vm(s), another for bastion. index 0 will always be the vm subnet. name is not required and will be 'vm-subnet' by default. note, the bastion subnet name cannot be changed"
  type = list(object({
    name = string
    address_prefixes = list(string)
  }))
  default = null
}

variable "subnet_resource_id" {
  description = "in case the module is not to create a new subnet, specify the id of the subnet of which to use. if this is specified, the vnet id must also be specified"
  type = string
  default = null
}

variable "pip_objects" {
  description = "a list of objects representing public ips to create. must have the same length as the total length of 'vm_windows_objects & 'vm_linux_objects'"
  type = list(object({
    name = string
    allocation_method = optional(string)
    sku = optional(string)
    tags = optional(map(string))
  }))
  default = null
}

variable "bastion_object" {
  description = "define a custom bastion configuration"
  type = object({
    name = string
    copy_paste_enabled = optional(bool)
    file_copy_enabled = optional(bool)
    sku = optional(string)
    scale_units = optional(number)
    tags = optional(map(string))
  })
  default = null
}

variable "nsg_objects" {
  description = "a list of objects representing network security groups for subnets"
  type = list(object({
    name = string
    subnet_id = optional(string)
    tags = optional(map(string))

    security_rules = optional(list(object({
      name = string
      priority = optional(number)
      direction = optional(string)
      access = optional(string)
      protocol = optional(string)
      source_port_range = optional(any)
      source_port_ranges = optional(any)
      destination_port_range = optional(any)
      destination_port_ranges = optional(any)
      source_address_prefix = optional(any)
      destination_address_prefix = optional(any)
    })))
  }))
  default = null
}

variable "vm_windows_objects" {
  description = "a list of objects representing a windows vm configuration"
  type = list(object({
    name = string
    admin_username = optional(string)
    admin_password = optional(string)
    allow_null_version = optional(bool)
    size = optional(string)
    size_pattern = optional(string)
    allow_extension_operations = optional(bool)
    availability_set_id = optional(string)
    bypass_platform_safety_checks_on_user_schedule_enabled = optional(bool)
    capacity_reservation_group_id = optional(string)
    computer_name = optional(string)
    custom_data = optional(string)
    dedicated_host_id = optional(string)
    dedicated_host_group_id = optional(string)
    edge_zone = optional(string)
    enable_automatic_updates = optional(bool)
    encryption_at_host_enabled = optional(bool)
    eviction_policy = optional(string)
    extensions_time_budget = optional(string)
    hotpatching_enabled = optional(bool)
    license_type = optional(string)
    max_bid_price = optional(number)
    os_name = string
    patch_assessment_mode = optional(string)
    patch_mode = optional(string)
    platform_fault_domain = optional(number)
    priority = optional(string)
    provisioning_vm_agent = optional(bool)
    proximity_placement_group_id = optional(string)
    reboot_setting = optional(string)
    source_image_id = optional(string)
    tags = optional(map(string))
    timezone = optional(string)
    virtual_machine_scale_set_id = optional(string)
    vtpm_enabled = optional(bool)
    zone = optional(string)
    secure_boot_enabled = optional(bool)
  
    boot_diagnostics = optional(object({
      storage_account = optional(object({
      name = string
      access_tier = optional(string)
      public_network_access_enabled = optional(bool)
      account_tier = optional(string)
      account_replication_type = optional(string)

      network_rules = optional(object({
        default_action = optional(string)
        bypass = optional(set(string))
        virtual_network_subnet_ids = optional(set(string))
        ip_rules = optional(set(string))

        private_link_access = optional(list(object({
          endpoint_resource_id = string
          endpoint_tenant_id = optional(string)
        })))
      }))
    }))
  }))

    additional_capabilities = optional(object({
      ultra_ssd_enabled = bool
    }))

    additional_unattend_content = optional(list(object({
      content = string
      setting = string
    })))

    gallery_application = optional(list(object({
      version_id = string
      configuration_blob_uri = optional(string)
      order = optional(number)
      tag = optional(string)
    })))

    identity = optional(object({
      type = string
      identity_ids = optional(set(string))
    }))

    os_disk = optional(object({
      caching = string
      storage_account_type = optional(string)
      disk_encryption_set_id = optional(string)
      disk_size_gb = optional(number)
      name = optional(string)
      source_vm_disk_encryption_set_id = optional(string)
      security_encryption_type = optional(string)
      secure_vm_disk_encryption_set_id = optional(string)
      write_accelerator_enabled = optional(bool)

      diff_disk_settings = optional(object({
        option = string
        placement = optional(string)
      }))
    }))

    secret = optional(list(object({
      key_vault_id = string

      certificate = optional(list(object({
        store = string
        url = string
      })))
    })))

    source_image_reference = optional(object({
      publisher = string
      offer = string
      sku = string
      version = string
    }))

    termination_notification = optional(object({
      enabled = bool
      timeout = optional(string)
    }))

    winrm_listener = optional(list(object({
      protocol = string
      certificate_url = optional(string)
    })))

    public_ip = optional(object({
      name = string
      allocation_method = string
      sku = optional(string)
      tags = optional(map(string))
    }))

    nic = optional(object({
      name = string
      dns_servers = optional(list(string))
      enable_ip_forwarding = optional(bool)
      edge_zone = optional(string)
      tags = optional(map(string))
      ip_configuration = optional(object({
        name = string
        private_ip_address_version = optional(string)
        private_ip_address = string
        private_ip_address_allocation = optional(string)
      }))
    }))

    nsg = optional(object({
      
    }))
  }))
  default = null
}

variable "vm_linux_objects" {
  description = "a list of objects representing a linux vm configuration"
  type = list(object({
    name = optional(string)
    admin_username = optional(string)
    admin_password = optional(string)
    allow_null_version = optional(bool)
    license_type = optional(string)
    size = optional(string)
    size_pattern = optional(string)
    allow_extension_operations = optional(bool)
    availability_set_id = optional(string)
    bypass_platform_safety_checks_on_user_schedule_enabled = optional(bool)
    capacity_reservation_group_id = optional(string)
    computer_name = optional(string)
    custom_data = optional(string)
    dedicated_host_id = optional(string)
    dedicated_host_group_id = optional(string)
    disable_password_authentication = optional(bool)
    edge_zone = optional(string)
    encryption_at_host_enabled = optional(bool)
    eviction_policy = optional(string)
    extensions_time_budget = optional(string)
    patch_assessment_mode = optional(string)
    patch_mode = optional(string)
    max_bid_price = optional(number)
    os_name = string
    platform_fault_domain = optional(number)
    priority = optional(string)
    provisioning_vm_agent = optional(bool)
    proximity_placement_group_id = optional(string)
    reboot_setting = optional(string)
    secure_boot_enabled = optional(bool)
    source_image_id = optional(string)
    tags = optional(map(string))
    user_data = optional(string)
    vtpm_enabled = optional(bool)
    virtual_machine_scale_set_id = optional(string)
    zone = optional(string)
    boot_diagnostic = optional(bool)

    additional_capabilities = optional(object({
      ultra_ssd_enabled = bool
    }))

    admin_ssh_key = optional(list(object({
      public_key = string
      username = string
    })))

    boot_diagnostics = optional(object({
      storage_account = optional(object({
        name = string
        access_tier = optional(string)
        public_network_access_enabled = optional(bool)
        account_tier = optional(string)
        account_replication_type = optional(string)

        network_rules = optional(object({
          default_action = optional(string)
          bypass = optional(set(string))
          virtual_network_subnet_ids = optional(set(string))
          ip_rules = optional(set(string))

          private_link_access = optional(list(object({
            endpoint_resource_id = string
            endpoint_tenant_id = optional(string)
          })))
        }))
      }))
    }))

    gallery_application = optional(list(object({
      version_id = string
      configuration_blob_uri = optional(string)
      order = optional(number)
      tag = optional(string)
    })))

    identity = optional(object({
      type = string
      identity_ids = optional(set(string))
    }))

    os_disk = optional(object({
      name = optional(string)
      caching = optional(string)
      storage_account_type = optional(string)
      disk_encryption_set_id = optional(string)
      disk_size_gb = optional(number)
      secure_vm_disk_encryption_set_id = optional(string)
      security_encryption_type = optional(string)
      write_accelerator_enabled = optional(bool)

      diff_disk_settings = optional(object({
        option = string
        placement = optional(string)
      }))
    }))

   plan = optional(object({
      name = string
      product = string
      publisher = string
   }))

    secret = optional(object({
      key_vault_id = string

      certificate = optional(list(object({
        url = string
      })))
    }))

    source_image_reference = optional(object({
      publisher = string
      offer = string
      sku = string
      version = string
    }))

    termination_notification = optional(object({
      enabled = bool
      timeout = optional(string)
    }))

    public_ip = optional(object({
      name = string
      allocation_method = string
      sku = optional(string)
      tags = optional(map(string))
    }))

    nic = optional(object({
      name = string
      dns_servers = optional(list(string))
      enable_ip_forwarding = optional(bool)
      edge_zone = optional(string)
      tags = optional(map(string))
    }))
  }))
  default = null
}

variable "script_name" {
  description = "define a custom path for the powershell script that will retrieve sku information"
  type = string
  default = null
}