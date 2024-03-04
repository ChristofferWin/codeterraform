$FinalPaths = $null
$env:ALL_CHANGED_FILES = "C:\Users\Christoffer Windahl\Desktop\for blog posts\codeterraform\terraform projects\modules\test modules\vm-bundle\unit_test.tftest.hcl"
foreach($Path in $($env:ALL_CHANGED_FILES)){
    if($Path -like "*.tftest.hcl" -or $Path -like "*\modules\test*.tf"){
      $FinalPaths += (Get-ChildItem -Path $Path).DirectoryName
    }
    elseif($Path -like "*.tf"){
      $Files = Get-ChildItem -Path ".\terraform projects\modules\test modules" -Recurse | ? {$_.Name -like "*.tf"}
      foreach($File in $Files){
        $Content = Get-Content -Path $($File.FullName)
        foreach($Line in $Content){
          if($Line -like "*source*$($Path.Split("\")[-2])*"){
          $FinalPaths += $($File.DirectoryName)
        }
      }
    }
  }}
  $FinalPaths = $FinalPaths | Select-Object -Unique
  return $FinalPaths