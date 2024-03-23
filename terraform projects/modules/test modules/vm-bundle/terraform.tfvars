location = "northeurope"

rg_name = "simple-test-vm-rg"

vnet_object = {
  name = "vm-bundle-vnet"
  address_space = ["172.20.0.0/20"]
}

subnet_objects = [
  {
    name = "vm-bundle-subnet"
    address_prefixes = ["172.20.0.0/24"]
  }
]

subnet_objects_with_bastion = [
  {
    name = "vm-bundle-subnet"
    address_prefixes = ["172.20.0.0/24"]
  },
  {
    address_prefixes = ["172.16.99.0/26"]
  }
]

vm_windows_objects_simple = [
  {
    name = "win-vm-01"
    os_name = "WiNDoWs10"
  },
  {
    name = "win-vm-02"
    os_name = "WiNDoWs11"
  }
]

vm_linux_objects_simple = [
  {
    name = "linux-vm-01"
    os_name = "DeBiAN10"
  },
  {
    name = "linux-vm-02"
    os_name = "DeBiAn11"
  }
]

vm_windows_objects_custom_config = [
  {
    name = "customwin1"
    os_name = "windowS10"
    admin_username = "testeradmin"
    admin_password = "DoESNoTMaTTer!123"
    size_pattern = "D4"
    allow_extension_operations = true
    secure_boot_enabled = true
    
    tags = {
      "env" = "vm-bundle"
    }

    boot_diagnostics = {
      
      storage_account = {
        name = "tester1331"
        access_tier = "Hot"
        public_network_access_enabled = true
        account_tier = "Standard"
        account_replication_type = "RAGRS"
      }
    }

    os_disk = {
      name = "customdisk"
      caching = "Read"
      disk_size_gb = 1024
      write_accelerator_enabled = true
    }

    source_image_reference = {
      publisher = "MicrosoftWindowsDesktop"
      offer     = "Windows-10"
      sku       = "win10-22h2-pron-g2"
      version   = "19045.3693.231109"
    }

    public_ip = {
      name = "customwinpip1"
      allocation_method = "Static"
      sku = "Standard"

      tags = {
        "env" = "vm-bundle"
      }
    }

    nic = {
      name = "customwinnic1"
      dns_servers          = ["8.8.8.8", "8.8.4.4"]
      enable_ip_forwarding = true
      tags                 = {
        "env" = "vm-bundle"
      }

      ip_configuration = {
        name                          = "custom-config"
        private_ip_address_version    = "IPv4"
        private_ip_address            = "172.20.0.100"
        private_ip_address_allocation = "Static"
      }
    }
  },
  {
    name = "customwin2"
    os_name = "windowS11"
    admin_username = "testeradmin"
    admin_password = "DoESNoTMaTTer!123"
    size_pattern = "D4"
    allow_extension_operations = true
    secure_boot_enabled = true
    
    tags = {
      "env" = "vm-bundle"
    }

    boot_diagnostics = {
      
      storage_account = {
        name = "tester1332"
        access_tier = "Hot"
        public_network_access_enabled = true
        account_tier = "Standard"
        account_replication_type = "RAGRS"
      }
    }

    os_disk = {
      name = "customdisk"
      caching = "Read"
      disk_size_gb = 1024
      write_accelerator_enabled = true
    }

    source_image_reference = {
      publisher = "MicrosoftWindowsDesktop"
      offer     = "Windows-11"
      sku       = "win11-23h2-pron"
      version   = "22631.2715.231109"
    }

    public_ip = {
      name = "customwinpip2"
      allocation_method = "Static"
      sku = "Standard"

      tags = {
        "env" = "vm-bundle"
      }
    }

    nic = {
      name = "customwinnic2"
      dns_servers          = ["8.8.8.8", "8.8.4.4"]
      enable_ip_forwarding = true
      tags                 = {
        "env" = "vm-bundle"
      }

      ip_configuration = {
        name                          = "custom-config"
        private_ip_address_version    = "IPv4"
        private_ip_address            = "172.20.0.101"
        private_ip_address_allocation = "Static"
      }
    }
  }
]

vm_linux_objects_custom_config = [
  {
    name = "customlinux1"
    os_name = "DeBiAn10"
    size_pattern = "B1"
    allow_extension_operations = true
    secure_boot_enabled = true
    
    tags = {
      "env" = "vm-bundle"
    }

    boot_diagnostics = {
      
      storage_account = {
        name = "tester1334"
        access_tier = "Hot"
        public_network_access_enabled = true
        account_tier = "Standard"
        account_replication_type = "RAGRS"
      }
    }

    os_disk = {
      name = "customdisk"
      caching = "Read"
      disk_size_gb = 1024
      write_accelerator_enabled = true
    }

    source_image_reference = {
      publisher = "Debian"
      offer     = "Debian-10"
      sku       = "10-gen2"
      version   = "0.20200210.166"
    }

    public_ip = {
      name = "customlinuxpip1"
      allocation_method = "Static"
      sku = "Standard"

      tags = {
        "env" = "vm-bundle"
      }
    }

    nic = {
      name = "customlinuxnic1"
      dns_servers          = ["8.8.8.8", "8.8.4.4"]
      enable_ip_forwarding = true
      tags                 = {
        "env" = "vm-bundle"
      }

      ip_configuration = {
        name                          = "custom-config"
        private_ip_address_version    = "IPv4"
        private_ip_address            = "172.20.0.102"
        private_ip_address_allocation = "Static"
      }
    }

    admin_ssh_key = [
      {
        public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDjm7vUE6KhuZN3yWT+JirtSI62YsNyywvf6//IjTVQq/SLLfybSDerV9LsyHG7VaqAGqLGLfjwGDdGaSB++Tm9qfWne5oh0cS2wscHoCzzt1/3pBd8C1cq9GmWnVo5rAdHnRp/XUvVFortwR0DnIOvVnMJxK1mpnnHwLdqWmyb7msZhizc6T+ipzN2V7oYY01gbndsn0+ZYkBSWz22eEZoMRDUdgiE+ZeMnCRZLSMxIDSK+6cxaE7L+MFJU45KMPcvdD3ZM/WKiZl2knNbdJbuytOESyWgDxfnDMVO9YztH3sHRlIf1a/COfc7sKgQH0vXFf9GU0Uzf24pW9D9OdlJ"
        username   = "testeradmin"
      }
    ]
  },
  {
    name = "customlinux2"
    os_name = "DeBiAn11"
    size_pattern = "B1"
    allow_extension_operations = true
    secure_boot_enabled = true
    
    tags = {
      "env" = "vm-bundle"
    }

    boot_diagnostics = {
      
      storage_account = {
        name = "tester1335"
        access_tier = "Hot"
        public_network_access_enabled = true
        account_tier = "Standard"
        account_replication_type = "RAGRS"
      }
    }

    os_disk = {
      name = "customdisk"
      caching = "Read"
      disk_size_gb = 1024
      write_accelerator_enabled = true
    }

    source_image_reference = {
      publisher = "Debian"
      offer     = "Debian-11"
      sku       = "11-gen2"
      version   = "0.20210814.734"
    }

    public_ip = {
      name = "customlinuxpip2"
      allocation_method = "Static"
      sku = "Standard"

      tags = {
        "env" = "vm-bundle"
      }
    }

    nic = {
      name = "customlinuxnic2"
      dns_servers          = ["8.8.8.8", "8.8.4.4"]
      enable_ip_forwarding = true
      tags                 = {
        "env" = "vm-bundle"
      }

      ip_configuration = {
        name                          = "custom-config"
        private_ip_address_version    = "IPv4"
        private_ip_address            = "172.20.0.103"
        private_ip_address_allocation = "Static"
      }
    }

    admin_ssh_key = [
      {
        public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDjm7vUE6KhuZN3yWT+JirtSI62YsNyywvf6//IjTVQq/SLLfybSDerV9LsyHG7VaqAGqLGLfjwGDdGaSB++Tm9qfWne5oh0cS2wscHoCzzt1/3pBd8C1cq9GmWnVo5rAdHnRp/XUvVFortwR0DnIOvVnMJxK1mpnnHwLdqWmyb7msZhizc6T+ipzN2V7oYY01gbndsn0+ZYkBSWz22eEZoMRDUdgiE+ZeMnCRZLSMxIDSK+6cxaE7L+MFJU45KMPcvdD3ZM/WKiZl2knNbdJbuytOESyWgDxfnDMVO9YztH3sHRlIf1a/COfc7sKgQH0vXFf9GU0Uzf24pW9D9OdlJ"
        username   = "testeradmin"
      }
    ]
  }
]