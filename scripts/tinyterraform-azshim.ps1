param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Args)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$logPath = $env:TINYTERRAFORM_AZ_LOG
Add-Content -Path $logPath -Value ($Args -join ' ')

$account = @{
    id = $env:ARM_SUBSCRIPTION_ID
    name = "TinyCloud"
    user = @{
        name = "tinycloud"
        type = "servicePrincipal"
    }
    tenantId = $env:ARM_TENANT_ID
    environmentName = "AzureCloud"
    isDefault = $true
}

if ($Args.Length -ge 1 -and $Args[0] -eq "version") {
    @{
        "azure-cli" = "2.99.0"
        "azure-cli-core" = "2.99.0"
    } | ConvertTo-Json -Compress
    exit 0
}

if ($Args.Length -ge 2 -and $Args[0] -eq "account" -and $Args[1] -eq "show") {
    $account | ConvertTo-Json -Compress
    exit 0
}

if ($Args.Length -ge 2 -and $Args[0] -eq "account" -and $Args[1] -eq "list") {
    @($account) | ConvertTo-Json -Compress -AsArray
    exit 0
}

if ($Args.Length -ge 2 -and $Args[0] -eq "account" -and $Args[1] -eq "get-access-token") {
    $resource = "https://management.azure.com/"
    for ($i = 0; $i -lt $Args.Length; $i++) {
        if ($Args[$i] -eq "--resource" -and $i + 1 -lt $Args.Length) {
            $resource = $Args[$i + 1]
        }
        if ($Args[$i] -eq "--scope" -and $i + 1 -lt $Args.Length) {
            $resource = ($Args[$i + 1] -replace "/.default$", "")
        }
    }

    $token = Invoke-RestMethod -Method Post -Uri "http://127.0.0.1:4566/oauth/token" -Body @{ resource = $resource }
    $expiresAt = (Get-Date).AddHours(1)
    @{
        accessToken = $token.access_token
        expiresOn = $expiresAt.ToString("yyyy-MM-dd HH:mm:ss.ffffff")
        expires_on = [int][double]::Parse((Get-Date $expiresAt -UFormat %s))
        subscription = $env:ARM_SUBSCRIPTION_ID
        tenant = $env:ARM_TENANT_ID
        tokenType = "Bearer"
    } | ConvertTo-Json -Compress
    exit 0
}

Write-Error ("unsupported az command: " + ($Args -join " "))
exit 1
