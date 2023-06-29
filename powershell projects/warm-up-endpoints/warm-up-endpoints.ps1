param(
    [parameter(Mandatory = $true)][array]$URLS,
    [parameter(Mandatory = $false)][switch]$Aggressive
)

[string]$Body = ""

foreach($URL in $URLS){
    if($URL -notmatch "^/((([A-Za-z]{3,9}:(?:\/\/)?)(?:[-;:&=\+\$,\w]+@)?[A-Za-z0-9.-]+|(?:www.|[-;:&=\+\$,\w]+@)[A-Za-z0-9.-]+)((?:\/[\+~%\/.\w-_]*)?\??(?:[-\+=&;%@.\w_]*)#?(?:[\w]*))?)/$"){
        $Body += "URL: $URL is NOT correctly formatted`n"
    }
    else{
        $Body += "URL: $URL is OK`n"
    }
}

if($Body.Contains("NOT")){
    Write-Warning "1 or more of the URLS provided were not correctly formatted"
    $Body
}