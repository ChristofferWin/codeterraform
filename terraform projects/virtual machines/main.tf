terraform {
  required_providers {
    azurerm = {
        source = "hashicorp/azurerm"
    }
    null = {
        source = "hashicorp/null"
    }
    random = {
        source = "hashicorp/random"
    }
    local = {
        source = "hashicorp/local"
    }
  }
}

provider "azurerm" {
  features {
  }
}

resource "null_resource" "powershell_script_object" {
    triggers = {
        build_number = "7"
    }
    provisioner "local-exec" {
        command = ".\\Measure-command.ps1 -Location westeurope -OperatingSystem windows11"
        interpreter = ["pwsh.exe", "-Command"]
    }
}

locals {
  content = jsondecode(data.local_file.test.content)
}

data "local_file" "test" {
  filename = "SKUs.json"

  depends_on = [ null_resource.powershell_script_object ]
}

resource "azurerm_resource_group" "rg_object" {
  name = "prod-rg"
  location = "westeurope"
}

resource "azurerm_virtual_network" "vnet_object" {
  name = "prod-vnet"
  resource_group_name = azurerm_resource_group.rg_object.name
  location = azurerm_resource_group.rg_object.location
  address_space = ["10.0.0.0/24"]
}

resource "azurerm_subnet" "subnet_object" {
  name = "vm-subnet"
  resource_group_name = azurerm_resource_group.rg_object.name
  virtual_network_name = azurerm_virtual_network.vnet_object.name
  address_prefixes = ["10.0.0.0/26"]
}

resource "azurerm_network_interface" "nic_object" {
  name = "prod-vm01-nic01"
  location = azurerm_resource_group.rg_object.location
  resource_group_name = azurerm_resource_group.rg_object.name

  ip_configuration {
    name = "ipconfig"
    subnet_id = azurerm_subnet.subnet_object.id
    private_ip_address_allocation = "Dynamic"
  }
}

resource "azurerm_virtual_machine" "vm_object" {
  name = "prod-vm01"
  location = azurerm_resource_group.rg_object.location
  resource_group_name = azurerm_resource_group.rg_object.name
  network_interface_ids = [azurerm_network_interface.nic_object.id]
  vm_size = local.content.VMSizes[1].Name

  storage_image_reference {
    publisher = local.content.Publisher
    offer     = local.content.Offer
    sku       = local.content.SKUs
    version   = local.content.Versions[0].Versions[0]
  }
  storage_os_disk {
    name              = "myosdisk1"
    caching           = "ReadWrite"
    create_option     = "FromImage"
    managed_disk_type = "Standard_LRS"
  }
  os_profile {
    computer_name  = "hostname"
    admin_username = "testadmin"
    admin_password = "Password1234!"
  }
  os_profile_linux_config {
    disable_password_authentication = false
  }
}

output "test" {
  value = local.content.Versions[0].SKU
}