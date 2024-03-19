provider "azurerm" {
  features {
  }
}

module "test" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"
  location = var.location
  rg_name = "tester2"
  vnet_object = var.vnet_object
  subnet_objects = var.subnet_objects
  vm_windows_objects = var.vm_windows_objects
  vm_linux_objects = var.vm_linux_objects
  bastion_object = var.bastion_object
  kv_object = var.kv_object
  create_kv_role_assignment = true
  create_diagnostic_settings = true
  nsg_objects = var.nsg_objects
  script_name = "./terraform projects/modules/azurerm-vm-bundle/test/Get-AzVMSKu.ps1"
}
/*
module "test2" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"
  rg_id = module.test.rg_object.id
  location = "northeurope"
  vm_linux_objects = [
    {
      name = "crm"
      os_name = "deBIAN10"
    }
  ]
  create_public_ip = true
  subnet_resource_id = "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/tester2/providers/Microsoft.Network/virtualNetworks/test-vnetter/subnets/subnetforvms"
  vnet_resource_id = "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/tester2/providers/Microsoft.Network/virtualNetworks/test-vnetter"
  create_diagnostic_settings = true
  create_kv_for_vms = true
  create_kv_role_assignment = true

  depends_on = [ module.test ]
}

module "test3" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"
  rg_name = "test-3-rg"
  location = "westus"
  env_name = "prod"
  create_diagnostic_settings = true
  create_nsg = true

  subnet_objects = [
    {
      name = "for bastion"
      address_prefixes = ["10.99.0.0/24"]
    },
    {
      name = "vm-subnet"
      address_prefixes = ["10.0.1.0/24"]
    }
  ]

  bastion_object = {
    name = "bastioncustom"
    copy_paste_enabled = false
    file_copy_enabled  = false
    sku                = "Basic"
    scale_units        = 2
    tags               = {
      "superhero" = "prod"
    }
  }
  vm_windows_objects = [
    {
      name = "vm1crm"
      os_name = "windows11"

      nic = {
        name = "niccrm"
      }
      }
  ]
}

module "custom_advanced_settings" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=1.3.0"

  rg_name = "custom-advanced-settings2-rg"
  location = "northeurope"

  //Windows 10 with a custom public ip and NIC configurations
  vm_windows_objects = [
    {
      name = "win10"
      os_name = "windows10"

      public_ip = {
        name = "vm-custom-pip"
        allocation_method = "Dynamic"
        sku = "Basic"
        
        tags = {
          "environment" = "prod"
        }
      }

      nic = {
        name = "vm-custom-nic"
        dns_servers = ["8.8.8.8", "8.8.4.4"] //Google DNS
        enable_ip_forwarding = true
        
        ip_configuration = {
          name = "ip-config"
          private_ip_address_version = "IPv4"
          private_ip_address_allocation = "Static"
          private_ip_address = "10.0.0.5" //First possible address in the subnet we are deploying, as Azure takes the first 4 and last 1
        }

        tags = {
          "vm_name" = "win10"
        }
      }
    }
  ]

  vnet_object = {
    name = "custom-with-bastion-vnet"
    address_space = ["10.0.0.0/20"]
  }

  subnet_objects = [
    {
      address_prefixes = ["10.0.10.0/26"]
    },
    {
      name = "custom-vm-subnet"
      address_prefixes = ["10.0.0.0/24"]

      tags = {
        "environment" = "prod"
      }
    },
    {
      //Name wont matter, it will be overwritten as the bastion subnet must have a specific name
      address_prefixes = ["10.0.10.0/26"]

      tags = {
        "environment" = "mgmt"
      }
    }
  ]

  bastion_object = {
    name = "custom-bastion" //must contain 'bastion'
    copy_paste_enabled = true
    file_copy_enabled = true
    sku = "Standard"
    scale_units = 5

    tags = {
      "environment" = "mgmt"
    }
  }
}
*/ 