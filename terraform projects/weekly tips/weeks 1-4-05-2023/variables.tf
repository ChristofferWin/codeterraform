variable "client_secret" {
  description = "the password of the service user in clear text"
  type = string
  sensitive = true
  default = null
}

variable "subscription_id" {
  description = "the id of the subscription of which to interract with azure"
  type = string
  default = ""
}

variable "tenant_id" {
  description = "the id of the tenant of which to interract with azure"
  type = string
  default = ""
}

variable "client_id" {
  description = "the app id of the spn"
  type = string
  default = ""
}