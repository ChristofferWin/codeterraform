output "pip_gw" {
  value = [for ip in azurerm_public_ip.pip_object.ip_address.*.ip_address : ip]
}