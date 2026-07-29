param (
    [string]$RuntimeTestPath = "./Get-AzVmSku.tests.json"
)
Remove-Module Calculate-ModuleRunTime
Import-Module "C:\Users\Chris\OneDrive\Desktop\codeterraform\powershell projects\modules\Get-AzVMSku\Calculate-ModuleRunTime.psm1"

$TestsRaw = Get-Content $RuntimeTestPath | ConvertFrom-Json -Depth 50
$TestResults = @()
foreach($Test in $TestsRaw) {
    if($null -in $Test.execution -or $null -in $Test.count){
        Write-Warning "Skipping execution on test list element $i due to malformed structure"
        Write-Host "Use structure attributes: 'execution' & 'count'"
        continue
    }
    $TestResults += Calculate-ModuleRuntime -CommandToExecute $Test.execution -Iterations $Test.count
}