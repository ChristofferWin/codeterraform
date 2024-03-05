locals {
  test = "./Get-Azasdsda.ps1"
  test2 = "${regexall(\"^(.*\\/)?([^\\/]+)\\.ps1$\",var.script_name)[0]}"
}

output "test" {
  value = local.test
}