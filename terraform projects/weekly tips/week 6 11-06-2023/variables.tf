//Lets first build a standard and simple string variable:
/*
variable "resource_base_name" {
  description = "the prefix to be used on all new resource names"
  type = string

  validation {
    condition = length(regexall("^(PRD|TST|DEV)-.*$", var.resource_base_name)) > 0
    error_message = "the value provided '${var.resource_base_name}' must start with eiter PRD-, TST- or DEV-"
  }
}
*/