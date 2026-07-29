function Calculate-ModuleRuntime {
  param (
    [Parameter(Mandatory)][string]$CommandToExecute,
    [string]$ModulePath = "C:\Users\Chris\OneDrive\Desktop\codeterraform\powershell projects\modules\Get-AzVMSku",
    [int]$Iterations = 10,
    [int]$Parallelism = 5
  )
  $ModuleName = $CommandToExecute.Split(" ")[0]
  Write-Verbose "Running command $CommandName $Iterations times"

  $ArrayOfResults = 1..$Iterations | Foreach-Object -ThrottleLimit $Parallelism -Parallel {
    Import-Module $USING:ModulePath  
    $scriptBlock = [scriptblock]::Create($USING:CommandToExecute)
    (Measure-Command $scriptBlock).TotalSeconds

  } | Sort-Object

  return [PSCustomObject]@{
    ModuleName = $ModuleName
    ModuleVersion = ((Get-Content (Join-Path $ModulePath "$($ModuleName).psd1") | ? {$_ -like "*ModuleVersion*"}).Split(" ")[-2]).Trim().Replace("'", "")
    CommandArguments = (($CommandToExecute.Split(" ") | Select -Skip 1) -join " ").ToLower() #Signature
    Iterations = $Iterations
    ExecutionTimeSeconds = [PSCustomObject]@{
      Average  = ($ArrayOfResults | Measure-Object -Sum).Sum / $Iterations
      Fastest = $ArrayOfResults[0]
      Slowest = $ArrayOfResults[-1]
    }
  } 
}

Export-ModuleMember Calculate-ModuleRunTime