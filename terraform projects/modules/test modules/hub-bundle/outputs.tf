output "vnets_test_1" {
  value = values(module.deployment_1_simple_hub_spoke.vnet_return_objects)
}

output "vnets_test_2" {
  value = values(module.deployment_2_simple_with_vpn.vnet_return_objects)
}

output "vnets_test_3" {
  value = values(module.deployment_3_simple_with_firewall.vnet_return_objects)
}

output "vnets_test_4" {
  value = values(module.deployment_4_advanced_with_all_custom_values.vnet_return_objects)
}

