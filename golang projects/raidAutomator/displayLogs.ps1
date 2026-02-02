param(
    [Parameter()]
    [ValidateSet("error", "information")]
    [string]$logType = "information",
    [switch]$clear = $false,

    [int]$intervalInSeconds = 5

)

$filePath = "C:\Users\Chris\OneDrive\Desktop\codeterraform\golang projects\raidAutomator"
$filePaths = Get-ChildItem -Path $filePath | ? {$_.BaseName -eq "error_log" -or $_.BaseName -eq "information_log"}

foreach($log in $filePaths) {
    if ($log.BaseName -eq "$($logType)_log") {
        $logToRead = $log.FullName
    }
}

while($true){
    $format = "MMMM dd, yyyy HH:mm:ss"
    $dateNowBefore  = [DateTime]::ParseExact((Get-Date -Format $format), $format, $null)
    $fileSizeNow = (Get-Content $logToRead).Length
    Start-Sleep -Seconds $intervalInSeconds
    $fileSizeAfter = (Get-Content -Path $logToRead).Length
    if ($fileSizeAfter -gt $fileSizeNow) {
        $entriesInCurrentScope = @()
        $logContent = Get-Content -Path $logToRead -Raw | ConvertFrom-Json -Depth 50
        for($x = $logContent.Length -1; $x -ge 0; $x--){
            if([DateTime]::ParseExact(($logContent[$x].time_stamp), $format, [System.Globalization.CultureInfo]::InvariantCulture) -gt $dateNowBefore){
                $entriesInCurrentScope += $logContent[$x]
            }
        }
        foreach ($log in $entriesInCurrentScope) {
            $type = $logType.Substring(0,1).ToUpper() + $logType.Substring(1)

            $details =
                if ($log.error) {
                    "Error     : $($log.error)"
                }
                elseif ($log.action) {
                    "Action    : $($log.action)"
                }

            $output = @"
        ────────────────────────────────────────
        $type
        $details
        Message   : $($log.message)
        Timestamp : $($log.time_stamp)
        ────────────────────────────────────────
"@

    Write-Host $output
}
    }
       
       #$logContent = Get-Content -Path $logToRead | ConvertFrom-Json -Depth 50
}