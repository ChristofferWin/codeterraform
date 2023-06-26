terraform {
  required_providers {
    azurerm = {
        source = "hashicorp/azurerm"
    }
    azuread = {
        source = "hashicorp/azuread"
    }
    local = {
        source = "hashicorp/local"
    }
    random = {
        source = "hashicorp/random"
    }
  }
}

provider "azurerm" {
  features {
  }
}

locals {
  tenant_id = data.azurerm_subscription.primary.tenant_id
  subscription_id = data.azurerm_subscription.primary.subscription_id
  domain_name = data.azuread_domains.aad_domains.domains[0].domain_name
}

data "azurerm_subscription" "primary" {}

data "azurerm_client_config" "user" {}

data "azuread_domains" "aad_domains" {}

data "azurerm_role_definition" "key_vault_administrator_role_object" {
  name = "Key Vault Administrator"
  scope = azurerm_key_vault.kv_object.id
}

resource "azurerm_resource_group" "rg_object"{
  name = "${var.base_name}-rg"
  location = var.location
}

resource "azurerm_storage_account" "storage_object" {
  name = "${random_string.storage_and_kv_name[0].result}storage01"
  location = azurerm_resource_group.rg_object.location
  account_tier = "Standard"
  account_replication_type = "LRS"
  resource_group_name = azurerm_resource_group.rg_object.name
  account_kind = "BlobStorage"
  access_tier = "Cool"
  enable_https_traffic_only = true
}

resource "random_string" "storage_and_kv_name" {
  count = 2
  length = 8
  special = false
  lower = true
  upper = false
}

resource "azurerm_virtual_network" "vnet_object" {
  name = "${var.base_name}-mgmt-vnet"
  location = azurerm_storage_account.storage_object.location
  resource_group_name = azurerm_resource_group.rg_object.name
  address_space = [var.address_space[0]]
}

resource "azurerm_subnet" "subnet_objects" {
  count = 2
  name = count.index == 0 ? "LAN" : "GatewaySubnet"
  address_prefixes = [var.address_space[count.index == 0 ? 1 : 2]]
  virtual_network_name = azurerm_virtual_network.vnet_object.name
  resource_group_name = azurerm_resource_group.rg_object.name
}

resource "azurerm_network_security_group" "nsg_lan_object" {
  name = "${var.base_name}-LAN-nsg"
  location = azurerm_resource_group.rg_object.location
  resource_group_name = azurerm_resource_group.rg_object.name
  
  security_rule {
    name = "ACCESS_FROM_HOME"
    priority = 100
    direction = "Inbound"
    access = "Allow"
    protocol = "Tcp"
    source_port_range = "*"
    destination_port_range = "3389"
    source_address_prefix = var.address_space[3]
    destination_address_prefix = var.address_space[1]
  }
}

resource "azurerm_virtual_network_gateway" "vpn_object" {
  name = "${var.base_name}-vpn-gw"
  location = azurerm_resource_group.rg_object.location
  resource_group_name = azurerm_resource_group.rg_object.name
  sku = "VpnGw2"
  type = "Vpn"
  active_active = false
  enable_bgp = false
  generation = "Generation2"
  private_ip_address_enabled = true
  vpn_type = "RouteBased"
  
  ip_configuration {
    private_ip_address_allocation = "Dynamic"
    subnet_id = azurerm_subnet.subnet_objects[1].id
    public_ip_address_id = azurerm_public_ip.pip_vpn_object.id
  }

  vpn_client_configuration {
    address_space = [var.address_space[3]]
    aad_audience = "41b23e61-6c1e-4545-b367-cd054e0ed4b4"
    aad_tenant = "https://login.microsoftonline.com/${local.tenant_id}/"
    aad_issuer = "https://sts.windows.net/${local.tenant_id}/"
    vpn_client_protocols = ["OpenVPN"]
    vpn_auth_types = ["AAD", "Certificate"]
  
    root_certificate {
      name = "vpn_client_pub_cert"
      public_cert_data = azurerm_key_vault_certificate.kv_cert_vpn_object.certificate_data_base64
    }
  }
}

resource "azurerm_public_ip" "pip_vpn_object" {
  name = "${var.base_name}-vpn-pip01"
  resource_group_name = azurerm_resource_group.rg_object.name
  location = azurerm_resource_group.rg_object.location
  allocation_method = "Static"
  sku = "Standard"
}

resource "azurerm_key_vault" "kv_object" {
  name = "${random_string.storage_and_kv_name[1].result}kv01"
  location = azurerm_resource_group.rg_object.location
  resource_group_name = azurerm_resource_group.rg_object.name
  sku_name = "standard"
  tenant_id = data.azurerm_subscription.primary.tenant_id
  enable_rbac_authorization = true
  purge_protection_enabled = true
}

resource "azurerm_role_assignment" "kv_administrator_object" {
  scope = azurerm_key_vault.kv_object.id
  role_definition_id = data.azurerm_role_definition.key_vault_administrator_role_object.role_definition_id
  principal_id = data.azurerm_client_config.user.object_id
}

resource "azurerm_key_vault_certificate" "kv_cert_vpn_object" {
  name = "client-vpn-cert"
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

      lifetime_action {
        action {
          action_type = "AutoRenew"
        }

        trigger {
          days_before_expiry = 30
        }
      }

      secret_properties {
        content_type = "application/x-pkcs12"
      }

      x509_certificate_properties {
      key_usage = [
          "cRLSign",
          "dataEncipherment",
          "digitalSignature",
          "keyAgreement",
          "keyCertSign",
          "keyEncipherment",
        ]
      subject = "CN=${local.domain_name}"
      validity_in_months = 12
    }
  }
}