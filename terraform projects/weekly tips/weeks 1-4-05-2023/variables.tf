variable "client_secret" {
  description = "the password of the service user in clear text"
  type = string
  sensitive = true
  default = null
}

variable "subscription_id" {
  description = "the id of the subscription of which to interract with azure"
  type = string
  default = "07ee07b5-49a4-4519-872a-93e899d90029"
}

variable "tenant_id" {
  description = "the id of the tenant of which to interract with azure"
  type = string
  default = "daa9a3d9-7507-44b4-a06a-805e8e0fee0b"
}

variable "client_id" {
  description = "the app id of the spn"
  type = string
  default = "4c4fe759-1ebf-43d3-a617-4df516258feb"
}