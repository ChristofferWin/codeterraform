provider "azurerm" {
  features {
  }
}

module "deployment_1_simple_hub_spoke" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=main"
  typology_object = var.deployment_1_simple_hub_spoke
}

module "deployment_2_simple_with_vpn" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=main"
  typology_object = var.deployment_2_simple_with_vpn
}

module "deployment_3_simple_with_firewall" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=main"
  typology_object = var.deployment_3_simple_with_firewall
}

module "deployment_4_advanced_with_all_custom_values" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=main"
  typology_object = var.deployment_4_mixed_settings
}

module "deployment_5_advanced_with_all_custom_values" {
  source = "github.com/ChristofferWin/codeterraform//terraform projects/modules/azurerm-hub-spoke?ref=main"
  typology_object = var.deployment_5_advanced_with_all_custom_values
} 