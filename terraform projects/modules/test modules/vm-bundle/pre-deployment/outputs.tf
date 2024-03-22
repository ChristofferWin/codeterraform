output "rg_id" {
  value = module.pre_deployment.rg_object.id
}

output "vnet_resource_id" {
  value = values(module.pre_deployment.vnet_object)[0].id
}

output "subnet_resource_id" {
  value = [for each in values(module.pre_deployment.subnet_object).*.id : each if length(regexall("bastion", lower(each))) == 0][0]
}

output "subnet_bastion_resource_id" {
  value = [for each in values(module.pre_deployment.subnet_object).*.id : each if length(regexall("bastion", lower(each))) > 0][0]
}