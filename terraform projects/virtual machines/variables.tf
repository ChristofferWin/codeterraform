variable "VM_Objects" {
  description = "a list of VM specifications to deploy"
  type = list(object({
    OS = string
    VM_Pattern = optional(string)
    File_name = optional(string)
    Note = optional(string)
  }))
  default = [ 
    {
        OS = "CentOS"
        VM_Pattern = "DC4s"
        File_name = "Centos6-Newest-SKUs.json"
        Note = "information required to deploy Centos 6 servers with then newest SKU"
    }
  ]
}

variable "env_name" {
  description = "name of the environment, must be either 'prod' or 'test'"
  type = string

  validation {
    condition = length(regexall("^test|prod", var.env_name)) > 0
    error_message = "the env_name provided '${var.env_name}' did not match pattern 'test|prod'"
  }
  default = "prod"
}

variable "Script_path" {
  description = "the litteral path to the powershell script file"
  type = string
  default = ".\\Get-AzVMSku"
}