terraform {
  required_providers {
    null = {
      source = "hashicorp/null"
    }
    local = {
      source = "hashicorp/local"
    }
  }
}

resource "null_resource" "ps_object" {
  provisioner "local-exec" {
    command     = "./test.ps1"
    interpreter = ["pwsh", "-Command"]
  }
}

data "local_file" "test" {
  filename = "./test.json"

  depends_on = [ null_resource.ps_object ]
}