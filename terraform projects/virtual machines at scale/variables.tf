variable "environment_objects" {
  description = "a map of objects representing environment configurations"
  type = map(object({
    name = string
    location = optional(string)
    address_space = list(string)
    number_of_windows_vms = number
    number_of_linux_vms = number
    size_pattern = optional(string)
  }))

  default = {
    "env_prod" = {
      name = "prod"
      location = "northeurope"
      address_space = ["10.0.0.0/16"]
      number_of_windows_vms = 3
      number_of_linux_vms = 2
      size_pattern = "B4Ms"
    },
    "env_dev" = {
      name = "dev"
      address_space = ["192.168.0.0/24"]
      number_of_windows_vms = 2
      number_of_linux_vms = 1
    }
  }
}