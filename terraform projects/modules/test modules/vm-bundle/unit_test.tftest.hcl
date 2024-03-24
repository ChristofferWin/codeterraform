run "pre_deployment_for_apply" {
  command = apply

    plan_options {
      target = [module.pre_deployment_vnet_subnet]
    }

    module {
      source = "./pre-deployment"
    }

    variables {
      rg_name = "vm-bundle-integration-test-rg"
      location = var.location
      vnet_object = var.vnet_object
      subnet_objects = var.subnet_objects 
    }
}

run "pre_deployment_for_apply2" {
  command = apply

  plan_options {
    target = [module.pre_deployment_mgmt_resources]
  }

  variables {
    rg_name = "vm-bundle-mgmt-test-rg" 
    location = var.location
    vnet_object = var.vnet_object
    subnet_objects = var.subnet_objects_with_bastion
  }
}

 
run "integration_test_1_check_vm_count_apply" { 
  command = apply

  plan_options {
    target = [module.unit_test_1_using_existing_resources]
  }

  variables {
    rg_id = run.pre_deployment_for_apply.rg_id
    vnet_resource_id = run.pre_deployment_for_apply.vnet_resource_id
    subnet_resource_id = run.pre_deployment_for_apply.subnet_resource_id
  }
}

run "integration_test_2_check_using_existing_resources" {
  command = apply

  plan_options {
    target = [module.unit_and_integration_test_2]
  }

  variables {
    rg_name = "vm-bundle-integration-test2-rg"
    location = var.location
    subnet_bastion_resource_id = run.pre_deployment_for_apply2.subnet_bastion_resource_id
  }
}