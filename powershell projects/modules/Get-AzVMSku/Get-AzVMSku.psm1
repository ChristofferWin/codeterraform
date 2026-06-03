function Select-Choice {
    param(
        [string[]]$ArrayOfOptions,
        [string]$ParameterName,
        [string]$Message
    )
    $MissingResponse = $true
    if($ArrayOfOptions.Count -eq 1){
        return $ArrayOfOptions[0]
    }
    do {
        $i = 1
        $Choices = @()
        if($ArrayOfOptions.Count -gt 1 -and !$NoInteractive){
            Write-Host -ForegroundColor Blue "###### Please select a specific $ParameterName below ######"
            foreach($Option in $ArrayOfOptions) {
                Write-Host "[$i] - $($Option)"
                $Choices += [PSCustomObject]@{
                    Name = $Option
                    Choice = $i
                }
                $i++
            }
            Write-Host -ForegroundColor Blue -NoNewline "Your selection...: "
            $UserChoice = Read-Host
            foreach($Choice in $Choices) {
                if($Choice.Choice -eq $UserChoice) {
                    Write-Verbose "The option of '$($Choice.Name)' has been selected..."
                    $Name = $Choice.Name
                    $MissingResponse = $false
                    return $Name
                }
            }
            if($MissingResponse) {
                Write-Warning "The choice of '$UserChoice' Is invalid, please select another Option..."
            }
        } elseif($ArrayOfOptions.Count -gt 1 && $NoInteractive) {
            if($Message){
                Write-Error "The command found multiple results and require user-input to continue. $Message" #In this case - The parameter name is a defined error defined from where the function is called
            } else {
                Write-Error "The command found multiple results and require user-input to continue. Please either use a more specific name for '$ParameterName' To Limit the $ParameterName result to 1, or remove the switch 'NoInteractive'"

            }
            return
        } else {
            return $Choices[0].Name
        }
    } while($MissingResponse)
}

<#
.SYNOPSIS
    Retrieves information about Azure VM image SKUs, publishers, offers, and available VM sizes, optionally formatted for automation.
 
.DESCRIPTION
    This function allows dynamic retrieval of Azure VM image metadata including publisher, offer, SKU, and version.
    It supports both interactive and non-interactive modes, and can output raw data for automation.
    Both the "PublisherName" & "OfferName" parameters are case-insensitive
 
    It also allows you to (In the context of a specific Azure Subscription):
    - Explore available VM image publishers, offers, skus and their respective versions
    - View supported VM sizes in a given region
    - Analyze quotas and VM family availability
    - Output structured data as JSON for programmatic use
 
.PARAMETER Location
    The Azure region to use (e.g., 'westeurope'). Required for all operations.
 
.PARAMETER VMPattern
    Optional filter for VM size names (e.g., 'D', 'E', etc.).
 
.PARAMETER PublisherName
    The name or pattern of the image publisher

.PARAMETER PublisherNameStartsWith
    Search through image publishers where it starts with your input
 
.PARAMETER OfferName
    The name or pattern of the image offer. Requires -PublisherName to be set.
 
.PARAMETER Top10NewestVersions
    Show only the newest 10 versions of a specific SKU of a Azure Market Place vendor-offer
 
.PARAMETER NewestSKUs
    Automatically select the newest SKU from the selected offer.
 
.PARAMETER NewestSKUsVersions
    Automatically select the newest image version for the selected SKU.
 
.PARAMETER NoInteractive
    Prevents all user interaction; must provide all required inputs via parameters, see examples.
 
.PARAMETER RawFormat
    Returns as json.

.PARAMETER UnfilteredPublishers
    Per default the module filters publisher names containing strings like test, punctuation and large numbers. Use this switch to include them
 
.PARAMETER NoVMInformation
    Skips VM size lookup and quota analysis (faster).
 
.PARAMETER ShowLocations
    Shows all valid Azure regions.
 
.PARAMETER ShowVMCategories
    Displays VM categories (e.g., general purpose, compute optimized) with descriptions.
 
.EXAMPLE
    Get-AzVMSku -Location "westeurope"
 
    Starts an interactive session to explore publishers, offers, and SKUs in West Europe.
 
.EXAMPLE
    Get-AzVMSku -Location "westeurope" -PublisherName "palo" #Could even do PALO as its incase-sensitive
 
    Starts an interactive session to look at ALL Palo-Altos current Offers in West Europe.

.EXAMPLE
    Get-AzVMSku -Location "westeurope" -PublisherNameStartsWith "microsoft"

    Look through only vendors with images starting with microsoft
 
.EXAMPLE
    Get-AzVMSku -Location "eastus" -PublisherName "bitnami" -OfferName "wordpress" -NewestSKUs -NewestSKUsVersions -NoInteractive | Set-AzVMSku -Force
 
    Non-interactively retrieves the latest Bitnami image version and SKU.
 
.EXAMPLE
    Get-AzVMSku -Location "northeurope" -VMPattern "Ds"
     
    Starts an interactive session to explore publishers, offers, and SKUs in West Europe. It also limits the vm's found
 
.EXAMPLE
    Get-AzVMSku -Location "northeurope" -NoVMInformation
     
    Starts an interactive session to explore publishers - Does not provide any information about vm's (runs fast)
 
.EXAMPLE
    Get-AzVMSku -ShowLocations
 
    Outputs all Azure regions that can be used with VM image commands.

.EXAMPLE
    Get-AzVMSku -ShowVMCategories
 
    Outputs Microsoft descriptions of each Azure Virtual Machine family
 
.EXAMPLE
    Get-AzVMSku -Location "northeurope" -RawFormat | Out-File az-skus.json
 
    Saves raw JSON-formatted output of image and VM size data to a file.
 
.OUTPUTS
    System.Management.Automation.PSCustomObject
 
.NOTES
    - You must be logged in to Azure using `Connect-AzAccount` before using this function.
    - When using -NoInteractive, you must specify required parameters such as PublisherName.
    - Over 2,500 publishers are available in Azure; exact names may vary across regions.
 
.LINK
    https://github.com/ChristofferWin/codeterraform
 
.LINK
    https://codeterraform.com/blog
#>
function Get-AzVMSku {
    [cmdletBinding(DefaultParameterSetName = 'ManualSettings')]
    param(
        [Parameter(ParameterSetName = "ManualSettings", Mandatory = $true)][string]$Location,
        [Parameter(ParameterSetName = "ManualSettings")][string]$VMPattern,
        [parameter(ParameterSetName = "ManualSettings")][string]$PublisherName,
        [parameter(ParameterSetName = "ManualSettings")][string]$PublisherNameStartsWith,
        [parameter(ParameterSetName = "ManualSettings")][string]$OfferName,
        [parameter(ParameterSetName = "ManualSettings")][switch]$Top10NewestVersions,
        #[Parameter(ParameterSetName = "ManualSettings")][switch]$ContinueOnError,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$NewestSKUs,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$NewestSKUsVersions,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$NoInteractive,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$RawFormat,
        [Parameter(ParameterSetName = "ShowCommandLocations")][switch]$ShowLocations,
        [Parameter(ParameterSetName = "ShowCommandVMs")][switch]$ShowVMCategories,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$NoVMInformation,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$UnfilteredPublishers
    )
    Update-AzConfig -DisplayBreakingChangeWarning $false | Out-Null
    $MSVMURL = "https://azure.microsoft.com/en-us/pricing/details/virtual-machines/series/"
    $CategoryObjects = @()
    $LocationObjects = @()
    $FinalOutput = [PSCustomObject]@{
        Context = [PSCustomObject]@{
            SubscriptionID = ""
            SubscriptionName = ""
            TenantID = ""
            TenantName = ""
        }
        Publisher = ""
        Offer = ""
        SKU = ""
        Version = ""
        URN = ""
        VMs = $null
        VMSizePattern = $VMPattern
        Agreement = $null
    }

    if ($PublisherName -and $PublisherNameStartsWith) {
        Write-Warning "Since both PublisherName & PublisherNameStartsWith has been parsed, PublisherName wins and the search for publishers will follow search *$PublisherName*"
    }

    if(!$PublisherName -and !$ShowLocations -and !$ShowVMCategories -and !$PublisherNameStartsWith){
        if($NoInteractive){
            Write-Error "You must provide a PublisherName because the switch -NoInteractive is true"
            return
        }
        Write-Warning "No PublisherName provided. The module will retrieve every single publisher of VM Images in Azure..."
        Start-Sleep -Seconds 3
    } 
    
    if($VMPattern -and $NoVMInformation){
       Write-Warning "The switch -NoVMInformation is true - The VM pattern of '$VMPattern' will be ignored..."
    }
    
    if($Top10NewestVersions -and $NewestSKUsVersions){
        Write-Warning "the switch 'Top10NewestVersions' Will be ignored because switch 'NewestSKUsVersions' Is set"
    }
    
    if($OfferName -and !$PublisherName){
        Write-Error "A PublisherName must be provided when the OfferName is used..."
        return
    }
    
    try {
        $Context = Get-AzContext -ErrorAction Stop
    }catch{
        Write-Error "You must be logged into Azure on an active subscription to run this module"
        return
    }
    $FinalOutput.Context.TenantID = $Context.Tenant.id
    $FinalOutput.Context.SubscriptionName = $Context.Subscription.Name
    try {
        $FinalOutput.Context.TenantName = (Get-AzTenant -ErrorAction Stop | ? {$_.Id -eq $FinalOutput.TenantID}).Name
    }
    catch {
        Write-Verbose "Was not possible to retrieve the Tenant name, continuing..."
    }
    $FinalOutput.Context.SubscriptionID = $Context.Subscription.Id
    if(!$FinalOutput.Context.SubscriptionID -and (!$ShowVMCategories -and !$ShowVMOperatingSystems)){
        Write-Error "No Azure context found. Please use either Connect-AzAccount or Set-AzAdvancedContext to get one..."
        return
    }
    
    if(!$ShowVMCategories){
        try{
            $Locations = Get-AzLocation -ErrorAction Stop
        }
        catch{
            Write-Error "An error occured while trying to retrieve all available locations from Azure...`n$_"
            return
        }
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
               'Name to use' = $Location
               'Display name' = ($Locations | ? {$_.Location -eq $Location}).DisplayName
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
    try{
        Get-AzVMUsage -Location "abc" -ErrorAction Stop #Made to fail to retrieve exception
    }
    catch{
        $AcceptableLocations = $_.Exception.Message.Split("`n")
        $AcceptableLocations = $AcceptableLocations[2].Split(" ")
        $AcceptableLocations = (($AcceptableLocations[($AcceptableLocations.IndexOf("locations") + 2)..$AcceptableLocations.Length]) -Replace "[^\w\s]", "").Trim()
    }
    
    if(!$AcceptableLocations.Contains($Location)){
        Write-Error "The location provided '$Location' is not valid...`nPlease provide one of the following locations:`n$AcceptableLocations"
        return
    }
     try {
        $AzurePublishers = Get-AzVMImagePublisher -Location $Location -ErrorAction Stop
    } catch {
        return $_
    }
    $AzurePublishers = $AzurePublishers | Sort-Object -Property PublisherName
    if(!$ShowVMCategories){
        $CapturedPublishers = $AzurePublishers | ? {$_.PublisherName.ToLower() -eq $PublisherName.ToLower()} #MATCH to lower
        if($CapturedPublishers.Count -eq 0) {
            Write-Verbose "No exact match found for PublisherName '$PublisherName' trying with wild-cards..."
            if($PublisherName) {
                $CapturedPublishers = $AzurePublishers | ? {$_.PublisherName.ToLower() -like "*$($PublisherName.ToLower())*"}
            } elseif($PublisherNameStartsWith) {
                $CapturedPublishers = $AzurePublishers | ? {$_.PublisherName -like "$($PublisherNameStartsWith.ToLower())*"}
            } else {
                $CapturedPublishers = $AzurePublishers
            }
            if($CapturedPublishers.Count -eq 0) {
                Write-Warning "No Publishers found using PublisherName '$PublisherName' The module searches using wild-cards as *<PublisherName>* Adjust the name and try again..."
                return
            } elseif($CapturedPublishers.Count -eq 1){
                Write-Host -ForegroundColor Green "1 match found for PublisherName '$PublisherName' => $($CapturedPublishers[0].PublisherName)"
            }
        } elseif($CapturedPublishers.Count -eq 1){
            Write-Host -ForegroundColor Green "1 exact match for PublisherName '$($CapturedPublishers[0].PublisherName)' found"
        }
        if (!$UnfilteredPublishers -and $CapturedPublishers.Count -gt 1) {
            Write-Verbose "As per default removing any publishers with the string 'test' In it.`nTo include all test publishers, use switch `IncludeTestPublishers"
            $CapturedPublishers = $CapturedPublishers | ? {$_.PublisherName -notmatch '(?i)(test|\.|\d{4,})'}
        }
        $MissingValidResponse = $true
        do {
            try {
                $FinalOutput.Publisher = Select-Choice -ArrayOfOptions $CapturedPublishers.PublisherName -ParameterName "Image Publisher" -ErrorAction Stop
            } catch{
                Write-Warning "No Publishers found for the given PublisherName $($FinalOutput.Publisher)`nReturning..."
                Start-Sleep -Seconds 3
                continue
            }
    
            if($FinalOutput.Publisher -in $null){
                return
            }
            
            try {
                $AzureOffers = Get-AzVMImageOffer -Location $Location -PublisherName $FinalOutput.Publisher -ErrorAction Stop
            } catch{
                return $_
            }
        
            if($AzureOffers.Count -eq 0) {
                Write-Warning "No Offer found for the given PublisherName $($FinalOutput.Publisher)`nReturning..."
                Start-Sleep -Seconds 3
                Continue
            }
            $AzureOffers = $AzureOffers | Sort-Object -Property Offer
            try {
                $NewListOfOffers = @()
                    if($OfferName) {
                        $NewListOfOffers = $AzureOffers.Offer | ? {$_.Trim().ToLower() -eq $OfferName.Trim().ToLower()}
                        if($NewListOfOffers.Count -eq 0) {
                            Write-Verbose "No exact match found for OfferName '$OfferName' trying with wild-cards..."
                            $NewListOfOffers = $AzureOffers.Offer | ? {$_ -like "*$OfferName*"}
                            if($NewListOfOffers.Count -eq 0) {
                                Write-Warning "No Offers found using OfferName '$Offername' The module searches using wild-cards as *<Offername>* Adjust the name and try again`nReturning..."
                                Start-Sleep -Seconds 3
                                return
                            } elseif($NewListOfOffers.Count -eq 1){
                                Write-Host -ForegroundColor Green "1 match for OfferName '$($NewListOfOffers)' found using wild-cards"
                            }
                        } elseif($NewListOfOffers.Count -eq 1 -and $NewListOfOffers[0] -ne ''){
                            Write-Host -ForegroundColor Green "1 exact match for OfferName '$($NewListOfOffers)' found"
                        }
                    } else {
                        $NewListOfOffers = $AzureOffers.Offer
                    }
                $FinalOutput.Offer = Select-Choice -ArrayOfOptions $NewListOfOffers -ParameterName "Image Offer" -ErrorAction Stop
            } catch{
                Write-Warning "No Offer found for the given PublisherName $($FinalOutput.Publisher)`nReturning..."
                Start-Sleep -Seconds 3
                Continue
            }
    
            if($FinalOutput.Offer -in $null){
                return
            }
        
            try {
                $AzureSkus = Get-AzVMImageSku -Location $Location -PublisherName $FinalOutput.Publisher -Offer $FinalOutput.Offer
            } catch {
                return $_
            }
        
            if($AzureSkus.Count -eq 0) {
                Write-Warning "No SKU found for the given offer $($FinalOutput.Offer)`nReturning..."
                Start-Sleep -Seconds 3
                Continue
            }
        
            try {
                if($NewestSKUs){
                    $FinalOutput.Sku = $AzureSkus[-1].Skus
                } else {
                    $FinalOutput.Sku = Select-Choice -ArrayOfOptions $AzureSkus.Skus -ParameterName "Image SKU" -Message "Either remove switch -NoInteractive to allow for user-input OR add switch -NewestSKUs" -ErrorAction Stop
                }
            } catch{
                Write-Warning "No SKU found for the given offer $($FinalOutput.Offer)`nReturning..."
                Start-Sleep -Seconds 3
                Continue
            }
    
            if($FinalOutput.Sku -in $null){
                return
            }
            try {
                $AzureImages = Get-AzVMImage -Location $Location -PublisherName $FinalOutput.Publisher -Offer $FinalOutput.Offer -Skus $FinalOutput.Sku
            } catch {
                return $_
            }
        
            if($AzureImages.Count -eq 0) {
                if($OfferName){
                    Write-Error "The OfferName: '$($OfferName)' is not valid under PublisherName: '$($FinalOutput.Publisher)'`nPlease either change the value or ommit it entirely..."
                    Return
                }
                Write-Warning "No Images found for the given URN: $($FinalOutput.Publisher):$($FinalOutput.Offer):$($FinalOutput.Sku)`nReturning..."
                Start-Sleep -Seconds 3
                Continue
            }
            $AzureImages = $AzureImages.Version | Sort-Object { [version]$_ }
            try {
                if($NewestSKUsVersions){
                    $FinalOutput.Version = $AzureImages[-1]
                } else {
                    $NewListOfVersions = @()
                    if($Top10NewestVersions){
                        $NewListOfVersions = $AzureImages | Select -Last 10
                    } else {
                        $NewListOfVersions = $AzureImages
                    }
                    $FinalOutput.Version = Select-Choice -ArrayOfOptions $NewListOfVersions -ParameterName "Image version" -Message "Either remove switch -NoInteractive to allow for user-input OR add switch -NewestSKUsVersion" -ErrorAction Stop
                }
            } catch{
                Write-Warning "No Version found for the given SKU $($FinalOutput.Version)`nReturning..."
                Start-Sleep -Seconds 3
                Continue
            }
            $MissingValidResponse = $false
        }while($MissingValidResponse)
        
        if($FinalOutput.Version -in $null){
            $FinalOutput.TenantID = ""; $FinalOutput.TenantName = ""; $FinalOutput.SubscriptionID = ""; $FinalOutput.SubscriptionName = ""
            Write-Error "No Version was found - This might be due to a bug.`nReport the following to https://github.com/ChristofferWin/codeterraform/issues/new`n`nCURRENT OUTPUT:`n$($FinalOutput | Format-List | Out-String)" 
            return
        }
    
        try {
            $FinalImage = Get-AzVMImage -Location $Location -PublisherName $FinalOutput.Publisher -Offer $FinalOutput.Offer -Skus $FinalOutput.Sku -Version $FinalOutput.Version -ErrorAction Stop
        } catch{
            return $_
        }
        $FinalOutput.URN = "$($FinalOutput.Publisher):$($FinalOutput.Offer):$($FinalOutput.Sku):$($FinalOutput.Version)"
        if ($FinalImage.PurchasePlan -notin $null){
            Write-Warning "Purchase plan detected for image: $($FinalOutput.URN)"
            Write-Verbose "You can use the command Set-AzVMSku to accept the Azure Market terms. Please note that deploying images with terms that are NOT accepted will lead to a failed deployment..."
        }
    }

    Write-Host @"
    #######################################
    ######### Image definition ############
    #######################################
                                         
    Publisher: $($FinalOutput.Publisher)
    Offer : $($FinalOutput.Offer)
    SKU : $($FinalOutput.Sku)
    Version : $($FinalOutput.Version)
    URN : $($FinalOutput.URN)
    Image agreement: $(if($FinalImage.PurchasePlan -notin $null){"True"}else{"False"})
                                         
    #######################################
"@ -ForegroundColor Green

    if(!$NoVMInformation) {
        Write-Verbose "Retrieving information about Azure Virtual Machines directly from Microsoft, this may take a moment..."
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
    }

    do{
        $LocationOK = $false
        if($AcceptableLocations.contains(($Location.Replace(" ", "").ToLower()))){
            $LocationOK = $true
            Break
        }
        else{
            if($NoInteractive -or !$ContinueOnError){
                Write-Error "The location: $Location was not found in the Azure database...`nUse command Get-AzVmSku -ShowLocations to see all valid values"
                return
            }
            else{
                Write-Warning "The location: $Location was not found in the Azure database..."
                $Location = Read-Host "Please provide a new location to use... If in any doubt, run this function with the -ShowLocations switch"
            }
        }
    }
    while(!$LocationOK)

    if(!$NoVMInformation) {
        do{
            try{
                if($VMPattern){
                $VMsizes = Get-AzComputeResourceSku -Location $Location -ErrorAction Stop | ? {$_.ResourceType -eq "virtualMachines"} | ? {$_.Name -like "Standard_$VMPattern*" -or $_.Name -like "Basic_$VMPattern*"}
                if($VMsizes.Count -eq 0){
                    if($ContinueOnError -and !$NoInteractive){
                        Write-Warning "0 Virtual machine sizes found using pattern '$VMPattern'..."
                        $VMPattern = Read-Host "Please provide a new pattern or simply press return to retrieve all vm sizes instead..."
                        Continue
                    }
                    Write-Warning "No Virtual machine sizes found using pattern '$VMPattern' No VM information is included in the output"
                    break
                }
                Write-Verbose "Found $($VMsizes.Count) VM sizes that matches the pattern '$VMPattern'..."
                }
                else{
                    $VMSizes = Get-AzComputeResourceSku -Location $Location -ErrorAction Stop | ? {$_.ResourceType -eq "virtualMachines"}
                }
                Write-Verbose "VM sizes successfully retrieved..."
            }
            catch{
                Write-Error "The following error occured while trying to retrieve vm sizes...:`n$_"
                return
            }
            $VMs = @()
            foreach($VM in $VMsizes){
               $VMSpecs = $VM.Capabilities
               $VMs += ([PSCustomObject]@{
                    Name = $VM.Name
                    CoresAvailable = ($VMSpecs | ? {$_.Name -eq "vCPUs"}).Value
                    MemoryInGB = ($VMSpecs | ? {$_.Name -eq "MemoryGB"}).Value
                    MaxDataDiskCount = ($VMSpecs | ? {$_.Name -eq "MaxDataDiskCount"}).Value
                    HyperVGeneration = ($VMSpecs | ? {$_.Name -eq "HyperVGenerations"}).Value
                    MaxNetworkInterfaces = ($VMSpecs | ? {$_.Name -eq "MaxNetworkInterfaces"}).Value
                    RetirementDate = if($null -in ($VMSpecs | ? {$_.Name -eq "RetirementDateUtc"}).Value){"No date found"}else{($VMSpecs | ? {$_.Name -eq "RetirementDateUtc"}).Value}
               })
            }

        }
        while(!$VMsizes)
        try{
            $AzureVmUsuage = Get-AzVMUsage -Location $Location -ErrorAction Stop | ? {$_.Name.LocalizedValue -like "*Standard*Family*"}
        }
        catch{
            return $_
        }
        
        $QuotasWithVMs = @{}
        foreach ($VM in $VMsizes) {
            $VMFamiliyPartName = $VM.Name -replace '^[^_]+_([A-Za-z])\d+([a-z]+)_v(\d+)$', '$1$2v$3'
            $VMSpecs = $VM.Capabilities
            foreach ($Quota in $AzureVmUsuage) {
                if ($Quota.Name.Value -like "*$VMFamiliyPartName*") {
                    $FamilyKey = "$VMFamiliyPartName-Family"
        
                    if (-not $QuotasWithVMs.ContainsKey($FamilyKey)) {
                        if ($Quota.Limit -gt 0 -and $Quota.CurrentValue -gt 0) {
                            $CPUsPercentUsage = [math]::Floor($Quota.CurrentValue / $Quota.Limit * 100)
                        } else {
                            $CPUsPercentUsage = 0
                        }
        
                        $QuotasWithVMs[$FamilyKey] = [pscustomobject]@{
                            FamilyName = $FamilyKey
                            Status = ""
                            AvailablevCPUQuota = $Quota.Limit - $Quota.CurrentValue
                            CPUQuotaConsumedPercent = "$CPUsPercentUsage %"
                            RemainingVMCapacity = @()
                            Sizes = @()
                            VMsCanBeDeployed = $false
                        }
                    }
                    if ($QuotasWithVMs[$FamilyKey].AvailablevCPUQuota -gt 0 -and ($VMSpecs | ? {$_.Name -eq "vCPUs"}).Value -gt 0) {
                        $VMCountBeforeLimit = [int]([math]::Floor($QuotasWithVMs[$FamilyKey].AvailablevCPUQuota / ($VMSpecs | ? {$_.Name -eq "vCPUs"}).Value))
                    } else {
                        $VMCountBeforeLimit = 0
                    }
        
                    $existingSizes = $QuotasWithVMs[$FamilyKey].RemainingVMCapacity | ? { $_.SizeName -eq $VM.Name }
                    if (-not $existingSizes) {
                        $QuotasWithVMs[$FamilyKey].RemainingVMCapacity += [PSCustomObject]@{
                            SizeName = $VM.Name
                            VMCountBeforeQuotaLimit = $VMCountBeforeLimit
                        }
                    }
                }
            }
        }    
    }

    $FinalOutput.Agreement = $FinalImage.PurchasePlan
    $Quotas = @()
    $Quotas += $QuotasWithVMs.Values
    for($x = 0; $x -le $Quotas.Count -1; $x++){
        if($Quotas[$x].AvailablevCPUQuota -eq 0) {
            $Quotas[$x].Status = "No vCPU quota available. Request a quota increase from Microsoft"
        } elseif(($Quotas[$x].RemainingVMCapacity.VMCountBeforeQuotaLimit | Measure-Object -Sum).Sum -eq 0) {
            $Quotas[$x].Status = "Quota is available for this VM family, but the available quota is insufficient to deploy any VM sizes in the family"
        } else {
            $Quotas[$x].Status = "VMs in this family can be deployed. Check the RemainingVMCapacity property for details"
            $Quotas[$x].VMsCanBeDeployed = $true
        }
        $Quotas[$x].RemainingVMCapacity = $Quotas[$x].RemainingVMCapacity | Sort-Object -Property VMCountBeforeQuotaLimit -Descending
    }
    $FinalOutput.VMs = $Quotas
    $VmSizesMap = @{}
    foreach($VM in $VMs){
        $VmSizesMap[$VM.Name] = $VM
    }
    for($x = 0; $x -le $FinalOutput.VMs.Count -1; $x++) {
        $VMsToAdd = @()
        foreach($Size in $FinalOutput.VMs[$x].RemainingVMCapacity) {
            $VMsToAdd += $VmSizesMap[$Size.SizeName]
        }
        $FinalOutput.VMs[$x].Sizes = $VMsToAdd
    }

    if($RawFormat){
        if($VMs.Count -ge 10){
            Write-Warning "The output is very large, its recommended to pipe the output to a file..."
            Start-Sleep -Seconds 2
        }
        $FinalOutput = $FinalOutput | ConvertTo-Json -Depth 50
    }
    return $FinalOutput
}

<#
.SYNOPSIS
    Accepts the Azure Marketplace terms required to deploy a virtual machine image.
 
.DESCRIPTION
    This function accepts the licensing terms for a given VM image that requires user consent through the Azure Marketplace.
    It must be provided with a valid object output from Get-AzVMSku (or a compatible structure), which contains all necessary metadata.
 
    You can either approve the terms interactively or use the -Force switch to bypass prompts (useful in automation scenarios).
 
.PARAMETER VMSku
    The VM image metadata object. This must be a [pscustomobject] output from Get-AzVMSku or a compatible structure
    containing Publisher, Offer, Sku, Version, and Agreement details.
 
.PARAMETER Force
    Automatically accepts the agreement without prompting the user for consent. Useful for CI/CD or automation pipelines.
 
.PARAMETER PassThru
    Returns the modified object to the pipeline. Use this switch when calling Set-AzVMSku in a pipeline to ensure the updated [pscustomobject] is returned to the user.
 
.EXAMPLE
    Get-AzVMSku -Location "westeurope" -PublisherName "palo" | Set-AzVMSku
 
    Pipes a VM image definition directly into Set-AzVMSku and prompts the user to accept the Marketplace terms.
 
.EXAMPLE
    Get-AzVMSku -Location "eastus" -PublisherName "bitnami" -OfferName "wordpress" -NewestSKUs -NewestSKUsVersions -NoInteractive | Set-AzVMSku -Force
 
    A complete non-interactive command-chain, parsing the image object directly into Set-AzVmSku using a pipe
 
.EXAMPLE
    $image = Get-AzVMSku -NoInteractive -Location "southcentralus" -PublisherName "cisco" -NewestSKUs -NewestSKUsVersions -OfferName "cisco-ccv" | Set-AzVMSku -Force -PassThru
 
    Stores the image object in a variable and then accepts the agreement with -Force (Without -PassThru, the variable $image will be empty).
 
.NOTES
    - Only necessary for images that require a Marketplace agreement.
    - You must be logged in with an account that has access to the target subscription.
    - If the image is not available under your current Azure subscription, the command will fail.
    - For automation, always use the -Force switch to avoid prompts.
 
.LINK
    https://github.com/ChristofferWin/codeterraform
 
.LINK
    https://codeterraform.com/blog
#>
function Set-AzVMSku {
    param(
    [Parameter(
        ValueFromPipeline = $true
    )]
    [object]$VMSku,
    [switch]$Force,
    [switch]$PassThru
    )

    if($VMSku -in $null){
        Write-Error "The input-object is null, please provide an object of type [pscustomobject]"
        return
    } elseif($VMSku.Agreement -in $null){
        Write-Warning "The input-object is does not contain an agreement, therefor nothing to do..."
        if($PassThru){
            $VMSku.Agreement = $ReturnTerms
            return $VMSku
        }
        return 
    }
    try {
        $Agreement = Get-AzMarketplaceterms -OfferType 'virtualmachine' -Name $VMSku.Agreement.Name -Product $VMSku.Agreement.Product -Publisher $VMSku.Agreement.Publisher -ErrorAction Stop
    } catch{
        Write-Error "The URN: $($VMSku.URN) provided cannot be used on the current SubscriptionID: $($VMSku.SubscriptionID) due to the image being restricted by the publisher"
        Write-Warning "Please either change the context (Via Connect-AzAccount OR Set-AzAdvancedContext) to a subscription with access or run the command again with another image..."
        return
    }

    if($Force){
        Write-Verbose "The 'Force' Parameter is used, therefore the agreement for URN: $($VMSku.URN) will be auto-accepted"
    } else {
        Write-Warning "All terms specified within the agreement for URN: $($VMSku.URN)"
        $Answer = (Read-Host "`n`n[Privacy Policy => $($Agreement.PrivacyPolicyLink)]`n`n[License => $($Agreement.LicenseTextLink)]`n`n[Marketplace Terms => $($Agreement.MarketplaceTermsLink)]`n`nDo you accept these terms? [y(yes)/n(no)]").Trim().ToLower()
        Write-Host "ANSWER: $($Answer)"
        if($Answer -ne "y" -and $Answer -ne "yes"){
            Write-Warning "Operation stopped - Agreement has been declined by the user..."
            return
        }
    }

    try {
       $ReturnTerms = Set-AzMarketplaceterms -SubscriptionId $VMSku.SubscriptionID -Product $VMSku.Offer -Publisher $VMSku.Publisher -Name $VMSku.Sku -Accept -Confirm:$false -Verbose -ErrorAction Stop
    }catch{
        return $_
    }
    Write-Host -ForegroundColor Green "The agreement for URN: $($VMSku.URN) has been accepted and the image can be used in deployments"
    if($PassThru){
        $VMSku.Agreement = $ReturnTerms
        return $VMSku
    }
}
Export-ModuleMember Get-AzVMSku, Set-AzVMSku