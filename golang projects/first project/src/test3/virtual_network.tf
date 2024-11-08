resource "azurerm_virtual_network" "test-vnet2323223" {

  name = "test-vnet2323223"

  resource_group_name = "test"

  location = "eastus"

  address_space = ["10.0.0.0/16"]


  subnet {

    name = "default"

    private_endpoint_network_policies = "Disabled"

    private_link_service_network_policies_enabled = true

    address_prefixes = ["10.0.0.0/24"]


  }


  subnet {

    name = "default2"

    private_endpoint_network_policies = "Disabled"

    private_link_service_network_policies_enabled = true

    address_prefixes = ["10.0.1.0/24"]


    delegation {

      name = "Microsoft.BareMetal/AzureVMware"


      service_delegation {

        name = "Microsoft.BareMetal/AzureVMware"

        actions = ["Microsoft.Network/networkinterfaces/*", "Microsoft.Network/virtualNetworks/subnets/join/action", "Microsoft.Network/networkinterfaces/*", "Microsoft.Network/virtualNetworks/subnets/join/action"]


      }
    }
  }


  subnet {

    name = "default3"

    default_outbound_access_enabled = false

    address_prefixes = ["10.0.2.0/24"]

    service_endpoints = ["Microsoft.Storage.Global"]

    private_endpoint_network_policies = "Disabled"

    private_link_service_network_policies_enabled = true


    delegation {

      name = "Microsoft.AzureCosmosDB/clusters"


      service_delegation {

        name = "Microsoft.AzureCosmosDB/clusters"

        actions = ["Microsoft.Network/virtualNetworks/subnets/join/action"]


      }
    }
  }

}