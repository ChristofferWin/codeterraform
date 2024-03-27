//Its always highly advised to use direct GIT URI´s for the module sources
//To contain consistency between the local and remote repo

//MUST define provider config
//Using inline azure context
provider "azurerm" {
  features {
  }
}

//Second module call - Notice that we use more input variables instead of direct module return calls to retrieve the first modules resource ids
module "integration_test_2_using_ids" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"

  //Define required params for the module
  rg_id = var.rg_id
  vnet_resource_id = var.vnet_resource_id
  subnet_resource_id = var.subnet_resource_id
  kv_resource_id = var.kv_resource_id
  location = var.location
  vm_windows_objects = var.vm_windows_objects
  vm_linux_objects = var.vm_linux_objects
}