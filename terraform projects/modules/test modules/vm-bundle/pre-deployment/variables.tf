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

variable "subnet_bastion_id" {
  type = string
  default = null
}

variable "kv_id" {
  type = string
  default = null
}

variable "subnet_objects" {
  type = any
  default = null 
}