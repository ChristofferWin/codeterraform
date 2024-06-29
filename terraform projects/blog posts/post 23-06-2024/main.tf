locals {
  test_object = {
    name = var.nested_object_with_nested_object_optional
    attribute1 = var.nested_object_with_nested_object_optional.resource_attributes.attribute1
    attribute2 = var.nested_object_with_nested_object_optional.resource_attributes.attribute2
  }
}

output "nested_object_attributes" {
  value = local.test_object
}