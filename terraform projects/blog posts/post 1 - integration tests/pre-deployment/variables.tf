variable "location" {
  type = string
  default = "northeurope"
}

variable "rg_name" {
  type = string
  default = "integration-test-rg"
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