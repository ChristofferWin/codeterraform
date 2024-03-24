variable "location" {
  type = string
}

variable "rg_name" {
  type = string
  default = null
}

variable "rg_id" {
  type = string
  default = null
}

variable "vnet_id" {
  type = string
  default = null
}

variable "subnet_id" {
  type = string
  default = null
}

variable "subnet_bastion_resource_id" {
  type = string
  default = null
}

variable "kv_id" {
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