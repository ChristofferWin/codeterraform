terraform {
  required_providers {
    azurerm = { 
        source = "hashicorp/azurerm"
    }
    kubernetes = {
        source = "hashicorp/kubernetes"
        version = "=2.18.0" //Forcing the version with '='
    }
  }
}

module "kubernetes_module" {
    source = "./kubernetes"
}