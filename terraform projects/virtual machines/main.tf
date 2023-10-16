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

resource "null_resource" "invoking_pwsh" {
  for_each = {for each in var.VM_Objects : each.OS => each}
    triggers = {
        build_number = "${timestamp()}"
    }
    provisioner "local-exec" {
        command = "${var.Script_path} -OS ${each.key} -VMPattern ${each.value.VM_Pattern} -OutputFileName ${each.value.File_name}"
        interpreter = ["pwsh.exe", "-Command"] //Use Powershell core to enforce UTF8 encoding of the output file
    }
}

locals {
  content = jsondecode("[${join(",", (values(data.local_file.SKU_objects)).*.content)}]")
}

data "local_file" "SKU_objects" {
  for_each = {for each in var.VM_Objects : each.OS => each}
  filename = each.value.File_name
  depends_on = [ null_resource.invoking_pwsh ]
}


resource "azurerm_resource_group" "rg_object" {
  name = "${var.env_name}-rg"
  location = "westeurope"
}

resource "azurerm_virtual_network" "vnet_object" {
  name = "${var.env_name}-vnet"
  resource_group_name = azurerm_resource_group.rg_object.name
  location = azurerm_resource_group.rg_object.location
  address_space = ["10.0.0.0/24"]
}

resource "azurerm_subnet" "subnet_object_vm" {
  name = "vm-subnet"
  resource_group_name = azurerm_resource_group.rg_object.name
  virtual_network_name = azurerm_virtual_network.vnet_object.name
  address_prefixes = ["10.0.0.0/26"]
}

resource "azurerm_network_interface" "nic_object" {
  count = length(var.VM_Objects)
  name = "${var.env_name}-${local.content[count.index].Offer}-vm-nic"
  location = azurerm_resource_group.rg_object.location
  resource_group_name = azurerm_resource_group.rg_object.name

  ip_configuration {
    name = "ipconfig"
    subnet_id = azurerm_subnet.subnet_object_vm.id
    private_ip_address_allocation = "Dynamic"
  }

  lifecycle {
    ignore_changes = [ name ]
  }
}

resource "random_password" "pw_object" {
  count = length(var.VM_Objects)
  length           = 16
  special          = true
  override_special = "!#$%&*()-_=+[]{}<>:?"
}

resource "azurerm_virtual_machine" "vm_object" {
  count = length(var.VM_Objects)
  name = "${var.env_name}-${local.content[count.index].Offer}-vm"
  location = azurerm_resource_group.rg_object.location
  resource_group_name = azurerm_resource_group.rg_object.name
  network_interface_ids = [azurerm_network_interface.nic_object[count.index].id]
  vm_size = local.content[count.index].Offer == "CentOS" && var.env_name == "prod" ? local.content[count.index].VMSizes[1].Name : local.content[count.index].VMSizes[0].Name
  delete_os_disk_on_termination = true

  storage_image_reference {
    publisher = local.content[count.index].Publisher
    offer     = local.content[count.index].Offer
    sku       = local.content[count.index].SKUs
    version   = local.content[count.index].Versions[0].Versions
  }

  storage_os_disk {
    name              = "${var.env_name}-${local.content[count.index].Offer}-os"
    caching           = "ReadWrite"
    create_option     = "FromImage"
    managed_disk_type = "Standard_LRS"
    os_type = local.content[count.index].Offer == "CentOS" ? "Linux" : "Windows"
  }

  os_profile {
    computer_name  = "hostname"
    admin_username = "testadmin"
    admin_password = random_password.pw_object[count.index].result
  }
  
  dynamic "os_profile_windows_config" {
    for_each = local.content[count.index].Offer != "CentOS" ? {OS = local.content[count.index].Offer} : {}
    content {
      provision_vm_agent = true
      enable_automatic_upgrades = true
    }
  }

  dynamic "os_profile_linux_config" {
    for_each = local.content[count.index].Offer == "CentOS" ? {OS = local.content[count.index].Offer } : {}
    content {
      disable_password_authentication = false
    }
  }

  depends_on = [ data.local_file.SKU_objects ]

  lifecycle {
    ignore_changes = [ name, storage_image_reference ]
  }
}