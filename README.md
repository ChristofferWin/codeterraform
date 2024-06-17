# codeterraform
Repo build to support the blog residing at https://codeterraform.com

## Newest release: The Terraform module "azurerm-hub-spoke"

This module is capable of deploying an entire hub-spoke typology as per described in the Microsoft Cloud Adoption Framework. See the <a href="https://github.com/ChristofferWin/codeterraform/releases/tag/1.0.0-hub-spoke">Release page</a> for more information!

Below here is a table describing the current states of all production ready modules and their respective version

| Test name       | Test purpose            | Test status | Test version | Test source code link                  |
|-----------------|-------------------------|-------------|--------------|----------------------------------------|
| Unit-testing      | CI action for commits to main  | [![azurerm-vm-bundle](https://github.com/ChristofferWin/codeterraform/actions/workflows/Test_terraform_module.yml/badge.svg)](https://github.com/ChristofferWin/codeterraform/actions/workflows/Test_terraform_module.yml) | 1.0.0 | [Source Code](https://github.com/ChristofferWin/codeterraform/tree/main/terraform%20projects/modules/test%20modules/hub-bundle/unit-testing)|
| Releases | Manual trigger to verify before publishing new release  |   [![azurerm-vm-bundle](https://github.com/ChristofferWin/codeterraform/actions/workflows/Test_terraform_module.yml/badge.svg)](https://github.com/ChristofferWin/codeterraform/actions/workflows/Test_terraform_module.yml)   |     1.0.0     | [Source Code](https://github.com/ChristofferWin/codeterraform/tree/main/terraform%20projects/modules/test%20modules/hub-bundle/release-testing)  |

![codeterraform.com](https://static.wixstatic.com/media/12b015_965de78de7c74fbda9620030b81f8a1e~mv2.png/v1/fill/w_1230,h_444,al_c,q_90,usm_0.66_1.00_0.01,enc_auto/12b015_965de78de7c74fbda9620030b81f8a1e~mv2.png "Blog logo")