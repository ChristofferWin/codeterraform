deployment_1_simple_hub_spoke = {
  
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

deployment_4_advanced_with_all_top_level_custom_values = {
  customer_name = "contoso"
  name_prefix = "fuck-sake"
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
/*
deployment_5_advanced_with_all_custom_values = {

}
*/