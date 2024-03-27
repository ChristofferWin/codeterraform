variable "location" {
  type = string
  default = "northeurope"
}

variable "rg_id" {
  type = string
  default = null
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
  default = null
}

variable "vm_linux_objects" {
  type = any
  default = null
}