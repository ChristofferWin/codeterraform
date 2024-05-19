provider "azurerm" {
  features {

  }
}

module "vms" {
    source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"
    rg_id = "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/rg-spoke2"
    subnet_resource_id = "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/rg-spoke2/providers/Microsoft.Network/virtualNetworks/vnet-spoke2/subnets/subnet1-spoke2"
    vnet_resource_id = "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/rg-spoke2/providers/Microsoft.Network/virtualNetworks/vnet-spoke2"

    vm_windows_objects = [
        {
            name = "test-connection"
            os_name = "windows11"
            admin_password = ",OpUgCJ{@PKeuKb"
        }
    ]
}

module "vms2" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-vm-bundle?ref=main"
  rg_id = "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/rg-spoke1"
  subnet_resource_id = "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/rg-spoke1/providers/Microsoft.Network/virtualNetworks/vnet-spoke1/subnets/subnet1-spoke1"
  vnet_resource_id = "/subscriptions/25d70457-06ad-442e-a428-fff5a8dd3db3/resourceGroups/rg-spoke1/providers/Microsoft.Network/virtualNetworks/vnet-spoke1"

    vm_windows_objects = [
        {
            name = "test-connection"
            os_name = "windows11"
            admin_password = ",OpUgCJ{@PKeuKb"
        }
    ]
  
}

output "deployment" {
  value = module.vms.summary_object
}