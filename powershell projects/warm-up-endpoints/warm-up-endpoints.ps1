function Run-URLWarmUp {
    [System.Collections.ArrayList]$URLObjects = @()
    [array]$Jobs = @()
    [int]$ThreadCount = 0

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
    
    $ThreadCount = $URLObjects.Count
    Write-Verbose "Script will create 1 thread per OK URL = $ThreadCount"
    
    for($i = 0; $i -le $ThreadCount -1; $i++){
        try{
            $Jobs += Start-Job -ScriptBlock {
                $timer = [Diagnostics.Stopwatch]::StartNew()
                $URL = $args[0]
                $Date = (Get-Date).ToString("dd-MM-yyyy")
                $Name = if($args[0] -like "www.*"){$URL.Split(".")[1]}else{($URL.Substring(7).Replace("/", "").Split(".")[0])}
                $Path = "Log-$Name-at-$Date.$($args[4])"
                
                if($args[3]){
                    cd $args[3]
                }
                $ReturnObject = @()
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
                                "json" {$ReturnObject, "," | ConvertTo-Json -Depth 100 | ForEach-Object { $_ | Add-Content -Path $Path}}
                                "yaml" {$ReturnObject | ConvertTo-Csv -Delimiter "," -NoTypeInformation | Out-File -Force -Append -FilePath $Path -ErrorAction Stop}
                                "csv" {$ReturnObject | ConvertTo-Csv -Delimiter "," -NoTypeInformation | Out-File -Force -Append -FilePath $Path -ErrorAction Stop}
                            }
                        }
                    }
                    catch{
                        $_
                    }
                }
            } `
             -ErrorAction Stop -PSVersion 5.1 -ArgumentList $URLObjects[$i].URL, $i, $Aggressive, $Path, $ReportExtension, $CreateReport
        }
        catch{
            Write-Warning "The following error occured inside of job $($Jobs[$i].Name)...`n$_"
        }
    }
    return $Jobs
}

function Show-Threads {

}

function Main {
    [CmdletBinding(DefaultParameterSetName = "Run")]
    param(
        [parameter(Mandatory = $true, ParameterSetName = "Run")][array]$URLS,
        [parameter(Mandatory = $false, ParameterSetName = "Run")][switch]$Aggressive,
        [parameter(Mandatory = $false, ParameterSetName = "Run")][string]$Path,
        [parameter(Mandatory = $false, ParameterSetName = "Run")][switch]$CreateReport,
        [parameter(Mandatory = $false, ParameterSetName = "Run")][ValidateSet("json", "csv", "yaml")][string]$ReportExtension = "json",
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

Main -Verbose -URLS @("test.dk", "https://test.dk", "https://google.dk", "https://codeterraform.com", "https://codeterraform.com/blog", "www.dr.dk") -Path "C:\Users\Christoffer Windahl\Desktop\for blog posts\codeterraform\powershell projects\warm-up-endpoints" -Aggressive -CreateReport

<#

LOOK INTO THIS AS MEMORY CONSUMPTION ONLY INCREASES FOR JOBS OVER TIME:
When you create a remote job, two jobs are created - one for the host and a child job (also on the host) for the remote job. When I used receive-job on the parent, I expected this to clear out all output streams (parent and child). It turned out that the child job still had a fully populated field $childJob.output.

I ended up using receive-job on the child job, and then immediately cleared its output using $childJob.output.clear().

In my tests, this didn't have any adverse affects - but, I wouldn't completely trust this method for more critical tasks without better testing.

After I did this, the memory consumption problem was resolved.

#>