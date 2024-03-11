variable "rg_id" {
  type = string
  default = "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/test-rg"
}

variable "vnet_resource_id" {
  type = string
  default = "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourcegroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-vnet"
}

variable "subnet_resource_id" {
  type = string
  default = "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/vm-tester"
}

variable "subnet_bastion_resource_id" {
  type = string
  default = "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-vnet/subnets/AzureBastion"
}

variable "vm_windows_objects" {
  type = any
  default = [
    {
        name = "test-win-vm01"
        os_name = "windows10"
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
        vm_size = "D1"
    },
    {
        name = "test-linux-vm02"
        os_name = "DeBiaN11"
        vm_size = "D1"
    }
]
}

variable "script_name" {
  type = string
  default = "./Get-AzVMSKu.ps1"
}

variable "location" {
  type = string
  default = "westeurope"
}

variable "subnet_object" {
  type = any
  default = [
    {
      name = "tester-subnet"
      address_prefixes = ["10.0.3.0/24"]
    },
    {
      name = "asdasdasd"
      address_prefixes = ["10.0.4.0/24"]
    }
  ]
}