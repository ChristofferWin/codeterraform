<#
.SYNOPSIS
    Retrieves Azure Virtual Machine SKUs information based on specified criteria.

.DESCRIPTION
    The Get-AzVMSKU function retrieves Azure Virtual Machine SKUs information based on specified criteria such as location, operating system, and other parameters. It can filter VM SKUs based on different settings and provide detailed information about available SKUs.

.PARAMETER Location
    Specifies the Azure region where you want to retrieve VM SKUs. Mandatory parameter when using ManualSettings parameter set.

.PARAMETER ContinueOnError
    Specifies whether the function should continue processing in case of errors. By default, it stops on errors.

.PARAMETER OperatingSystem
    Specifies the target operating system for which you want to retrieve VM SKUs. Mandatory parameter when using ManualSettings parameter set.

.PARAMETER OSPattern
    Specifies a pattern to filter operating system SKUs. It can be a partial match pattern.

.PARAMETER VMPattern
    Specifies a pattern to filter VM SKUs. It can be a partial match pattern.

.PARAMETER RawFormat
    Specifies whether to return the VM sizes in raw format without formatting.

.PARAMETER NoInteractive
    Specifies whether to suppress interactive prompts.

.PARAMETER NewestSKUs
    Specifies whether to retrieve only the newest available SKUs.

.PARAMETER NewestSKUsVersions
    Specifies whether to retrieve only the newest available versions of SKUs.

.PARAMETER CheckAgreement
    Specifies whether to check for legal agreements for SKUs.

.PARAMETER ShowLocations
    Specifies whether to retrieve and display available Azure locations.

.PARAMETER ShowVMCategories
    Specifies whether to retrieve and display available VM categories.

.PARAMETER ShowVMOperatingSystems
    Specifies whether to retrieve and display available VM operating systems.

.NOTES
    File Name      : Get-AzSku.psm1
    Author         : Christoffer Windahl Madsen
    Prerequisite    : This function requires the Az PowerShell modules: .

.LINK
    https://github.com/ChristofferWin/codeterraform
#>

function Get-AzVMSKU {
    [cmdletBinding(DefaultParameterSetName = 'ManualSettings')]
    param(
        [Parameter(ParameterSetName = "ManualSettings", Mandatory = $true)][string]$Location,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$ContinueOnError,
        [Parameter(ParameterSetName = "ManualSettings", Mandatory = $true)][string]$OperatingSystem,
        [Parameter(ParameterSetName = "ManualSettings")][String]$VMPattern,
        [Parameter(ParameterSetName = "ManualSettings")][string]$OSPattern,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$RawFormat,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$NoInteractive,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$NewestSKUs,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$NewestSKUsVersions,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$CheckAgreement,
        [Parameter(ParameterSetName = "ShowCommandLocations")][switch]$ShowLocations,
        [Parameter(ParameterSetName = "ShowCommandVMs")][switch]$ShowVMCategories,
        [Parameter(ParameterSetName = "ShowCommandVMsOS")][switch]$ShowVMOperatingSystems
    )   

    $MSVMURL = "https://azure.microsoft.com/en-us/pricing/details/virtual-machines/series/"
    $CategoryObjects = @()
    $LocationObjects = @()

    $HelperObjects = @(
        [pscustomobject]@{
            Publisher = "MicrosoftWindowsServer"
            Offer = "WindowsServer"
            SKUs = "" #All WindowsServer SKUs's share the same offer and publisher
            Alias = "Server2008, Server2012, Server2012R2, Server2016, Server2019, Server2022"
        },
        [PSCustomObject]@{
            Publisher = "MicrosoftWindowsDesktop"
            Offer = "Windows-7"
            SKUs = "win7*$OSPattern*"
            Alias = "Windows7"
        },
        [pscustomobject]@{
            Publisher = "MicrosoftWindowsDesktop"
            Offer = "Windows-10"
            SKUs = "win10*$OSPattern*"
            Alias = "Windows10"
        },
        [pscustomobject]@{
            Publisher = "MicrosoftWindowsDesktop"
            Offer = "Windows-11"
            SKUs = "win11*$OSPattern*"
            Alias = "Windows11"
        },
        [pscustomobject]@{
            Publisher = "OpenLogic"
            Offer = "CentOS"
            SKUs = "*$OSPattern*"
            Alias = "CentOS"
        },
        [pscustomobject]@{
            Publisher = "Canonical"
            Offer = "UbuntuServer"
            SKUs = "*$OSPattern*"
            Alias = "Ubuntu"
        },
        [pscustomobject]@{
            Publisher = "Debian"
            Offer = "Debian-10"
            SKUs = "*$OSPattern*"
            Alias = "Debian10"
        },
        [pscustomobject]@{
            Publisher = "Debian"
            Offer = "Debian-11"
            SKUs = "*$OSPattern*"
            Alias = "Debian11"
        },
        [pscustomobject]@{
            Publisher = "Redhat"
            Offer = "rhel"
            SKUs = "*$OSPattern*"
            Alias = "Redhat"
        }
    )

    $FinalOutput = [PSCustomObject]@{
        SubscriptionID = ""
        SubscriptionName = ""
        TenantID = ""
        TenantName = ""
        Publisher = ""
        Offer = ""
        SKUs = "*$OSPattern*"
        Alias = ""
        Versions = [System.Collections.ArrayList]@()
        VMSizes = [System.Collections.ArrayList]@()
        CoresAvailable = 0
        CoresLimit = 0
    }

    $AliasArray = @()
    $AliasArray += $HelperObjects[0].Alias.Split(",")
    $AliasArray += $HelperObjects[1..8].Alias
    $AliasArray = $AliasArray | % {$_.Trim().ToLower()}

    $Context = Get-AzContext
    $FinalOutput.SubscriptionID = $Context.Subscription
    if(!$FinalOutput.SubscriptionID -and (!$ShowVMCategories -and !$ShowVMOperatingSystems)){
        Write-Error "No Azure context found. Please use either Login-AzAccount or Set-AzAdvancedContext to get one..."
        return
    }
    $FinalOutput.SubscriptionName = $Context.Subscription.Name
    $FinalOutput.TenantID = $Context.Tenant.id
    try {
        $FinalOutput.TenantName = (Get-AzTenant -ErrorAction Stop | ? {$_.Id -eq $FinalOutput.TenantID}).Name
    }
    catch {
        Write-Verbose "Was not possible to retrieve the Tenant name, continuing..."
    }

    if(!$ShowVMCategories -and !$ShowVMOperatingSystems){
        try{
            $Locations = Get-AzLocation -ErrorAction Stop
        }
        catch{
            Write-Error "An error occured while trying to retrieve all available locations from Azure...`n$_"
            return
        }
    }

    if($ShowVMOperatingSystems){
        return $AliasArray
    }

    if($ShowLocations){
        try{
            Get-AzVMUsage -Location "abc" -ErrorAction Stop #Made to fail to retrieve exception
        }
        catch{
            $AcceptableLocations = $_.Exception.Message.Split("`n")
            $AcceptableLocations = $AcceptableLocations[2].Split(" ")
            $AcceptableLocations = (($AcceptableLocations[($AcceptableLocations.IndexOf("locations") + 2)..$AcceptableLocations.Length]) -Replace "[^\w\s]", "").Trim()
        }
        foreach($Location in $AcceptableLocations){
            $LocationObjects += [PSCustomObject]@{
               ShortName = $Location
               LongName = ($Locations | ? {$_.Location -eq $Location}).DisplayName
            }
        }
        return $LocationObjects
    }

    if($ShowVMCategories){
        try{
            $WebsiteContent = (Invoke-WebRequest -UseBasicParsing -Uri $MSVMURL -ErrorAction Stop).Content.Split("`n")
        }
        catch{
            Write-Warning "Was not able to retrieve required information from Microsoft for the flag 'ShowVMCategories'..."
        }
        $Categories = $WebsiteContent | ? {$_ -like "*h2*series*"}
        foreach($Category in $Categories){
            $CategoryObjects += [pscustomobject]@{
                Title = $Category.Trim().Replace("<h2>", "").Replace("</h2>", "")
                Description = $WebsiteContent[$WebsiteContent.IndexOf($Category) + 2].Trim() -Replace "<[^>]+>|&#\d+;", ""
            }
        }
        return $CategoryObjects
    }

    do{
        $OperatingSystemOK = $false
        if($AliasArray.Contains($OperatingSystem.ToLower())){
            $OperatingSystemOK = $true
            Continue
        }
        if($NoInteractive -or !$ContinueOnError){
            Write-Error "The provided operating system did not match any possible value.`nPlease use the command again with switch 'ShowVMOperatingSystems'"
            return
        }
        $OperatingSystem = Read-Host "Please provide a valid operating system, if in any doubt, run the function with switch 'ShowVMOperatingSystems'"
    }
    while(!$OperatingSystemOK)

    try{
        Get-AzVMUsage -Location "abc" -ErrorAction Stop #Made to fail to retrieve exception
    }
    catch{
        $AcceptableLocations = $_.Exception.Message.Split("`n")
        $AcceptableLocations = $AcceptableLocations[2].Split(" ")
        $AcceptableLocations = ($AcceptableLocations[($AcceptableLocations.IndexOf("locations") + 2)..$AcceptableLocations.Length]) -Replace "[^\w\s]", "" 
    }

    do{
        $LocationOK = $false
        if($AcceptableLocations.contains(($Location.Replace(" ", "").ToLower()))){
            $LocationOK = $true
            Break
        }
        else{
            if($NoInteractive -or !$ContinueOnError){
                Write-Error "The location: $Location was not found in the Azure database..."
                return
            }
            else{
                Write-Warning "The location: $Location was not found in the Azure database..."
                $Location = Read-Host "Please provide a new location to use... If in any doubt, run this function with the -ShowLocations switch"
            }
        }
    }
    while(!$LocationOK)

    do{
        try{
            $Usage = (Get-AzVMUsage -Location $Location -ErrorAction Stop | ? {$_.Name.LocalizedValue -eq "Total Regional vCPUs"})
            $FinalOutput.CoresAvailable = $Usage.Limit - $Usage.CurrentValue
            $FinalOutput.CoresLimit = $Usage.Limit
        }
        catch{
            if($_.Exception.Message -like "*Microsoft.Azure.Management.Compute.Models.VirtualMachine', on 'T MaxInteger*"){
                $Usage = $true
                Write-Verbose "Powershell_5 detected, cannot retrieve the quota for VM cores on subscription: $SubscriptionID"
                continue
            }
            if(!$ContinueOnError){
                Write-Error "An error occured while trying to retrieve the current quotas from the subscription: $SubscriptionID`n$_"
                return
            }
        }
    }
    while(!$Usage)

    do{
        try{
            if($VMPattern){
               $VMsizes = Get-AzVMSize -Location $Location -ErrorAction Stop | ? {$_.Name -like "Standard_$VMPattern*" -or $_.Name -like "Basic_$VMPattern*"}
               if($VMsizes.Count -eq 0){
                   if($ContinueOnError -and !$NoInteractive){
                       Write-Warning "0 Virtual machine sizes found using pattern '$VMPattern'..."
                       $VMPattern = Read-Host "Please provide a new pattern or simply press return to retrieve all vm sizes instead..."
                       Continue
                   }
                   Write-Error "No Virtual machine sizes found using pattern '$VMPattern'..."
                   return
               }
               Write-Verbose "Found $($VMsizes.Count) VM sizes that matches the pattern '$VMPattern'..."
            }
            else{
                $VMSizes = Get-AzVmSize -Location $Location -ErrorAction Stop
            }
            Write-Verbose "VM sizes successfully retrieved..."
        }
        catch{
            Write-Error "The following error occured while trying to retrieve vm sizes...:`n$_"
            return
        }
        if($RawFormat){
            $FinalOutput.VMSizes = $VMsizes
        }
        else{
            foreach($VM in $VMsizes){
                $FinalOutput.VMSizes.Add([PSCustomObject]@{
                    Name = $VM.Name
                    CoresAvailable = $VM.NumberOfCores
                    MemoryInGB = if($VM.MemoryInMB -gt 0){$VM.MemoryInMB / 1024} else{0} 
                    MaxDataDiskCount = $VM.MaxDataDiskCount
                    OSDiskSizeInGB = if($VM.OSDiskSizeInMB -gt 0){$VM.OSDiskSizeInMB / 1024} else{0}
                    TempDriveSizeInGB = if($VM.ResourceDiskSizeInMB -gt 0){$VM.ResourceDiskSizeInMB / 1024} else{0}
                }) | Out-Null
            }
        }
    }
    while(!$VMsizes)

    switch($OperatingSystem){
        "Server2008" {$FinalOutput.SKUs = "2008-*$OSPattern*"; $FinalOutput.Offer = $HelperObjects[0].Offer; $FinalOutput.Publisher = $HelperObjects[0].Publisher}
        "Server2012" {$FinalOutput.SKUs = "2012-data*$OSPattern*";$FinalOutput.Offer = $HelperObjects[0].Offer; $FinalOutput.Publisher = $HelperObjects[0].Publisher}
        "Server2012R2" {$FinalOutput.SKUs = "2012-r2*$OSPattern*"; $FinalOutput.Offer = $HelperObjects[0].Offer; $FinalOutput.Publisher = $HelperObjects[0].Publisher}
        "Server2016" {$FinalOutput.SKUs = "2016*$OSPattern*"; $FinalOutput.Offer = $HelperObjects[0].Offer; $FinalOutput.Publisher = $HelperObjects[0].Publisher}
        "Server2019" {$FinalOutput.SKUs = "2019*$OSPattern*"; $FinalOutput.Offer = $HelperObjects[0].Offer; $FinalOutput.Publisher = $HelperObjects[0].Publisher}
        "Server2022" {$FinalOutput.SKUs = "2022*$OSPattern*"; $FinalOutput.Offer = $HelperObjects[0].Offer; $FinalOutput.Publisher = $HelperObjects[0].Publisher}
        "Windows7" {$FinalOutput.Offer = $HelperObjects[1].Offer; $FinalOutput.Publisher = $HelperObjects[1].Publisher}
        "Windows10" {$FinalOutput.Offer = $HelperObjects[2].Offer; $FinalOutput.Publisher = $HelperObjects[2].Publisher}
        "Windows11" {$FinalOutput.Offer = $HelperObjects[3].Offer; $FinalOutput.Publisher = $HelperObjects[3].Publisher}
        "CentOS" {$FinalOutput.Offer = $HelperObjects[4].Offer; $FinalOutput.Publisher = $HelperObjects[4].Publisher}
        "Ubuntu" {$FinalOutput.Offer = $HelperObjects[5].Offer; $FinalOutput.Publisher = $HelperObjects[5].Publisher}
        "Debian10" {$FinalOutput.Offer = $HelperObjects[6].Offer; $FinalOutput.Publisher = $HelperObjects[6].Publisher}
        "Debian11" {$FinalOutput.Offer = $HelperObjects[7].Offer; $FinalOutput.Publisher = $HelperObjects[7].Publisher}
        "Redhat" {$FinalOutput.Offer = $HelperObjects[8].Offer; $FinalOutput.Publisher = $HelperObjects[8].Publisher}
    }
 
    try{
        $FinalOutput.SKUs = (Get-AzVMImageSku -Location $Location -PublisherName $FinalOutput.Publisher -Offer $FinalOutput.Offer -ErrorAction Stop | ? {$_.SKUs -like $FinalOutput.SKUs}).Skus
        if($NewestSKUs){
            $FinalOutput.SKUs = $FinalOutput.SKUs[-1]
        }
        if($FinalOutput.SKUs.count -eq 0) {
            Write-Warning "No SKUs found using the operating system: '$OperatingSystem' and OS pattern: '$OSPattern'"
            return
        }
    }
    catch{
        Write-Error "The following error occured while trying to retrieve the current SKUs for OS: '$OperatingSystem'`n$_"
        return
    }
    $i = 0
    do {
        try{
            $SKUPlaceholder = if($FinalOutput.SKUs.count -eq 1){$FinalOutput.Skus}else{$FinalOutput.Skus[$i]}
            $Versions = (Get-AzVMImage -Location $Location -PublisherName $FinalOutput.Publisher -Offer $FinalOutput.Offer -Skus $SKUPlaceholder -ErrorAction Stop).Version
            try{
               if($CheckAgreement) {
                  $Agreement = Get-AzMarketplaceterms -Publisher $FinalOutput.Publisher -Product $FinalOutput.Offer -Name '$($SKUPlaceholder)' -ErrorAction Stop
               }
            }
            catch{
                if($_.Exception.Message -like "*Unable to find legal terms for this offer*" -or $_.Exception.Message -like "*The offer with Offer ID*"){
                    Write-Verbose "No legal terms to sign for sku: $SKUPlaceholder"
                }
                else{
                    Write-Warning "Legal terms to sign for sku: $SKUPlaceholder"
                }
            }
            if($NewestSKUsVersions -and $Versions.Count -gt 0){
                $Versions = $Versions[-1]           
            }
            if($CheckAgreement){
                $FinalOutput.Versions.Add([PSCustomObject]@{
                    SKU = $SKUPlaceholder
                    Versions = $Versions
                    Agreement = $Agreement
                }) | Out-Null
            }
            else {
                $FinalOutput.Versions.Add([PSCustomObject]@{
                    SKU = $SKUPlaceholder
                    Versions = $Versions
                }) | Out-Null
            }
        }
        catch{
            if($_.Exception.Message -like "*VMImage was not found*"){
                Write-Warning "The SKU: $SKUPlaceholder was not found in Azure"
                if($SKUPlaceholder.Count -gt 1) {
                    $FinalOutput.Versions.Remove($SKUPlaceholder)
                }
            }
            else{
                if(!$ContinueOnError){
                    Write-Error "The following error occured while trying to information about the SKU: $SKUPlaceholder`n$_"
                    return
                }
            }
        }
        $i++ #Need to use a do-while instead of for, as we want to run at least 1 time.
    }
    while($i -le $FinalOutput.SKUs.Count -1)
    #Please check whether a pattern of null results in 0 skus being found or simply that every single possible SKUs for a given OS is found
    if($FinalOutput.Versions.SKU.Count -eq 0){
        Write-Error "No SKUs were found for Operating system: $OperatingSystem $(if($OSPattern){'and using OSPattern: $($OSPattern)'})`nTry to change the pattern or simply ommit it..."
        return
    }
    return $FinalOutput
}
Export-ModuleMember Get-AzVMSKU