## Description
Please see below examples in order to get a good understanding of the functionality of the module.

### Example 1 - The use of the show-commands
<span style="color:green">//In case the user is in any doubt about what information to input.
Provide information about which VM series are available in Azure. Use this new information and parse a 'VMPattern' to target only sizes of interest. See example 3 for more details about this parameter.</span>
```
Get-AzVMSku -ShowVMCategories
```
<span style="color:green">//Provide information about which operating systems are supported by the module. The list will grow as the module grows.</span>
```
Get-AzVMSku -ShowVMOperatingSystems
```
<span style="color:green">//Provide information about which locations are supported by the module. Note, if no Azure context exists, the command will throw an exception.</span>
```
Get-AzVMSku -ShowLocations
```
### Example 2 - Simple run of the module with only required params
<span style="color:green">//By only supplying the required params the module will return all possible SKUs & VM sizes of which is available given the current Azure context.</span>
```
Get-AzVMSku -Location westeurope -OperatingSystem windows11
```
<span style="color:green">//Note - For operation purposes, the use of -Verbose is always recommended.</span>

### Example 3 - Using different filter parameters to make return results more specific
<span style="color:green">//Its possible to filter on the specific Vm sizes. Note this filter is not case-sensitive. Only supports a litteral string. Also note that all filters can be used at the same time.</span>
```
Get-AzVMSku -Location westeurope -OperatingSystem windows11 -VMPattern A //For only retreiving a sized vms
```
<span style="color:green">//Filter on SKUs. Note this filter is not case-sensitive. Only supports a litteral string. Also note that in case no SKU's are found, the command will throw an exception.</span>
```
Get-AzVMSku -Location westeurope -OperatingSystem windows11 -OSPattern pro
```
### Example 4 - Using available switches
<span style="color:green">//Multiple different switches are available. They can all be used in any combination.</span>
```
Get-AzVMSku -Location westeurope -OperatingSystem windows10 -RawFormat //Skips data transformation of return values for vm sizes.
```
```
Get-AzVMSku -Location westeurope -OperatingSystem windows10 -NoInteractive //Skips the possibility for the module to request information in a prompt in case information parsed is incorrect. Useful when using the module in any automatic proces, simply note that in case information is wrong and due to the switch being set, the module will throw an exception.
```
```
Get-AzVMSku -Location westeurope -OperatingSystem windows10 -NewestSKUs //Use this instead of the filter 'OSPattern' To only retrieve the absolut newest available SKU.
```
```
Get-AzVMSku -Location westeurope -OperatingSystem windows10 -NewestSKUsVersions //Use this in case only the newest version of each SKU is wanted.
```
```
Get-AzVMSku -Location westeurope -OperatingSystem windows10 -CheckAgreement //Use this to validate whether each SKU has a legal agreement that must be signed prior to deploying the image. In case an agreement is required, the powershell command 'Set-AzMarketplaceTerms' Can be used to accept any terms. Note, this switch can assist in understanding whether an image can simply be used or whether additional actions must be taken. Its therefor always advised to use this feature, just note additional runtime will be added to the command.
```

### Example 5 - Using the module together with ConvertTo-Json so that output can be directly used in an IaC tool like Terraform
<span style="color:green">//Note the return output of any command used for the module will be of type 'pscustomobject' Which makes it easy to convert to other datatypes like simple JSON strings.</span>
```
Get-AzVMSku -Location northeurope -OperatingSystem WINDOWs11 -OSPattern Pro -NewestSKUs -NewestSKUsVersions -CheckAgreement -VMPattern DC32 -Verbose | ConvertTo-Json -Depth 3 | Out-File .\SKUoutput.json //Note, depth 3 must be used
```