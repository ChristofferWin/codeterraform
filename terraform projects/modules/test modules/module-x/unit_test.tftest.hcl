run "unit_test_1_check_rg_id" {
  command = plan

  assert {
    condition = length([for each in flatten(values(module.unit_test_1_using_existing_resources.nic_object).*.ip_configuration) : true if length(regexall(var.rg_id,"${each.subnet_id}")) == 1]) > 0
    error_message = "The resource group used for deployment via inputting a resource id does not match the resource group used for deployment. Resource group used is: ${split("/", values(module.unit_test_1_using_existing_resources.nic_object)[0].id)[4]}"
  }
}

run "unit_test_2_check_vnet_and_sub_ids" {
  command = plan

  assert {
    condition = length([for each in flatten(values(module.unit_test_1_using_existing_resources.nic_object).*.ip_configuration) : true if length(regexall(var.vnet_resource_id, replace(each.subnet_id, "resourceGroups", "resourcegroups"))) == 1  && each.subnet_id == var.subnet_resource_id]) > 0
    error_message = "Either the virtual network used for the deployment, which is: ${split("/subnets/", values(module.unit_test_1_using_existing_resources.nic_object)[0].ip_configuration[0].subnet_id)[0]} not match the vnet resouce id or the subnet used which is: ${values(module.unit_test_1_using_existing_resources.nic_object)[0].ip_configuration[0].subnet_id} not match"
  }
}

run "unit_test_3_check_vm_count" {
  command = plan

  assert {
    condition = length(flatten([module.unit_test_1_using_existing_resources.summary_object.linux_objects, module.unit_test_1_using_existing_resources.summary_object.windows_objects])) == length(flatten([var.vm_linux_objects, var.vm_windows_objects]))
    error_message = "The amount of VMs defined in variables: ${length(flatten([var.vm_linux_objects, var.vm_windows_objects]))} does not match the amount planned: ${length(flatten([module.unit_test_1_using_existing_resources.summary_object.linux_objects, module.unit_test_1_using_existing_resources.summary_object.windows_objects]))}"
  }
}
/*
run "unit_test_4_check_vm_count_apply" {
  //Default command is apply
  assert {
    condition = length(flatten([module.unit_test_1_using_existing_resources.summary_object.linux_objects, module.unit_test_1_using_existing_resources.summary_object.windows_objects])) == length(flatten([var.vm_linux_objects, var.vm_windows_objects]))
    error_message = "The amount of VMs defined in variables: ${length(flatten([var.vm_linux_objects, var.vm_windows_objects]))} does not match the amount planned: ${length(flatten([module.unit_test_1_using_existing_resources.summary_object.linux_objects, module.unit_test_1_using_existing_resources.summary_object.windows_objects]))}"
  }
}
*/