variable "nested_objects_with_optional" {
  description = "some description"
  type = object({
    name = string

    nested_objects = optional(list(object({
      attribute1 = string
      attribute2 = optional(string)
    })))
  })
  default = {
    name = "test"
    nested_objects = [ {
      attribute1 = "attribute1"
    } ]
  }
}

variable "nested_object_with_nested_object_optional" {
  description = "some description"
  type = object({
    name = string

    resource_attributes = optional(object({
      attribute1 = optional(string)
      attribute2 = number
    }))
  })
  default = {
    name = "test"

    resource_attributes = {
      attribute2 = 5
    }
  }
}