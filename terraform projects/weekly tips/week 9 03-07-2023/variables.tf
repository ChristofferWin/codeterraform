variable "name_prefix" {
  description = "a default prefix name to use on any resource type"
  type = string
  default = "company"
}

variable "location" {
  description = "the location of any resource type"
  type = string
  default = "westeurope"
}

variable "environment_type" {
  description = "must be exactly one of the following strings: 'dev', 'test', 'prod'"
  type = string

  validation {
    condition = length(regexall("^(dev|test|prod)$", var.environment_type)) > 0
    error_message = "the environment_type provided '${var.environment_type}' did not match any of 'dev||test||prod'"
  }
}

variable "ip_address_space" {
  description = "a list of string defining the address space(s) of the environment"
  type = list(string)
  default = [ "10.0.0.0/24" ]
}