# Use case of utillizing the module Get-AzVMSku together with Terraform

## Description
In conjunction with the release of the Get-AzVMSku module, it is essential to not only provide specific examples of its usage with various configurations but also demonstrate its practical application. If you haven't reviewed the module's readme, which is accessible here: <a href="https://github.com/ChristofferWin/codeterraform/blob/main/powershell%20projects/modules/Get-AzVMSku/Examples.md">Get-AzVMSku.md</a>, I strongly recommend doing so before proceeding further.

The practical scenario outlined below leverages Terraform and PowerShell to establish an efficient deployment process for creating Azure Virtual Machines in diverse configurations. The PowerShell component plays a crucial role in gathering all necessary information required for deployment purposes.

## Prerequisites
Please see below requirements before cloning the repo:

1. Have Terraform installed, either using a build agent or local machine. Install from => <a href="https://developer.hashicorp.com/terraform/tutorials/aws-get-started/install-cli">Terraform</a>
2. Have PowerShell Core (7) installed. Install from => <a href="https://learn.microsoft.com/en-us/powershell/scripting/install/installing-powershell-on-windows?view=powershell-7.3">PSCore</a>
    1. Get-AzVMSku module installed and its dependencies. Install from => <a href="https://www.powershellgallery.com/packages/Get-AzVMSku/1.9">PSGallery</a>
3. Access to an Azure subscription, which must be active
4. TF providers authenticated - The provided code example utilizes 'direct' authentication, requiring Terraform to operate within the existing terminal context

## Limitations
1. The provided code is meant to be run directly from a terminal and not a DevOps pipeline.
2. The Virtual machine passwords will not be in any other output than the statefile itself.
3. Although the provided code deploys all the essential components for a Virtual Machine to function, it does not include the deployment of a public IP or bastion. This omission is intentional, as the practical example focuses solely on demonstrating the integration possibilities between the PowerShell module and Terraform.


## Blueprint of system flow
Please see below for a visiual description of how these scripting tools work together to form the overall deployment flow:

</br>

![Blueprint](https://github.com/ChristofferWin/codeterraform/blob/main/terraform%20projects/virtual%20machines/Automated-VM-Deployment.drawio.png?raw=true)

## Getting started using provided code
1. Create an empty folder anywhere on the local system
2. Clone down the repo
3. Open the folder 'root/terraform projects/virtual machines/'
4. Open 'variables.tf'
5. Define the list of VM configurations found in the variable 'VM_Objects'
    1. If you are uncertain about the values to set, such as 'OS' or 'VMPattern,' you can use the module to retrieve this information. Refer to the examples here: <a href="https://github.com/ChristofferWin/codeterraform/blob/main/powershell%20projects/modules/Get-AzVMSku/Examples.md#example-4---using-available-switches">Find OS and VMPattern</a>
6. Define the environment found in variable 'env_name'
7. Open the terminal and set the location to '/terraform projects/virtual machines/'
8. Run 'terraform init'
```
terraform init

Terraform has been successfully initialized!

You may now begin working with Terraform. Try running "terraform plan" to see
any changes that are required for your infrastructure. All Terraform commands
should now work.

If you ever set or change modules or backend configuration for Terraform,
rerun this command to reinitialize your working directory. If you forget, other
commands will detect it and remind you to do so if necessary.
```
9. Run 'terraform plan'
    1. If plan is OK, continue
```
terraform plan

Plan: 11 to add, 0 to change, 0 to destroy. (Amount of VM configs defined + VM requirements like vnet)
```
10. Run 'terraform apply --auto-approve=true'
```
terraform apply --auto-approve=true

Apply complete! Resources: 11 added, 0 changed, 0 destroyed.
```
11. Run 'terraform destroy' To remove newly created resources
    1. Only press 'yes' If output is OK

```
terraform destroy

Plan: 0 to add, 0 to change, 11 to destroy.

Do you really want to destroy all resources?
  Terraform will destroy all your managed infrastructure, as shown above.
  There is no undo. Only 'yes' will be accepted to confirm.

  Enter a value: yes

Destroy complete! Resources: 11 destroyed.
```