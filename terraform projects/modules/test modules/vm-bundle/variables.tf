variable "rg_name" {
  type = string
  default = null
}

variable "location" {
  type = string
  default = null
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

variable "subnet_bastion_resource_id" {
  type = string
  default = null
}

variable "vnet_object" {
  type = any
  default = null
}

variable "subnet_objects" {
  type = any
  default = null
}

variable "subnet_objects_with_bastion" {
  type = any
  default = null
}

variable "vm_windows_objects_simple" {
  type = any
  default = null
}

variable "vm_linux_objects_simple" {
  type = any
  default = null
}

variable "vm_windows_objects_custom_config" {
  type = any
  default = null
}

variable "vm_linux_objects_custom_config" {
  type = any
  default = null
}

variable "vm_windows_objects_mix" {
  type = any
  default = null
}

variable "vm_linux_objects_mix" {
  type = any
  default = null
}