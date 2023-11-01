param (
    [string]$Location = "westeurope", #Default value
    [string]$OS,
    [string]$VMPattern,
    [string]$OSPattern,
    [string]$OutputFileName = ".\SKUs.json",
    [switch]$OnlyWithVersion
)

if($OnlyWithVersion) {
    Get-AzVMSKu -Location $Location -OperatingSystem $OS -NewestSKUs -NewestSKUsVersions -Verbose -VMPattern $VMPattern -OSPattern $OSPattern | ConvertTo-Json -Depth 3 | Out-File -FilePath $OutputFileName -Force
}
else{
    Get-AzVMSKu -Location $Location -OperatingSystem $OS -NewestSKUs -Verbose -VMPattern $VMPattern -OSPattern $OSPattern | ConvertTo-Json -Depth 3 | Out-File -FilePath $OutputFileName -Force
    #Get-AzVMSKu -Location $Location -OperatingSystem $OS -NewestSKUs -NewestSKUsVersions -Verbose -VMPattern $VMPattern -OSPattern $OSPattern | ConvertTo-Json -Depth 3 | Out-File -FilePath $OutputFileName -Force
}