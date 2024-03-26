//Its always highly advised to use direct GIT URI´s for the module sources
//To contain consistency between the local and remote repo

//MUST define provider config
//Using inline azure context
provider "azurerm" {
  features {
  }
}

//First module call - running using "create" switches for a simple deployment
module "integration_test_1_using_create" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"

  //Define required params for the module
  rg_name = var.rg_name
  location = var.location
  create_nsg = true
  create_public_ip = true
  create_diagnostic_settings = true
  create_kv_for_vms = true
  create_kv_role_assignment = true

  vm_windows_objects = var.vm_windows_objects
  vm_linux_objects = var.vm_linux_objects
}