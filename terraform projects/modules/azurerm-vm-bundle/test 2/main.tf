terraform {
  required_providers {
    null = {
      source = "hashicorp/null"
    }
  }
}

resource "null_resource" "ps_object" {
  provisioner "local-exec" {
    command     = "./test.ps1"
    interpreter = ["pwsh", "-Command"]
  }
}