terraform {
  required_providers {
    azurerm = {
        source = "hashicorp/azurerm" 
    }
    local = {
        source = "hashicorp/local" 
    }
    random = {
        source = "hashicorp/random" 
    }
    azuread = {
        source = "hashicorp/azuread"
    }
    null = {
        source = "hashicorp/null"
    }
  }
}

provider "azurerm" {
  features {
  }
  client_id = var.client_id
  client_secret = var.client_secret
  subscription_id = var.subscription_id
  tenant_id = var.tenant_id
}

locals {
  location = azurerm_resource_group.demo_rg_object.location
  base_resource_name = split("-", azurerm_resource_group.demo_rg_object.name)[0]
  rg_name = azurerm_resource_group.demo_rg_object.name
  vm_admin_username = "${join("-", azurerm_virtual_machine.demo_vm_object.os_profile.*.admin_username)}"
  vm_admin_password = "${random_password.vm_demo_password_admin_object.result}"
}

data "azurerm_client_config" "current" {}

resource "azurerm_resource_group" "demo_rg_object" {
  name = "demo-rg"
  location = "west europe"
}

resource "random_string" "storage_random_string_object" {
    length = 3
    min_numeric = 3
}

resource "azurerm_storage_account" "demo_storage_object" {
  name = "${local.base_resource_name}${random_string.storage_random_string_object.result}storage"
  location = local.location
  account_tier = "Standard"
  account_replication_type = "LRS"
  resource_group_name = local.rg_name
}

resource "azurerm_virtual_network" "demo_vn_object" {
  name = "${local.base_resource_name}-vn"
  location = local.location
  resource_group_name = local.rg_name
  address_space = ["192.168.0.0/24"]
}

resource "azurerm_subnet" "demo_client_subnet_object" {
  name = "${local.base_resource_name}-client-subnet"
  resource_group_name = local.rg_name
  virtual_network_name = azurerm_virtual_network.demo_vn_object.name
  address_prefixes = ["192.168.0.64/26"]
}

resource "azurerm_network_security_group" "demo_client_nsg_object" {
  name = "${local.base_resource_name}-client-nsg"
  location = local.location
  resource_group_name = local.rg_name

  security_rule {
    name = "ALLOW-RDP-PUBLIC"
    priority = 100
    direction = "Inbound"
    access = "Allow"
    protocol = "Tcp"
    source_port_range = "*"
    destination_port_range = "3389" //RDP
    source_address_prefix = "85.83.136.22/32" //Define your own public IP
    destination_address_prefix = "*"
  }
}

resource "azurerm_subnet_network_security_group_association" "demo_nsg_link_oject" {
  subnet_id = azurerm_subnet.demo_client_subnet_object.id
  network_security_group_id = azurerm_network_security_group.demo_client_nsg_object.id
}

resource "azurerm_public_ip" "demo_client_pip_object" {
  name = "${local.base_resource_name}-client-pip"
  resource_group_name = local.rg_name
  location = local.location
  sku = "Basic"
  allocation_method = "Static"
}

resource "azurerm_network_interface" "demo_client_nic_object" {
  name = "${local.base_resource_name}-client-nic01"
  location = local.location
  resource_group_name = local.rg_name
  
  ip_configuration {
    name = "client_lan_ip_configuration"
    subnet_id = azurerm_subnet.demo_client_subnet_object.id
    private_ip_address_allocation = "Static"
    private_ip_address = "192.168.0.68" //First available address in subnet
    public_ip_address_id = azurerm_public_ip.demo_client_pip_object.id
  }
}

resource "azurerm_virtual_machine" "demo_vm_object" {
  name = "${local.base_resource_name}-vm01"
  location = local.location
  resource_group_name = local.rg_name
  network_interface_ids = [azurerm_network_interface.demo_client_nic_object.id]
  vm_size = "Standard_B2ms"
  delete_os_disk_on_termination = true

  storage_image_reference {
    publisher = "MicrosoftWindowsDesktop"
    sku = "win11-22h2-pro"
    version = "22621.1702.230505"
    offer = "windows-11"
  }

  storage_os_disk {
    name = "${local.base_resource_name}-os-disk"
    create_option = "FromImage"
    caching = "ReadWrite"
    disk_size_gb = 128
    os_type = "Windows"

  }

  os_profile_windows_config {
    provision_vm_agent = true
    enable_automatic_upgrades = true
  }

  os_profile {
    computer_name = "${local.base_resource_name}-vm01"
    admin_username = "${local.base_resource_name}admin"
    admin_password = random_password.vm_demo_password_admin_object.result
  }
}

resource "random_password" "vm_demo_password_admin_object" {
  length           = 16
  special          = true
  override_special = "!*()-_=+[]<>:"
}

resource "azurerm_key_vault" "demo_kv_object" {
  name = "${local.base_resource_name}-${random_string.kv_random_string_object.result}-kv"
  location = local.location
  resource_group_name = local.rg_name
  sku_name = "standard"
  tenant_id = var.tenant_id
  purge_protection_enabled = true
  public_network_access_enabled = true

  access_policy {
    object_id = data.azurerm_client_config.current.object_id
    tenant_id = var.tenant_id

    secret_permissions = [
      "Backup",
      "Delete",
      "Get",
      "List",
      "Purge",
      "Recover",
      "Restore",
      "Set"
    ]
  }
}

resource "random_string" "kv_random_string_object" {
  length = 3
  min_numeric = 3
}

resource "azurerm_key_vault_secret" "demo_vm_admin_secret_object" {
  name = "${local.base_resource_name}vmadmin"
  value = "username: ${local.vm_admin_username} password: ${local.vm_admin_password}"
  key_vault_id = azurerm_key_vault.demo_kv_object.id
}

resource "null_resource" "invoke_ps_object" {
  triggers = {
      build_number = "${timestamp()}"
  }
  provisioner "local-exec" {
      command = "${path.module}/Start-RDPSession.ps1 -IPAddress ${azurerm_public_ip.demo_client_pip_object.ip_address} -Username ${local.vm_admin_username} -Password ${local.vm_admin_password}"
      interpreter = ["powershell.exe","-Command"]
  }
}