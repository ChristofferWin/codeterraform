variable "rg_id" {
  type = string
  default = "/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/test-rg"
}

variable "vnet_resource_id" {
  type = string
  default = "/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourcegroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-vnet-1337"
}

variable "subnet_resource_id" {
  type = string
  default = "/subscriptions/d519214d-1363-451a-a24a-234b92d5642b/resourceGroups/test-rg/providers/Microsoft.Network/virtualNetworks/test-vnet-1337/subnets/default"
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
    },
    {
        name = "test-linux-vm02"
        os_name = "DeBiaN11"
    }
]
}