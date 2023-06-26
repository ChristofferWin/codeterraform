variable "base_name" {
  description = "the base name to be used as a prefix on any given resource"
  type = string
  default = "vs-code"
}

variable "location" {
  description = "the azure location of which to create resources"
  type = string
  default = "westeurope"
}

variable "address_space" {
  description = "the ip range of which this environment shall run"
  type = list(string)
  default = [ "192.168.0.0/24", "192.168.0.0/26", "192.168.0.64/27", "172.16.0.0/24"]
}