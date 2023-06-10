function Get-AzVMSizes {
    [cmdletBinding(DefaultParameterSetName = 'ManualSettings')]
    param(
        [Parameter(ParameterSetName = "DefaultSettings", Mandatory = $true)][switch]$UseDefault,
        [Parameter(ParameterSetName = "ManualSettings", Mandatory = $true)][string]$Location,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$ContinueOnError,
        [Parameter(ParameterSetName = "ManualSettings")][int]$Top,
        [Parameter(ParameterSetName = "ManualSettings")][string]$Pattern,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$NoInteractive,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$TerminalOutput,
        [Parameter(ParameterSetName = "ManualSettings")][switch]$FileOutput,
        [Parameter(ParameterSetName = "ManualSettings")][string]$FilePath
    )

    $SubscriptionID = (Get-AzContext).Subscription
    if(!$SubscriptionID){
        Write-Error "No Azure context found. Please use either Login-AzAccount or Set-AzAdvancedContext to get one..."
    }

    try{
        $Locations = Get-AzLocation -ErrorAction Stop
    }
    catch{
        Write-Error "An error occured while trying to validate the location: $Location`n$_"
    }
    
    foreach($Location in $Locations){
        
    }
    
    if($UseDefault){
        $Location = "West Europe"
        $Top = 0 # 0 = all
        $Pattern = "none"
        $TerminalOutput = $true
        $OutputFileExtension = ".json"
    }

    #Map location correctly - All locations must be without space otherwise it will fail
    try{
        Get-AzQuota -Scope $URI
    }


    try{
        Register-AzResourceProvider -ProviderNamespace "Microsoft.Quota" -ErrorAction Stop
    }
    catch{
        Write-Error "The following error occured while trying to register required providers...:`n$_"
    }
    do{
        try{
            $AllQuotas = Get-AzQuota -Scope "/subscriptions/$SubscriptionID/providers/Microsoft.Compute/locations/$Location" -ErrorAction Stop
        }
        catch{
            if($_.Exception.Message -like "*The request was throttled*"){
                Write-Warning "API sent a timeout of 300 seconds..."
                Write-Warning "Trying again in 300 seconds..."
                for($i = 1; $i -le 10; $i++){
                    Start-Sleep -Seconds 30
                    Write-Verbose "$(-1*($i * 30 - 300)) seconds left..."
                    Continue
                }
            }
             Write-Verbose "Provider call failed... Trying again in 5 seconds..."
             Start-Sleep -Seconds 5
        }
    }
    while(!$AllQuotas)
    return $AllQuotas
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
    Write-Verbose "User: $((Get-AzContext).Account) successfully logged in to Azure tenant: $TenantID"
    return
}

function Get-RequiredModules {

}
Export-ModuleMember Get-AzAdvancedContext, Get-AzVMSizes