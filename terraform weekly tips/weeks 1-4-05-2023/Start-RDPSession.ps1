param(
    [parameter(Mandatory=$true)][string]$IPAddress,
    [parameter(Mandatory=$true)][string]$Username,
    [parameter(Mandatory=$true)][string]$Password
)

Start-Sleep -Seconds 5
cmdkey /generic:$IPAddress /user:$Username /pass: $Password
mstsc /v:$IPAddress
Start-Sleep -Seconds 3
cmdkey /delete:$IPAddress