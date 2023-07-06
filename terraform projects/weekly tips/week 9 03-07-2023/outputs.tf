output "pip_gw_ip" {
  value = azurerm_public_ip.pip_object.*.ip_address
}