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

locals {
  rg_name = azurerm_resource_group.rg_object.name
  rg_id = azurerm_resource_group.rg_object.id
}

resource "azurerm_resource_group" "rg_object" {
  name = "test-rg"
  location = "westeurope"
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
module "test_vms" {
    source = "../../azurerm-vm-bundle"
    rg_id = local.rg_id
    vm_windows_objects = [
        {
          name = "Windows10Machin"
          os_name = "WINDOWS10"

          admin_username = "mofo"
          admin_password = "asdasd123123123540øø^*!"
          allow_extension_operations = true

          boot_diagnostics = {
            storage_account = {
              name = "teststorage1256775"
              account_replication_type = "GRS"
              account_tier = "Premium"
            }
          }
        },
        {
          name = "WINSERVER"
          os_name = "SERVEr2012"
        }
    ]
    create_nsg = true
    create_diagnostic_settings = true
    create_public_ip = true
}
