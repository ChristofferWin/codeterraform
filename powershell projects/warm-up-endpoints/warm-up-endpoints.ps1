function Run-URLWarmUp {
    [System.Collections.ArrayList]$URLObjects = @()
    [array]$Jobs = @()
    [int]$ThreadCount = $URLObjects.Count * $ThreadMultiplier
    foreach($URL in $URLS){
        if($URL -notmatch "^(https?://|www\.)[^\s/$.?#]+\.[^\s]*$"){
            $URLObjects += [PSCustomObject]@{
                URLValidationText = "URL: $URL is NOT correctly formatted`n"
                URLValidationStatus = $false
            } 
        }
        else{
            $URLObjects += [PSCustomObject]@{
                URLValidationText = "URL: $URL is OK`n"
                URLValidationStatus = $true
                URL = $URL
            }
        }
    }

    for($i = 0; $i -le $URLObjects.Count -1; $i++){
        if($URLObjects[$i].URLValidationStatus){
            try{
                Invoke-WebRequest -UseBasicParsing -Uri $URLObjects[$i].URL -ErrorAction Stop | Out-Null
                Write-Verbose "URL[$i] format -> $($URLObjects[$i].URLValidationText) -> OK"
            }
            catch{
                Write-Warning "Removing URL $($URLObjects[$i].$URL) due to the following error...`n$_"
                $URLObjects.RemoveAt($i) > $null
            }
        }
        else {
            Write-Verbose "URL[$i] format -> $($URLObjects[$i].URLValidationText) -> NOT OK"
            Write-Warning "Removing URL $($URLObjects[$i].URLValidationText)..."
            $URLObjects.RemoveAt($i) > $null
        }
    }

    if(!$Path -and $CreateReport){
        Write-Warning "Creating log files on default location: $(if($PSVersionTable.Edition -eq "Desktop"){"$HOME\Documents"}else{"$HOME"})"
    }
    
    Write-Verbose "Script will create $($URLObjects.Count) * $($ThreadMultiplier) thread(s) per OK URL = $ThreadCount"
    $ReturnObject = @()
    
    if($CreateReport){
        foreach($URLObject in $URLObjects){
            $URL = $URLObject.URL
            $Name = if($URL -like "www.*"){$URL.Split(".")[1]}elseif(($URL.Substring(7).Split("/") | ? {"" -notin $_}).Count -gt 1){"$($URL.Substring(7).Replace('/', '').Split('.'))".Replace(" com", "-")}else{$URL.Substring(7).Replace("/", "").Split(".")[0]}
            switch($ReportExtension){
                "json" {"[`n`n]" | Out-File "Log-$Name-at-$((Get-Date).ToString("dd-MM-yyyy")).json"}
                "yaml" {}
                "csv" {}
            }
        }
    }
    $Jobs = @()
    foreach($URLObject in $URLObjects){
        for($i = 0; $i -le $ThreadMultiplier -1; $i++){
            try{
                $Jobs += Start-Process pwsh -WindowStyle Hidden -ArgumentList  -{
                    $timer = [Diagnostics.Stopwatch]::StartNew()
                    $URL = $args[0]
                    $Date = (Get-Date).ToString("dd-MM-yyyy")
                    $Name = if($URL -like "www.*"){$URL.Split(".")[1]}elseif(($URL.Substring(7).Split("/") | ? {"" -notin $_}).Count -gt 1){"$($URL.Substring(7).Replace('/', '').Split('.'))".Replace(" com", "-")}else{$URL.Substring(7).Replace("/", "").Split(".")[0]}
                    $Path = "Log-$Name-at-$Date.$($args[4])"
                    $ReturnObject = @()
                    
                    if($args[3]){
                        cd $args[3]
                    }
                    while($true){
                        try{
                            $WebCall = Invoke-WebRequest -UseBasicParsing -Uri $URL -ErrorAction Stop
                            $ResponseTime = $((Measure-Command -Expression {Invoke-WebRequest -UseBasicParsing -URI $URL}).Milliseconds)
                        }
                        catch{
                            $_
                        }
                    $ReturnObject = [PSCustomObject]@{
                            Time = '{0:hh\:mm\:ss}' -f $timer.Elapsed
                            URL = $args[0]
                            ThreadNumber = $args[1]
                            HTTPStatus = $WebCall.StatusCode
                            ResponseTimeInMS = $ResponseTime
                        }
                        if($args[2]){
                            Start-Sleep -Seconds 1
                        }
                        else{
                            Start-Sleep -Seconds 30
                        }
                        try{
                            
                            if($args[5]){
                                switch($args[4]){
                                    "json" {
                                        $OutputArray = @()
                                        
                                        
                                    }
                                   # "yaml" {$ReturnObject | ConvertTo- -Delimiter "," -NoTypeInformation | Out-File -Force -Append -FilePath $Path -ErrorAction Stop}
                                    "csv" {$ReturnObject | ConvertTo-Csv -Delimiter "," -NoTypeInformation | Out-File -Force -Append -FilePath $Path -ErrorAction Stop}
                                }
                            }
                        }
                        catch{
                            $_
                        }
                        $Error.Clear()
                    }
                }
                foreach($Job in $Jobs){
                    $IDs += (Start-Process pwsh -ArgumentList "", "-Command", $Job -Verbose).id
                }
                Write-Host "HELLO WORLD"
            }
            catch{
                Write-Warning "The following error occured inside of job $($Jobs[$i].Name)...`n$_"
            }
        }
    }
    Write-Host "STOP"
}

function Show-Threads {
    
    [System.Collections.ArrayList]$JobObjects = @()

    try{
        $Jobs += Get-Job -ErrorAction Stop | ? {$_.Command.ToString() -like '*$ResponseTime = $((Measure-Command*' -and $_.State -eq "Running"}
    }
    catch{
        Write-Error "Something happened while trying to retrieve the running background jobs...`n$_"
    }

    foreach($Job in $Jobs){
        $JobObjects += [PSCustomObject]@{
        }
    }

    return $JobObjects
}

function Main {
    [CmdletBinding(DefaultParameterSetName = "Run")]
    param(
        [parameter(Mandatory = $true, ParameterSetName = "Run")][array]$URLS,
        [parameter(Mandatory = $false, ParameterSetName = "Run")][switch]$Aggressive,
        [parameter(Mandatory = $false, ParameterSetName = "Run")][string]$Path,
        [parameter(Mandatory = $false, ParameterSetName = "Run")][switch]$CreateReport,
        [parameter(Mandatory = $false, ParameterSetName = "Run")][ValidateSet("json", "csv", "yaml")][string]$ReportExtension = "json",
        [parameter(Mandatory = $false, ParameterSetName = "Run")][int]$ThreadMultiplier = 1,
        [parameter(Mandatory = $true, ParameterSetName = "Show")][switch]$ShowThreads,
        [parameter(Mandatory = $true, ParameterSetName = "Kill")][switch]$KillThreads
    )

     # Determine the activated parameter set
     $activatedSet = "Run"

     # Switch based on the activated parameter set
     switch ($activatedSet) {
         "Run" {
            Run-URLWarmUp
         }
 
         "Show" {
            Show-Threads
         }
         "Kill" {
            Kill-Threads
         }
 
         "Report" {
            New-Report
         }
     }    
}

Main -Verbose -URLS @("https://bt.dk", "https://ekstrabladet.dk") -Path "C:\Users\Christoffer Windahl\Desktop\for blog posts\codeterraform\powershell projects\warm-up-endpoints" -Aggressive -ThreadMultiplier 2 -CreateReport -ReportExtension "json"
$Test = Show-Threads
<#

LOOK INTO THIS AS MEMORY CONSUMPTION ONLY INCREASES FOR JOBS OVER TIME:
When you create a remote job, two jobs are created - one for the host and a child job (also on the host) for the remote job. When I used receive-job on the parent, I expected this to clear out all output streams (parent and child). It turned out that the child job still had a fully populated field $childJob.output.

I ended up using receive-job on the child job, and then immediately cleared its output using $childJob.output.clear().

In my tests, this didn't have any adverse affects - but, I wouldn't completely trust this method for more critical tasks without better testing.

After I did this, the memory consumption problem was resolved.

#>