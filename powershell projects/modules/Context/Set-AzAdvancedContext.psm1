<#
.SYNOPSIS
    Sets the Azure context using advanced authentication methods.

.DESCRIPTION
    The Set-AzAdvancedContext function allows users to set the Azure context using advanced authentication methods, including specifying credentials and tenant/subscription IDs. It provides interactive prompts and error handling for a seamless authentication experience.

.PARAMETER Credential
    Specifies a PSCredential object containing Azure authentication details. Mandatory parameter when using AzureEnvironment parameter set.

.PARAMETER TenantID
    Specifies the Azure Active Directory tenant ID. Mandatory parameter when using AzureEnvironment parameter set.

.PARAMETER SubscriptionID
    Specifies the Azure subscription ID. Mandatory parameter when using AzureEnvironment parameter set.

.PARAMETER ContinueOnError
    Specifies whether the function should continue processing in case of errors. By default, it stops on errors.

.PARAMETER NoInteractive
    Specifies whether to suppress interactive prompts.

.OUTPUTS
    None - Using the -Verbose switch will cause information to be sent to the standard output

.NOTES
    File Name      : Set-AzAdvancedContext.psm1
    Author         : Christoffer Windahl Madsen
    Prerequisite    : This function requires the Az PowerShell module.

.LINK
    https://github.com/ChristofferWin/codeterraform

.EXAMPLE
    //Simply log in using all information => The credential object can either contain SPN ID + secret or username + password
    //If a user account is provided and it requires MFA, a pop-up will show
    Set-AzAdvancedContext -Credential (Get-Credential) -TenantID "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX" -SubscriptionID "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX"

.EXAMPLE
    //Using a SPN ID + secret to login and also making sure the call is non-interractive as it must be used in an automatic proces
    Set-AzAdvancedContext -Credential (Get-Credential) -TenantID "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX" -SubscriptionID "XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX" -NoInteractive

.EXAMPLE
    //Calling the function without any input causing it to automically search for an active session, if none is found a credential is requested
    Set-AzAdvancedContext

.EXAMPLE
    //Calling the function without any input and also forcing it to be non-interractive causing it to throw an error if no context can be automically found
    Set-AzAdvancedContext -NoInteractive
#>
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
        try{
            $Credential = Get-Credential -ErrorAction Stop
        }
        catch{
            Write-Warning "Information for the credential object is lacking, function stopping..."
            return
        }
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
    Write-Verbose "User: $($Context.Account) successfully logged in to Azure tenant: $($Context.Tenant.ID) and subscription: $($Context.Subscription.Id)"
    return
}
Export-ModuleMember Set-AzAdvancedContext