terraform {
  required_providers {
    azurerm = {
      source = "hashicorp/azurerm"
    }
  }
}

provider "azurerm" {
  features {
  }
}
/*
resource "azurerm_storage_account" "storage_object" {
  name = "test12123"
  resource_group_name = local.rg_name
  location = "westeurope"
  account_tier = "Standard"
  account_kind = "BlobStorage"
  account_replication_type = "LRS"
}

resource "azurerm_key_vault" "kv_object" {
  name = "test2123-kv"
  sku_name = "standard"
  tenant_id = "16cb613f-419e-4b3e-b37e-27c5a3c54bcf"
  resource_group_name = local.rg_name
  location = "westeurope"
  enable_rbac_authorization = true
}

resource "azurerm_role_assignment" "assignment_kv_object" {
  scope = azurerm_key_vault.kv_object.id
  role_definition_name = "Key Vault Administrator"
  principal_id = "adb5291a-282d-43b0-b3b4-1d631d480f0e"
}

resource "azurerm_key_vault_certificate" "kv_cert_object" {
  name = "testcert"
  key_vault_id = azurerm_key_vault.kv_object.id

  certificate_policy {
    
    issuer_parameters {
      name = "Self"
    }

    key_properties {
      exportable = true
      key_size = 2048
      key_type = "RSA"
      reuse_key = true
    }

    secret_properties {
      content_type = "application/x-pkcs12"
    }

    x509_certificate_properties {
      extended_key_usage = ["1.3.6.1.5.5.7.3.1"]

      key_usage = [
        "cRLSign",
        "dataEncipherment",
        "digitalSignature",
        "keyAgreement",
        "keyCertSign",
        "keyEncipherment",
      ]

      subject_alternative_names {
        dns_names = ["helloworld.com"]
      }

      subject = "CN=hello-world"
      validity_in_months = 12
    }
  }
  depends_on = [ azurerm_role_assignment.assignment_kv_object ]
}
*/
/*
module "test_vms" {
    source = "../../azurerm-vm-bundle"
    rg_id = local.rg_id
    vm_windows_objects = [
        {
          name = "Windows10Machin"
          os_name = "WINDOWS10"

          allow_extension_operations = true
        },
        {
        name = "Windows10VM"
        os_name = "Windows10"
        allow_extension_operations = true
        bypass_platform_safety_checks_on_user_schedule_enabled = true
        computer_name = "tester"
        disable_password_authentication = true
        eviction_policy = "Deallocate"
        extensions_time_budget = "PT1H15M"
        patch_mode = "AutomaticByPlatform"
        secure_boot_enabled = true
        vtpm_enabled = true
        priority = "Spot"

        termination_notification = {
          enabled = true
          timeout = "PT10M"
        }

        os_disk = {
          name = "testervm-os-disk"
          caching = "ReadWrite"
          disk_size_gb = "1000"
          security_encryption_type = "DiskWithVMGuestState"
          write_accelerator_enabled = true
        }

        source_image_reference = {
          offer = "WindowsServer"
          sku = "2016-datacenter-server-core-g2"
          version = "14393.6351.231007"
          publisher = "MicrosoftWindowsServer"
        }

        identity = {
          type = "SystemAssigned"
        }

        boot_diagnostics = {
          storage_account = {
            name = "win123storage1ie"
            access_tier = "Hot"
            account_tier = "Premium"

            network_rules = {
              bypass = ["AzureServices"]
              ip_rules = ["85.83.136.22/32"]
              
              private_link_access = [
                {
                  endpoint_resource_id = "/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/test2-rg/providers/Microsoft.Network/networkInterfaces/test-vm02346"
                }
              ]
            }
          }
        }
      }
    ]
    vm_linux_objects = [
      {
        name = "UbuntuVM"
        os_name = "ubuntu"
        allow_null_version = true
        allow_extension_operations = true
        bypass_platform_safety_checks_on_user_schedule_enabled = true
        computer_name = "tester"
        disable_password_authentication = true
        eviction_policy = "Deallocate"
        extensions_time_budget = "PT1H15M"
        patch_mode = "AutomaticByPlatform"
        secure_boot_enabled = true
        vtpm_enabled = true
        priority = "Spot"

        termination_notification = {
          enabled = true
          timeout = "PT10M"
        }

        os_disk = {
          name = "testervm-os-disk"
          caching = "ReadWrite"
          disk_size_gb = "1000"
          security_encryption_type = "DiskWithVMGuestState"
          write_accelerator_enabled = true
        }

        source_image_reference = {
          offer = "UbuntuServer"
          sku = "19_10-daily-gen2"
          version = "19.10.202007100"
          publisher = "Canonical"
        }

        identity = {
          type = "SystemAssigned"
        }

        admin_ssh_key = [
          {
            public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDjm7vUE6KhuZN3yWT+JirtSI62YsNyywvf6//IjTVQq/SLLfybSDerV9LsyHG7VaqAGqLGLfjwGDdGaSB++Tm9qfWne5oh0cS2wscHoCzzt1/3pBd8C1cq9GmWnVo5rAdHnRp/XUvVFortwR0DnIOvVnMJxK1mpnnHwLdqWmyb7msZhizc6T+ipzN2V7oYY01gbndsn0+ZYkBSWz22eEZoMRDUdgiE+ZeMnCRZLSMxIDSK+6cxaE7L+MFJU45KMPcvdD3ZM/WKiZl2knNbdJbuytOESyWgDxfnDMVO9YztH3sHRlIf1a/COfc7sKgQH0vXFf9GU0Uzf24pW9D9OdlJ"
            username = "localadmin"
          }
        ]

        boot_diagnostics = {
          storage_account = {
            name = "ubuntustorage1ie"
            access_tier = "Hot"
            account_tier = "Premium"

            network_rules = {
              bypass = ["AzureServices"]
              ip_rules = ["85.83.136.22/32"]
              
              private_link_access = [
                {
                  endpoint_resource_id = "/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/test2-rg/providers/Microsoft.Network/networkInterfaces/test-vm02346"
                }
              ]
            }
          }
        }



      }
    ]
    create_public_ip = true
    create_nsg = true
}

output "test" {
  value = module.test_vms.summary_object
}




/*
module "test2_vms" {
  source = "../../azurerm-vm-bundle"
  rg_id = "/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourcegroups/rg-test3"
  vnet_resource_id = "/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourcegroups/rg-test3/providers/Microsoft.Network/virtualNetworks/test-vnet3"
  subnet_resource_id = "/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/rg-test3/providers/Microsoft.Network/virtualNetworks/test-vnet3/subnets/default"

  vnet_object = {
    address_space = ["10.0.0.0/24"]
    name = "vnet-vms"
  }

  vm_linux_objects = [
    {
      os_name = "ubuntu"
      name = "testubuntu"
    }
  ]
}
*/
/*
module "test3_vms" {
  source = "../../azurerm-vm-bundle"
  rg_name = "test5-rg"
  
  vm_windows_objects = [
    {
      name = "windows10-vm"
      os_name = "windows10"

      public_ip = {
        allocation_method = "Static"
        name = "windows10-vm-pip"
        sku = "Standard"
      }

      source_image_reference = {
        offer = "Windows-10"
        version = "19045.3570.231001"
        sku = "win10-22h2-pron-g2"
        publisher = "MicrosoftWindowsDesktop"
      }

      boot_diagnostics = {
        storage_account = {
          name = "win1337stor"
        }
      }
    }
  ]

  vm_linux_objects = [
    {
      name = "test-vm"
      os_name = "ubuntu"

      public_ip = {
        name = "ubuntupip"
        sku = "Basic"
        allocation_method = "Static"
      }

      admin_ssh_key = [
        {
          username = "localadmin"
          public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDjm7vUE6KhuZN3yWT+JirtSI62YsNyywvf6//IjTVQq/SLLfybSDerV9LsyHG7VaqAGqLGLfjwGDdGaSB++Tm9qfWne5oh0cS2wscHoCzzt1/3pBd8C1cq9GmWnVo5rAdHnRp/XUvVFortwR0DnIOvVnMJxK1mpnnHwLdqWmyb7msZhizc6T+ipzN2V7oYY01gbndsn0+ZYkBSWz22eEZoMRDUdgiE+ZeMnCRZLSMxIDSK+6cxaE7L+MFJU45KMPcvdD3ZM/WKiZl2knNbdJbuytOESyWgDxfnDMVO9YztH3sHRlIf1a/COfc7sKgQH0vXFf9GU0Uzf24pW9D9OdlJ"
        }
      ]

      boot_diagnostics = {
        storage_account = {
          name = "ubuntustorage123dsa"
          access_tier = "Hot"
        }
      }
    },
    {
      name = "Centos1"
      os_name = "Centos"

       admin_ssh_key = [
        {
          username = "localadmin"
          public_key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDjm7vUE6KhuZN3yWT+JirtSI62YsNyywvf6//IjTVQq/SLLfybSDerV9LsyHG7VaqAGqLGLfjwGDdGaSB++Tm9qfWne5oh0cS2wscHoCzzt1/3pBd8C1cq9GmWnVo5rAdHnRp/XUvVFortwR0DnIOvVnMJxK1mpnnHwLdqWmyb7msZhizc6T+ipzN2V7oYY01gbndsn0+ZYkBSWz22eEZoMRDUdgiE+ZeMnCRZLSMxIDSK+6cxaE7L+MFJU45KMPcvdD3ZM/WKiZl2knNbdJbuytOESyWgDxfnDMVO9YztH3sHRlIf1a/COfc7sKgQH0vXFf9GU0Uzf24pW9D9OdlJ"
        }
      ]
    }
  ]

  kv_object = {

    network_acls = {
      add_vm_subnet_id = true
    }
  }

  create_nsg = true
  create_public_ip = true
  create_diagnostic_settings = true
}

output "connect" {
  value = module.test3_vms.summary_object
}
*/

data "azurerm_client_config" "current" {}

module "my_first_vm" {
  source = "../../azurerm-vm-bundle" //Always use a specific version of the module

  rg_name = "vm-rg2" //Creating a new rg

  //We will only define some simple object configurations here. For more information, see the advanced examples
vnet_object = {
  address_space = ["192.168.0.0/20"]
  name = "custom-vnet"
  tags = {
    "environment" = "prod"
  }
}

//You need define 2 subnets in case 'create_bastion = true' The module will always use index 0 for the vm's
//Name is not required and for the bastion subnet it will always be 'AzureBastionSubnet' Regardless of user defined name
subnet_objects = [
  {
    name = "custom-vm-subnet"
    address_prefixes = ["192.168.0.0/22"]
  },
  {
    address_prefixes = ["192.168.10.0/24"]
  }
]

  vm_linux_objects = [
    {
      name = "ubuntu-vm"
      os_name = "ubuntu"
    }
  ]

  bastion_object = {
    copy_paste_enabled = false
    file_copy_enabled = false
    name = "my-custom-bastion"
    scale_units = 6
    sku = "Standard"
  }

  kv_object = {
    enabled_for_deployment = false
    enabled_for_disk_encryption = true
    enabled_for_template_deployment = true
    
    network_acls = {
      add_vm_subnet_id = true
    }
  }
}