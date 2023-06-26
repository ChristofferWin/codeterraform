locals {
  ip_configuration = {
        "loadbalancer-prod" = {
            host_address = ["10.0.0.100", "10.0.0.101"]
            subnet_prefix = "/24"
        },
        "loadbalancer-test" = {
            host_address = ["10.0.1.100", "10.0.0.101"]
            subnet_prefix = "/24"
        }
    }
  list_of_ips = flatten(values(local.ip_configuration).*.host_address)
}

output "list_of_ips" {
  value = local.list_of_ips[1]
}