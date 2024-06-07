run "test_1_simple_deployment_apply" {
  command = apply

  plan_options {
    target = [module.deployment_1_simple_hub_spoke]
  } 

   assert {
     condition = length(([for a, b in values(module.deployment_1_simple_hub_spoke.vnet_return_objects) : true if length(split("-", b.name)) == 3])) == length(module.deployment_1_simple_hub_spoke.vnet_return_objects)
     error_message = "the names of which failed the validation: ${jsonencode([for a, b in values(module.deployment_1_simple_hub_spoke.vnet_return_objects) : b.name if length(split("-", b.name)) != 2])}"
   }

   assert {
     condition = length([for a, b in var.deployment_1_simple_hub_spoke.hub_object.network.address_spaces : b if contains([for c, d in values(module.deployment_1_simple_hub_spoke.vnet_return_objects) : d.address_space if strcontains(d.name, "hub")][0], b)]) == 2
     error_message = "the address_space (FOR HUB) in Azure: ${jsonencode([for a, b in values(module.deployment_1_simple_hub_spoke.vnet_return_objects) : b.address_space if strcontains(b.name, "hub")])} did not match the address_spaces assigned to it: ${jsonencode([for a, b in var.deployment_1_simple_hub_spoke.hub_object.network.address_spaces : b])}"
   }
}

run "test_2_simple_deployment_with_vpn_apply" {
  command = apply

  plan_options {
    target = [module.deployment_2_simple_with_vpn]
  }
}
/*
run "test_3_simple_deployment_with_firewall_apply" {
  command = apply

  plan_options {
    target = [module.deployment_3_simple_with_firewall]
  }
}
*/
run "test_4_advanced_deployment_with_all_custom_values" {
  command = apply

  plan_options {
    target = [module.deployment_4_advanced_with_all_custom_values]
  }
}