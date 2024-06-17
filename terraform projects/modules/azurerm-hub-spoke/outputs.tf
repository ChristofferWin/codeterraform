
output "rg_return_objects" {
  value = local.rg_return_objects
}

output "vnet_return_objects" {
  value = local.vnet_return_objects
}

output "subnet_return_objects" {
  value = local.subnet_return_objects
}

output "rt_return_objects" {
  value = local.rt_return_objects
}

output "fw_return_object" {
  value = local.fw_return_object
}

output "gw_return_object" {
  value = local.gw_return_object
}

output "pip_return_objects" {
  value = local.pip_return_object
}

output "log_return_object" {
  value = can(local.log_return_helper_object[0]) ? {
    id = local.log_return_helper_object[0].id
    workspace_id = local.log_return_helper_object[0].workspace_id
  } : {}
}