variable "client_secret" {
  description = "the password of the service user in clear text"
  type = string
  sensitive = true
  default = null
}