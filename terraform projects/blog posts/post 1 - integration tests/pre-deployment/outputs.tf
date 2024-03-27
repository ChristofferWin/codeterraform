  output "rg_id" {
    value = module.integration_test_1_using_create.rg_object.id
  }

  output "vnet_resource_id" {
    value = values(module.integration_test_1_using_create.vnet_object)[0].id
  }

  output "subnet_resource_id" {
    value = values(module.integration_test_1_using_create.subnet_object)[1].id
  }

  output "kv_resource_id" {
    value = values(module.integration_test_1_using_create.kv_object)[0].id
  }