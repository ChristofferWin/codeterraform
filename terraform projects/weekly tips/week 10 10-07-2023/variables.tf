variable "environment_name" {
  description = "the name of the environment, must be 'prod' 'preprod' or 'dev'"
  type = string

  validation {    
    condition = length(regexall("^(dev|test|prod)$", var.environment_type)) > 0
    error_message = "the environment_type provided '${var.environment_type}' did not match any of 'dev||test||prod'"
  }
}

variable "location" {
  description = "the location for the azure resouces"
  type = string
}

variable "resource_base_name" {
  description = "the base resource name which will be used as a prefix"
  type = string
}

variable "ip_address_space" {
  description = "the ip address spaces of the virtual networks"
  type = list(string)
}