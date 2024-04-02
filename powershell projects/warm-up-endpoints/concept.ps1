$env:URLObjects = "https://bt.dk"

$env:i = 0
$env:Aggresive = $true
$env:Path = "C:\Users\Christoffer Windahl\Desktop\for blog posts\codeterraform\powershell projects\warm-up-endpoints"
$env:ReportExtension = "json"
$env:CreateReport = $true

(Start-Process pwsh -PassThru -ErrorAction Stop -NoNewWindow -ArgumentList "-Command {
                    $timer = $([Diagnostics.Stopwatch]::StartNew())
                    $URL = $env:URLObjects
                    $Date = (Get-Date).ToString('dd-MM-yyyy'))
                    $Name = if($URL -like 'www.*'){$URL.Split('.')[1]}elseif(($URL.Substring(7).Split('/') | ? {'' -notin $_}).Count -gt 1){'$URL.Substring(7).Replace('/', '').Split('.')'.Replace(' com', '-')}else{$URL.Substring(7).Replace('/', '').Split('.')[0]}
                    $Path = 'Log-$Name-at-$Date.$env:Path'
                    $ReturnObject = @()
                    
                    if($env:CreateReport){
                        cd $env:Path
                    }
                    while(`$true){
                        try{
                            $WebCall = Invoke-WebRequest -UseBasicParsing -Uri $URL -ErrorAction Stop
                            $ResponseTime = Measure-Command -Expression {Invoke-WebRequest -UseBasicParsing -URI $URL}).Milliseconds
                        }
                        catch{
                            $_
                        }
                    $ReturnObject = [PSCustomObject]@{
                            Time = '{0:hh\:mm\:ss}' -f $timer.Elapsed
                            URL = $URL
                            ThreadNumber = $env:i
                            HTTPStatus = $WebCall.StatusCode
                            ResponseTimeInMS = $ResponseTime
                        }
                        if($env:Aggresive){
                            Start-Sleep -Seconds 1
                        }
                        else{
                            Start-Sleep -Seconds 30
                        }
                        try{
                            
                            if($env:CreateReport){
                                switch($env:ReportExtension){
                                    'json' {
                                        $OutputArray = @()
                                        $ReturnObject | ConvertTo-Json | Out-File -Path $Path
                                        
                                    }
                                   # 'yaml' {$ReturnObject | ConvertTo- -Delimiter ',' -NoTypeInformation | Out-File -Force -Append -FilePath $Path -ErrorAction Stop}
                                    'csv' {$ReturnObject | ConvertTo-Csv -Delimiter ',' -NoTypeInformation | Out-File -Force -Append -FilePath $Path -ErrorAction Stop}
                                }
                            }
                        }
                        catch{
                            $_
                        }
                        $Error.Clear()
                    }
}").id