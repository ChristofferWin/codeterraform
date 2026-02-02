$pathRootFolder = wslpath "C:\Users\Chris\OneDrive\Desktop\codeterraform\golang projects\raidAutomator"
$ipOfVM = "135.116.212.127"
$usernameVM = "azureuser"
$pemName = "raidautomator2.pem"
cd $pathRootFolder
$env:GOOS = "linux"
$env:GOARCH = "amd64"
go build
$itemsToCopy = Get-ChildItem -Path $pathRootFolder -Recurse | ? {$_.Name -like "*.json" -or $_.Name -eq "RaidAutomator"}
ssh ssh -i ~/.ssh/$pemName "$usernameVM@$ipOfVM" "rm ./RaidAutomator"
foreach($file in $itemsToCopy){
    scp -i ~/.ssh/$pemName $file.FullName "$usernameVM@$ipOfVM`:./"
}
ssh -i ~/.ssh/$pemName "$usernameVM@$ipOfVM" "chmod +x ~/RaidAutomator && sudo systemctl restart RaidAutomator"