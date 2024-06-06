run "test_1_simple_deployment_apply" {
  command = apply
   
   plan_options {
     target = [module.deployment_1_simple_hub_spoke]
   }

   assert {
     condition = length(([for a, b in values(module.deployment_1_simple_hub_spoke.vnet_return_objects) : true if length(split("-", b.name)) == 2])) == length(module.deployment_1_simple_hub_spoke.vnet_return_objects)
     error_message = "the names of which failed the validation: ${jsonencode([for a, b in values(module.deployment_1_simple_hub_spoke.vnet_return_objects) : "${b.name}," if length(split("-", b.name)) != 2])}"
   }

   assert {
     condition = length([for a, b in var.deployment_1_simple_hub_spoke.hub_object.network.address_spaces : b if contains([for c, d in values(module.deployment_1_simple_hub_spoke.vnet_return_objects) : d.address_space if strcontains(d.name, "hub")][0], b)]) == 2
     error_message = "the address_space in Azure: ${jsonencode([for a, b in values(module.deployment_1_simple_hub_spoke.vnet_return_objects) : b.address_space if strcontains(b.name, "hub")])} did not match the address_spaces assigned to it: ${jsonencode([for a, b in var.deployment_1_simple_hub_spoke.hub_object.network.address_spaces : b])}"
   }
}

