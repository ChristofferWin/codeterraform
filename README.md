# codeterraform
Repo build to support the blog residing at https://codeterraform.com

## Most recent update
The PowerShell module used to retrieve Azure Virtual Machine images directly from the Azure Marketplace has received a MASSIVE upgrade in its latest major release, version 3.0.2.

The module now makes it possible to browse the ENTIRE Azure image marketplace, requiring only a valid Azure location to get started.

Read more about the module’s capabilities here. => https://github.com/ChristofferWin/codeterraform/blob/main/powershell%20projects/modules/Get-AzVMSku/Examples.md#table-of-contents

Download and get started today => https://www.powershellgallery.com/packages/Get-AzVMSku/3.0.2

## Newest release: The Terraform module "azurerm-hub-spoke"

This module is capable of deploying an entire hub-spoke typology as per described in the Microsoft Cloud Adoption Framework. See the <a href="https://github.com/ChristofferWin/codeterraform/releases/tag/1.0.0-hub-spoke">Release page</a> for more information!

Below here is a table describing the current states of all production ready modules and their respective version

| Name       | Description            | Test status | Module version | Source code link                  |
|-----------------|-------------------------|-------------|--------------|----------------------------------------|
|   azurerm-hub-spoke    | Deploy hub / spokes in a multi context Terraform environments  | [![azurerm-vm-bundle](https://github.com/ChristofferWin/codeterraform/actions/workflows/Test_terraform_module.yml/badge.svg)](https://github.com/ChristofferWin/codeterraform/actions/workflows/Test_terraform_module.yml) | 2.0.0-hub-spoke | [Source Code](https://github.com/ChristofferWin/codeterraform/blob/2.0.0-hub-spoke/terraform%20projects/modules/azurerm-hub-spoke/azurerm-hub-spoke.tf)|
| azurerm-hub-spoke | Deploy hub / spokes in 1 subscription  |   [![azurerm-vm-bundle](https://github.com/ChristofferWin/codeterraform/actions/workflows/Test_terraform_module.yml/badge.svg)](https://github.com/ChristofferWin/codeterraform/actions/workflows/Test_terraform_module.yml)   |     1.0.0-hub-spoke     | [Source Code](https://github.com/ChristofferWin/codeterraform/blob/1.0.0-hub-spoke/terraform%20projects/modules/azurerm-hub-spoke/azurerm-hub-spoke.tf)  |

![codeterraform.com](https://static.wixstatic.com/media/12b015_965de78de7c74fbda9620030b81f8a1e~mv2.png/v1/fill/w_1230,h_444,al_c,q_90,usm_0.66_1.00_0.01,enc_auto/12b015_965de78de7c74fbda9620030b81f8a1e~mv2.png "Blog logo")