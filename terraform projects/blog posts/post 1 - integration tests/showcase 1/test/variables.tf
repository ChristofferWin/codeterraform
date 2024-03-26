variable "rg_name" {
  type = string
  default = "integration-test-rg"
}

variable "rg_id" {
  type = string
  default = null
}

variable "location" {
  type = string
  default = "northeurope"
}

variable "vnet_resource_id" {
  type = string
  default = null
}

variable "subnet_resource_id" {
  type = string
  default = null
}

variable "kv_resource_id" {
  type = string
  default = null
}

variable "vm_windows_objects" {
  type = any
  default = [
    {
      name = "test-win-01"
      os_name = "windows11"
    }
  ]
}

variable "vm_linux_objects" {
  type = any
  default = [
    {
      name = "test-linux-01"
      os_name = "debian11"
    }
  ]
}