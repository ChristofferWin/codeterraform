#This script suites to retrieve all required information about a specific resource from the tf docs given an arm template

#System variables
$JSBaseScriptName = "retrieve-js-content.js"

#params
param (
  [array]$ArmTemplateFilePaths = @("./")
)

##OVERALL ARM TEMPLATE FILE HANDELING
$ArmTemplateFileContents = @()
$ArmLocalFiles = @()
$ArmProviders = @()

foreach($FilePath in $ArmTemplateFilepaths) {
  try{
    $ArmLocalFiles = Get-ChildItem -Path $FilePath -Recurse -Name -Filter "*arm*" -Force -ErrorAction Stop
    foreach($ArmLocalFile in $ArmLocalFiles){
        $ArmTemplateFileContents += Get-Content $ArmLocalFile -ErrorAction Stop | ConvertFrom-Json
    }
}
  catch{
    $_
  }
}

foreach($Provider in $ArmTemplateFileContents){
  #https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/resources/network_security_group
  $ArmProviders = @("Microsoft.Network/network_security_group")
}

#Retrieve web scrape with JS executed from hashicorp docs site
foreach($Provider in $ArmProviders){
  $FileName = "$($JSBaseScriptName.Replace("js-content", "js-$($Provider.split("/")[1])"))"
  $ChangeContent = Get-Content $JSBaseScriptName | % {if($_ -like "*goto*"){$_.replace("''", "'https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs/$($Provider.split("/")[1])'")}else{$_}}
  $ChangeContent | Out-File $FileName -Force

  #Execute node .js on the hasicorp docs site
  node $JSBaseScriptName | Out-File $FileName.Replace(".js", ".html") -Force
  
  #Handeling the HTML
  $RawHTML = (Get-Content $FileName.Replace(".js", ".html")) -join "`n"
  $CleanHTML = ConvertFrom-HTML -Content $RawHTML -Engine AngleSharp
  #$TextContentRaw = "azurerm_$($CleanHTML.TextContent.split($Provider.split("/")[1]))"
}

#We wont know how nested an ARM template is
$DecompiledContent = [string[]]

foreach($ArmTemplate in $ArmTemplateFileContents){
  foreach($Property in $ArmTemplate.properties | Select-Object *){
    if($Property.ToString().ToLower() -notlike "default*"){
      $Property
    }
    else {
      Write-Output "Hello world"
    }
  }
}



