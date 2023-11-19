param (
    [string]$Location,
    [switch]$AllowNoVersions,
    [string]$OS,
    [string]$VMPattern,
    [string]$OutputFileName = ".\SKUs.json"
)

$Module = Get-Module -ListAvailable | ? {$_.Name -eq "Get-AzVMSku"}
if($Module.Length -eq 0){
    Install-Module -Name Get-AzVMSku -Force
}

Import-Module Get-AzVMSku

try{
    $Skus = Get-AzVMSKu -Location $Location -OperatingSystem $OS -NewestSKUsVersions -VMPattern $VMPattern -ErrorAction Stop
}
catch{
    $_
    Exit
}

for($i = $Skus.Versions.count -1; $i -ne 0; $i--){
   
    if($SKUs.Versions[$i].Versions.Length -gt 0 -and $SKUs.Versions[$i].SKU -notlike "*cn*"){
        $Value = [PSCustomObject]@{
            SubscriptionID = $SKUs.SubscriptionID
            SubscriptionName = $SKUs.SubscriptionName
            TenantID = $SKUs.TenantID
            TenantName = $SKUs.TenantName
            Publisher = $SKUs.Publisher
            Offer = $SKUs.Offer
            SKUs = $SKUs.SKUs[$i]
            Versions = @($SKUs.Versions[$i])
            VMSizes = $SKUs.VMSizes
            CoresAvailable = $SKUs.CoresAvailable
            CoresLimit = $SKUs.CoresLimit
        } 
       $Value | ConvertTo-Json -Depth 3 | Out-File $OutputFileName -Force
       Return
    }
    Write-Output "The SKU: $($SKUs.Versions[$i].SKU) will be skipped due to missing version..."
}

if($AllowNoVersions){
    $Skus | ConvertTo-Json -Depth 3 | Out-File $OutputFileName -Force
    Return
}