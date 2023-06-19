function Get-AzVMSizes {
    [cmdletBinding(DefaultParameterSetName = 'ManualSettings')]
    param(
        [Parameter(ParameterSetName = "ManualSettings", Mandatory = $true)][string]$Location,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$ContinueOnError,
        [Parameter(ParameterSetName = "ManualSettings", Mandatory = $true)][string]$OperatingSystem,
        [Parameter(ParameterSetName = "ManualSettings")][string]$OSVersion,
        [Parameter(ParameterSetName = "ManualSettings")][string]$VMPattern,
        [Parameter(ParameterSetName = "ManualSettings")][string]$OSPattern,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$NoInteractive,
        [Parameter(ParameterSetName = "ShowCommandLocations")][switch]$ShowLocations,
        [Parameter(ParameterSetName = "ShowCommandVMs")][switch]$ShowVMCategories,
        [Parameter(ParameterSetName = "ShowCommandVMsOS")][switch]$ShowVMOperatingSystems
    )   

    $MSVMURL = "https://azure.microsoft.com/en-us/pricing/details/virtual-machines/series/"
    $CategoryObjects = @()
    $LocationObjects = @()
    $TotalObjects = @()

    $PredefinedImageDefinitions = @(
        [pscustomobject]@{
            Publisher = "MicrosoftWindowsServer"
            Offer = "WindowsServer"
            SKU = "" #All WindowsServer SKU's share the same offer and publisher
            Alias = "Server2008, Server2012, Server2012R2, Server2016, Server2019, Server2022"
        },
        [PSCustomObject]@{
            Publisher = "MicrosoftWindowsDesktop"
            Offer = "Windows-7"
            SKU = "win7*$OSPattern*"
            Alias = "Windows7"
        },
        [pscustomobject]@{
            Publisher = "MicrosoftWindowsDesktop"
            Offer = "Windows-10"
            SKU = "win10*$OSPattern*"
            Alias = "Windows10"
        },
        [pscustomobject]@{
            Publisher = "MicrosoftWindowsDesktop"
            Offer = "Windows-11"
            SKU = "win11*$OSPattern*"
            Alias = "Windows11"
        },
        [pscustomobject]@{
            Publisher = "OpenLogic"
            Offer = "CentOS"
            SKU = "*$OSPattern*"
            Alias = "CentOS"
        },
        [pscustomobject]@{
            Publisher = "Canonical"
            Offer = "UbuntuServer"
            SKU = "*$OSPattern*"
            Alias = "Ubuntu"
        },
        [pscustomobject]@{
            Publisher = "Debian"
            Offer = "Debian-10"
            SKU = "*$OSPattern*"
            Alias = "Debian10"
        },
        [pscustomobject]@{
            Publisher = "Debian"
            Offer = "Debian-11"
            SKU = "*$OSPattern*"
            Alias = "Debian11"
        },
        [pscustomobject]@{
            Publisher = "Redhat"
            Offer = "rhel"
            SKU = "*$OSPattern*"
            Alias = "Redhat"
        }
    )

    $AliasArray = @()
    $AliasArray += $PredefinedImageDefinitions[0].Alias.Split(",")
    $AliasArray += $PredefinedImageDefinitions[1..8].Alias
    $AliasArray = $AliasArray | % {$_.Trim().ToLower()}

    $SubscriptionID = (Get-AzContext).Subscription
    if(!$SubscriptionID){
        Write-Error "No Azure context found. Please use either Login-AzAccount or Set-AzAdvancedContext to get one..."
        return
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
            $CoresAvailable = Get-AzVMUsage -Location $Location -ErrorAction Stop
        }
        catch{
            if(!$ContinueOnError){}
        }
    }
    while(!$CoresAvailable)

    do{
        try{
            if($VMPattern){
               $VMsizes = Get-AzVMSize -Location $Location -ErrorAction Stop | ? {$_.Name -like "Standard_*$VMPattern*" -or $_.Name -like "Basic_*$VMPattern*"}
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
            Break
        }
        catch{
            Write-Error "The following error occured while trying to retrieve vm sizes...:`n$_"
            return
        }
    }
    while(!$VMsizes)

    switch($OperatingSystem){
        "Server2008" {$PredefinedImageDefinitions[0].Sku = "2008-*$OSPattern*"; $ImageDefinition = $PredefinedImageDefinitions[0]}
        "Server2012" {$PredefinedImageDefinitions[0].Sku = "2012-data*$OSPattern*"; $ImageDefinition = $PredefinedImageDefinitions[0]}
        "Server2012R2" {$PredefinedImageDefinitions[0].Sku = "2012-r2*$OSPattern*"; $ImageDefinition = $PredefinedImageDefinitions[0]}
        "Server2016" {$PredefinedImageDefinitions[0].Sku = "2016*$OSPattern*"; $ImageDefinition = $PredefinedImageDefinitions[0]}
        "Server2019" {$PredefinedImageDefinitions[0].Sku = "2019*$OSPattern*"; $ImageDefinition = $PredefinedImageDefinitions[0]}
        "Server2022" {$PredefinedImageDefinitions[0].Sku = "2022*$OSPattern*"; $ImageDefinition = $PredefinedImageDefinitions[0]}
        "Windows7" {$ImageDefinition = $PredefinedImageDefinitions[1]}
        "Windows10" {$ImageDefinition = $PredefinedImageDefinitions[2]}
        "Windows11" {$ImageDefinition = $PredefinedImageDefinitions[3]}
        "CentOS" {$ImageDefinition = $PredefinedImageDefinitions[4]}
        "Ubuntu" {$ImageDefinition = $PredefinedImageDefinitions[5]}
        "Debian10" {$ImageDefinition = $PredefinedImageDefinitions[6]}
        "Debian11" {$ImageDefinition = $PredefinedImageDefinitions[7]}
        "Redhat" {$ImageDefinition = $PredefinedImageDefinitions[8]}
    }
 
    try{
        $ImageDefinition.SKU = (Get-AzVMImageSku -Location $Location -PublisherName $ImageDefinition.Publisher -Offer $ImageDefinition.Offer -ErrorAction Stop | ? {$_.SKUs -like $ImageDefinition.SKU}).Skus
    }
    catch{
        Write-Error "The following error occured while trying to retrieve the current SKUs for OS: $OperatingSystem`n$_"
        return
    }

    #Please check whether a pattern of null results in 0 skus being found or simply that every single possible SKU for a given OS is found
    if($ImageDefinition.SKU.Count -eq 0){
        Write-Error "No SKUs were found for Operating system: $OperatingSystem and using OSPattern: $OSPattern`nTry to change the pattern or simply ommit it..."
        return
    }

    foreach($SKU in $ImageDefinition.SKU){
        $Temp = @()
        try{
           $Temp += [pscustomobject]@{
            SKU = $SKU 
            Versions = (Get-AzVMImage -Location $Location -PublisherName $ImageDefinition.Publisher -Offer $ImageDefinition.Offer -Sku $SKU).Version
            }
        }
        catch{

        }
}
    return $VMsizes
}

function Set-AzAdvancedContext {
    [cmdletBinding(DefaultParameterSetName = 'ManualSettings')]
    param(
        [Parameter(ParameterSetName = "ManualSettings")][switch]$ContinueOnError,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$NoInteractive,
        [Parameter(ParameterSetName = "AzureEnvironment", Mandatory = $true)][pscredential]$Credential,
        [Parameter(ParameterSetName = "AzureEnvironment", Mandatory = $true)][ValidatePattern('(\{|\()?[A-Za-z0-9]{4}([A-Za-z0-9]{4}\-?){4}[A-Za-z0-9]{12}(\}|\()?')][string]$TenantID,
        [Parameter(ParameterSetName = "AzureEnvironment", Mandatory = $true)][ValidatePattern('(\{|\()?[A-Za-z0-9]{4}([A-Za-z0-9]{4}\-?){4}[A-Za-z0-9]{12}(\}|\()?')][string]$SubscriptionID
    )
    $AlreadyLoggedIn = (Get-AzContext) -notin $null
    if($Credential.Count -eq 0 -and !$NoInteractive -and !$AlreadyLoggedIn){
        Write-Warning "No context could be found. Please provide a credential object... "
        Write-Output "Either username/password or app id/secret"
        $Credential = Get-Credential
    }
    elseif($AlreadyLoggedIn){
        #Simply continue
    }
    elseif($Credential.Count -gt 0 -and !$NoInteractive){
        #Simply continue
    }
    else{
        Write-Verbose "No Azure context found and no credential is provided..."
        Write-Error "Cannot ask for credential due to flag 'NoInteractive'"
        break
    }
    do{
        if(!$TenantID -and !$AlreadyLoggedIn -and !$NoInteractive){
            try{
                Login-AzAccount -ErrorAction Stop
                $AlreadyLoggedIn = $true
                Continue
            }
            catch{
                if(!$ContinueOnError){
                    Write-Error "Threw an error: $_"
                    break
                }
                Write-Warning "The interactive login failed... retrying..."
            }
        }
        elseif($TenantID -and $SubscriptionID -and !$AlreadyLoggedIn){
            try{
                Login-AzAccount -Tenant $TenantID -Subscription $SubscriptionID -Credential $Credential -ErrorAction Stop
                $AlreadyLoggedIn = $true
                Continue
            }
            catch{
                if($NoInteractive -or !$ContinueOnError){
                    if($_.Exception.Message -like "*ROPC does not support MSA accounts*" -and $NoInteractive){
                        Write-Error "The account: $($Credential.Username) must use interactive authentication..."
                    }
                    elseif($_.Exception.Message -like "*validating credentials due to invalid username or password*" -or $_.Message -like "*password is expired*" -or $_.Message -like "*user account is disabled*" -or $_.Message -like "*does not have access to subscription*" -or $_.Message -like "*must use multi-factor authentication*"){
                        Write-Error "Username or password is incorrect for tenant: $TenantID"
                    }
                    else{
                        Write-Error "An error occured while trying to login using the provided credential:`n$_"
                    }
                }
                else{
                    if($_.Exception.Message -like "*Tenant* not found*"){
                        Write-Warning "The Azure Tenant provided: $TenantID was not found..."
                        do{
                            try{
                                $TenantID = Read-Host "Please provide a valid TenantID..." -ErrorAction Stop
                                $OK = $true
                            }
                            catch{
                                Write-Warning "The TenantID provided is not a valid GUID..."
                            }
                        }
                       while(!$OK)
                    }
                    $OK = $false
                    $CredentialOK = $false
                    if($_.Exception.Message -like "*validating credentials due to invalid username or password*" -or $_.Exception.Message -like "*password is expired*" -or $_.Exception.Message -like "*user account is disabled*" -or $_.Exception.Message -like "*must use multi-factor authentication*" -or $_.Exception.Message -like "* Unsupported User Type*" -and !$CredentialOK){
                        Write-Warning "The following error occured while trying to login:`n$_"
                        $Credential = Get-Credential
                    }
                    else{
                        $CredentialOK = $true
                    }
                    try{
                        Login-AzAccount -Tenant $TenantID -Subscription $SubscriptionID -Credential $Credential -ErrorAction Stop -WarningVariable Warnings 3>$null
                    }
                    catch{
                        if($Warnings.Message -like "*The subscription*could not be found*"){
                            Write-Warning "The subscription: $SubscriptionID could not be found in Azure..."
                        }
                        do{
                            try{
                                $SubscriptionID = Read-Host "Please provide a valid SubscriptionID..." -ErrorAction Stop
                                $OK = $true
                            }
                            catch{
                                Write-Warning "The SubscriptionID provided is not a valid GUID..."
                            }
                        }
                        while(!$OK)
                        if($Warnings.Message -like "*does not have authorization to perform action 'Microsoft.Resources/subscriptions/read'*"){
                            if(!$ContinueOnError){
                                Write-Error "The user does not have read access to the subscription: $SubscriptionID..."
                                break
                            }
                            $Warnings = ""
                            Write-Verbose "Running through 10 cycels of 30 seconds each for a total of 5minutes..."
                            for($i = 1; $i -le 10; $i++){
                                Write-Warning "Go to the Azure subscription: $SubscriptionID and provide a minimum of reader for the user: $($Credential.Username)"
                                Start-Sleep -Seconds 30
                                try{
                                    Login-AzAccount -Tenant $TenantID -Subscription $SubscriptionID -Credential $Credential -ErrorAction Stop -WarningVariable Warnings 3>$null
                                }
                                catch{
                                    if($Warnings.Message -like "*does not have authorization to perform action 'Microsoft.Resources/subscriptions/read'*" -and $i -ne 10){
                                        Write-Verbose "Cycle: $i / 10 - $(($i * 30)/60) minutes gone..."
                                        Write-Warning "The user: $($Credential.Username) does still not have the minimum role on the subscription: $SubscriptionID"
                                    }
                                    else{
                                        Write-Error "The following error occured while trying to verify whether the user: $($Credential.Username) has access to subscription: $SubcriptionID, error:`n$_"
                                        break
                                    }
                                }
                            }
                        }
                    }
                }   
            }    
        } 
    }
    while(!$AlreadyLoggedIn)
    $Context = Get-AzContext
    Write-Verbose "User: $($Context.Account) successfully logged in to Azure tenant: $($Context.Tenant.ID)"
    return
}
<#
function Get-RequiredModules {

}
Export-ModuleMember Get-AzAdvancedContext, Get-AzVMSizes

#>


