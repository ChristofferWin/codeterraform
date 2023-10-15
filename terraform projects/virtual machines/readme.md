# Use case of utillizing the module Get-AzVMSku together with Terraform

## Description
In conjunction with the release of the Get-AzVMSku module, it is essential to not only provide specific examples of its usage with various configurations but also demonstrate its practical application. If you haven't reviewed the module's readme, which is accessible here: <a href="https://github.com/ChristofferWin/codeterraform/blob/main/powershell%20projects/modules/Get-AzVMSku/Examples.md"><Get-AzVMSku.md</a>, I strongly recommend doing so before proceeding further.

The practical scenario outlined below leverages Terraform and PowerShell to establish an efficient deployment process for creating Azure Virtual Machines in diverse configurations. The PowerShell component plays a crucial role in gathering all necessary information required for deployment purposes.

## Blueprint of system flow
Please see below for a visiual description of how these scripting tools work together to form the overall deployment flow:
![Blueprint](https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/virtual%20machines/Automated-VM-Deployment.drawio.png?raw=true)