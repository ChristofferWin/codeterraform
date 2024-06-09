deployment_1_simple_hub_spoke = {

  name_prefix = "test1"
  
  hub_object = {
    network = {
      address_spaces = ["10.0.0.0/24", "172.16.0.0/24"]
    } 
  }
  #Making 2 spokes and 3 subnets with 0 custom configuration
  spoke_objects = [
    {
      network = {
        
        subnet_objects = [
          {

          },
          {

          }
        ]
      }
    },
    {
      network = {

        subnet_objects = [
          {

          }
        ]
      }
    }
  ]
}

deployment_2_simple_with_vpn = {

  name_suffix = "test2"
  
  hub_object = {
    network = {
      address_spaces = ["192.168.0.0/22"]
      vpn = {}

      subnet_objects = [
        {
          name = "GatewaySubnet"
        }
      ]
    }
  }

  spoke_objects = [
    {
      network = {
        
        subnet_objects = [
          {

          },
          {

          }
        ]
      }
    }
  ]
}

deployment_3_simple_with_firewall = {

  name_prefix = "test3"
  
  hub_object = {
    network = {
      address_spaces = ["172.16.99.0/24"]
      firewall = {}

      subnet_objects = [
        {
          name = "AzureFirewallSubnet"
        }
      ]
    }
  }

  spoke_objects = [
    {
      network = {
        
        subnet_objects = [
          {

          },
          {

          }
        ]
      }
    }
  ]
}

deployment_4_mixed_settings = {
  customer_name = "contoso"
  name_prefix = "test4"
  env_name = "prod"
  dns_servers = ["8.8.8.8", "8.8.4.4"]
  
  tags = {
    "environment" = "prod"
  }

  address_spaces = ["172.16.0.0/22"]
  subnets_cidr_notation = "/27"

  hub_object = {
    network = {

    }
  }

  spoke_objects = [
    {
      network = {
        subnet_objects = [
        {

        },
        {
          
        }
      ]
      }
    }
  ]
}

deployment_5_advanced_with_all_custom_values = {
  hub_object = {
  rg_name = "custom-rg-hub"
  location = "northeurope"

  tags = {
    "custom" = "tag"
  }
    network = {
      vnet_name = "hub-custom-vnet"
      vnet_cidr_notation = "/26"
      address_spaces = ["172.16.0.0"]
      dns_servers = ["1.1.1.1", "8.8.4.4"]
      vnet_peering_name = "custom-peering"
      vnet_peering_allow_virtual_network_access = false
      vnet_peering_allow_forwarded_traffic = false

      tags = {
        "custom2" = "tag"
      }

      vpn = {
        gw_name = "advanced-gw"
        address_space = ["192.168.0.0/24"]
        pip_name = "advanced-pip"
      }

      firewall = {
        name = "custom-fw"
        threat_intel_mode = true
        pip_name = "fw-custom-pip"
        no_logs = true
        no_rules = true
      }

      subnet_objects = [
        {
          name = "subnet1-customhub"
          address_prefix = ["172.16.0.0/27"]
        },
        {
          name = "subnet2-customhub"
          address_prefix = ["172.16.0.32/27"]
        }
      ]
    }
  }

  spoke_objects = [
    {
      rg_name = "spoke1-custom-rg"
      location = "eastus"
      
      tags = {
        "custom2" = "tag"
      }

      network = {
        subnet_objects = [
          {
             name = "subnet-custom1-spoke1"
             use_last_subnet = true
          },
          {
            name = "subnet-custom2-spoke1"
            use_last_subnet = true
          }
        ]
      }
    },
    {
      rg_name = "spoke2-custom-rg"
      location = "westus"
      
      tags = {
        "custom1" = "tag"
      }

      network = {
        subnet_objects = [
          {
             name = "subnet-custom1-spoke2"
             use_first_subnet = true
          },
          {
            name = "subnet-custom2-spoke2"
            use_first_subnet = true
          }
        ]
      }
    }
  ]
}
