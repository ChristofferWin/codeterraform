variable "rg_id" {
  type = string
  default = "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/test-rg"
}

variable "vnet_resource_id" {
  type = string
  default = "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourcegroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-rg"
}

variable "subnet_resource_id" {
  type = string
  default = "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-rg/subnets/vm-tester"
}

variable "subnet_bastion_resource_id" {
  type = string
  default = "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-rg/subnets/AzureBastion"
}

variable "vm_windows_objects" {
  type = any
  default = [
    {
        name = "test-win-vm01"
        os_name = "windows10"

        source_image_reference = {
          offer = "WindowsServer"
          publisher = "MicrosoftWindowsServer"
          sku = "2022-datacenter-smalldisk-g2"
          version = "20348.2340.240303"
        }

        boot_diagnostics = {
          storage_account = {
            name = "testerstorage137"
            public_network_access_enabled = false
            account_tier = "Standard"
            account_replication_type = "RAGRS"
            access_tier = "Cool"

            network_rules = {
              default_action = "Allow"
              bypass = ["Logging"]
              virtual_network_subnet_ids = ["/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/tester2/providers/Microsoft.Network/virtualNetworks/test-vnetter/subnets/subnetforvms"]
              ip_rules = ["212.112.152.31"]
            }
          }
        }

        identity = {
          type = "SystemAssigned"
        }

        os_disk = {
          caching = "ReadOnly"
          disk_size_gb = 1024
        }

        public_ip = {
          name = "win1pip"
          allocation_method = "Static"
          sku = "Standard"
          tags = {
            "demo" = "hero-pip1"
          }
        }

        nic = {
            name = "vm1nic"
            dns_servers = ["8.8.8.8", "8.8.4.4"]
            enable_ip_forwarding = true
            tags = {
              "demo" = "hero-nic1"
            }

            ip_configuration = {
              name = "some-if-config"
              private_ip_address_version = "IPv4"
              private_ip_address = "192.168.0.10"
              private_ip_address_allocation = "Static"
            }
          }
    },
    {
        name = "test-win-vm02"
        os_name = "windows11"
    }
]
}

variable "vm_linux_objects" {
  type = any
  default = [
    {
        name = "test-linux-vm01"
        os_name = "DeBiAn10"
    },
    {
        name = "test-linux-vm02"
        os_name = "DeBiaN11"

        source_image_reference = {
          offer = "Debian-11"
          publisher = "Debian"
          sku = "11-gen2"
          version = "0.20240211.1654"
        }

        boot_diagnostics = {
          storage_account = {
            name = "testerstorage138"
            public_network_access_enabled = false
            account_tier = "Standard"
            account_replication_type = "RAGRS"
            access_tier = "Cool"

            network_rules = {
              default_action = "Allow"
              bypass = ["Logging"]
              virtual_network_subnet_ids = ["/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/tester2/providers/Microsoft.Network/virtualNetworks/test-vnetter/subnets/subnetforvms"]
              ip_rules = ["212.112.152.31"]
            }
          }
        }

        identity = {
          type = "SystemAssigned"
        }

        os_disk = {
          caching = "None"
          disk_size_gb = 1024
        }

        public_ip = {
          name = "win2pip"
          allocation_method = "Static"
          sku = "Standard"
          tags = {
            "demo" = "hero-pip2"
          }
        }

        nic = {
            name = "vm2nic"
            dns_servers = ["8.8.8.8", "8.8.4.4"]
            enable_ip_forwarding = true
            tags = {
              "demo" = "hero-nic2"
            }

            ip_configuration = {
              name = "some-if-config"
              private_ip_address_version = "IPv4"
              private_ip_address = "192.168.0.11"
              private_ip_address_allocation = "Static"
            }
          }
    }
]
}

variable "script_name" {
  type = string
  default = "./Get-AzVMSKu.ps1"
}

variable "location" {
  type = string
  default = "northeurope"
}

variable "bastion_object" {
  type = any
  default = {
    name = "bastionobject"
    copy_paste_enabled = false
    file_copy_enabled = false
    sku = "Standard"
    scale_units = 4
    tags = {
      "demo" = "hero-bastion"
    }
  }
}

variable "vnet_object" {
  type = any
  default = {
    name = "test-vnetter"
    address_space = ["192.168.0.0/24", "10.0.0.0/24"]
    tags = {
      "demo" = "hero-vnet"
    }
  }
}

variable "subnet_objects" {
  type = any
  default = [
    {
      name = "WILL BE BASTION"
      address_prefixes = ["10.0.0.0/26"]
      tags = {
        "demo" = "hero-subnet"
      }
    },
    {
      name = "subnetforvms"
      address_prefixes = ["192.168.0.0/26"]
      tags = {
        "demo" = "hero-subnet2"
      }
    }
  ]
}

variable "kv_object" {
  type = any
  default = {
    name = "somelv12333"
    sku_name = "premium"
    enabled_for_deployment = true
    enabled_for_disk_encryption = true
    enabled_for_template_deployment = true
    purge_protection_enabled = true
    soft_delete_retention_days = 90
    
    network_acls = {
      bypass = "AzureServices"
      default_action = "Deny"
      ip_rules = ["212.112.152.31"]
    }
  }
}

variable "nsg_objects" {
  type = any
  default = [
    {
      name = "nsg1"
      
      security_rules = [
        {
          name = "rule1"
          priority = 100
          direction = "Inbound"
          protocol = "Tcp"
          source_port_range = null
          destination_port_range = null
          destination_port_ranges = [22, 3389]
          source_port_range = "*"
          source_address_prefix = "192.168.0.0/26"
          destination_address_prefix = "10.10.10.0/24"
          access = "Deny"
        }
      ]
    }
  ]
}