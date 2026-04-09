[CmdletBinding()]
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$TerraformArgs
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

if (-not $TerraformArgs -or $TerraformArgs.Count -eq 0) {
    throw "usage: .\scripts\tinyterraform.ps1 <terraform arguments>"
}

function Normalize-TerraformArgs {
    param([string[]]$InputArgs)

    $normalized = New-Object System.Collections.Generic.List[string]
    for ($i = 0; $i -lt $InputArgs.Count; $i++) {
        $arg = $InputArgs[$i]
        if ($arg -eq "-chdir=" -and $i + 1 -lt $InputArgs.Count) {
            $normalized.Add("-chdir=$($InputArgs[$i + 1])")
            $i++
            continue
        }
        $normalized.Add($arg)
    }

    return $normalized.ToArray()
}

function Get-TerraformSubcommand {
    param([string[]]$InputArgs)

    for ($i = 0; $i -lt $InputArgs.Count; $i++) {
        $arg = $InputArgs[$i]
        if ($arg -eq "-chdir" -or $arg -eq "-chdir=") {
            $i++
            continue
        }
        if (-not [string]::IsNullOrWhiteSpace($arg) -and -not $arg.StartsWith("-")) {
            return $arg.ToLowerInvariant()
        }
    }
    return ""
}

function Test-RequiresTinyCloudRuntime {
    param([string]$Subcommand)

    if ([string]::IsNullOrWhiteSpace($Subcommand)) {
        return $false
    }

    return $Subcommand -notin @(
        "help",
        "version",
        "fmt",
        "validate",
        "providers",
        "state",
        "output",
        "show",
        "graph",
        "workspace",
        "force-unlock",
        "taint",
        "untaint"
    )
}

$TerraformArgs = Normalize-TerraformArgs -InputArgs $TerraformArgs

$terraformSubcommand = Get-TerraformSubcommand -InputArgs $TerraformArgs
$requiresTinyCloudRuntime = Test-RequiresTinyCloudRuntime -Subcommand $terraformSubcommand
$requiresPrivilegedRuntime = $requiresTinyCloudRuntime -and $terraformSubcommand -ne "init"

$principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if ($requiresPrivilegedRuntime -and -not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    throw "tinyterraform requires an elevated PowerShell session because it temporarily maps management.azure.com in the hosts file."
}

$repoRoot = Split-Path -Parent $PSScriptRoot
$terraformDir = (Get-Location).Path
$overridePath = Join-Path $terraformDir "tinycloud_providers_override.tf"
$runtimeRoot = Join-Path $repoRoot ".tinyterraform-runtime"
$dataRoot = Join-Path $runtimeRoot "data"
$shimDir = Join-Path $runtimeRoot "shim"
$serverStdout = Join-Path $runtimeRoot "tinycloud.stdout.log"
$serverStderr = Join-Path $runtimeRoot "tinycloud.stderr.log"
$shimLog = Join-Path $runtimeRoot "azshim.log"
$tinycloudExe = Join-Path $runtimeRoot "tinycloud.exe"
$hostsPath = Join-Path $env:SystemRoot "System32\drivers\etc\hosts"
$hostsStartMarker = "# tinycloud terraform begin"
$hostsEndMarker = "# tinycloud terraform end"
$healthEndpoint = "http://127.0.0.1:4566/_admin/healthz"

$goCache = $env:GOCACHE
if ([string]::IsNullOrWhiteSpace($goCache)) {
    $goCache = Join-Path $repoRoot ".gocache"
}

function Resolve-TerraformExe {
    if (-not [string]::IsNullOrWhiteSpace($env:TERRAFORM_EXE) -and (Test-Path $env:TERRAFORM_EXE)) {
        return (Resolve-Path -LiteralPath $env:TERRAFORM_EXE).Path
    }

    $searched = New-Object System.Collections.Generic.List[string]

    $command = Get-Command terraform -ErrorAction SilentlyContinue
    if ($command -and $command.Source -and (Test-Path $command.Source)) {
        return $command.Source
    }
    $searched.Add("PATH:terraform")

    $command = Get-Command terraform.exe -ErrorAction SilentlyContinue
    if ($command -and $command.Source -and (Test-Path $command.Source)) {
        return $command.Source
    }
    $searched.Add("PATH:terraform.exe")

    $candidates = @(
        "C:\Program Files\Terraform\terraform.exe",
        "C:\HashiCorp\Terraform\terraform.exe"
    )
    if (-not [string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) {
        $candidates += (Join-Path $env:LOCALAPPDATA "Microsoft\WinGet\Packages\Hashicorp.Terraform_Microsoft.Winget.Source_8wekyb3d8bbwe\terraform.exe")
        $candidates += (Join-Path $env:LOCALAPPDATA "Programs\Terraform\terraform.exe")
    }
    if (-not [string]::IsNullOrWhiteSpace($HOME)) {
        $candidates += (Join-Path $HOME "AppData\Local\Microsoft\WinGet\Packages\Hashicorp.Terraform_Microsoft.Winget.Source_8wekyb3d8bbwe\terraform.exe")
    }
    foreach ($candidate in $candidates) {
        $searched.Add($candidate)
        if (Test-Path $candidate) {
            return $candidate
        }
    }

    $wingetMatches = Get-ChildItem "C:\Users" -Directory -ErrorAction SilentlyContinue |
        ForEach-Object {
            Join-Path $_.FullName "AppData\Local\Microsoft\WinGet\Packages\Hashicorp.Terraform_Microsoft.Winget.Source_8wekyb3d8bbwe\terraform.exe"
        } |
        Where-Object { Test-Path $_ }
    if ($wingetMatches) {
        return ($wingetMatches | Select-Object -First 1)
    }
    $searched.Add("C:\Users\*\AppData\Local\Microsoft\WinGet\Packages\Hashicorp.Terraform_Microsoft.Winget.Source_8wekyb3d8bbwe\terraform.exe")

    throw ("terraform.exe was not found. Set `$env:TERRAFORM_EXE to the full path if needed. Searched: " + ($searched -join ", "))
}

function Remove-HostsBlock {
    if (-not (Test-Path $hostsPath)) {
        return
    }

    $content = Get-Content -Raw $hostsPath
    $pattern = [regex]::Escape("`r`n$hostsStartMarker`r`n127.0.0.1 management.azure.com`r`n$hostsEndMarker`r`n")
    $updated = [regex]::Replace($content, $pattern, "")
    $updated = $updated.Replace("`n$hostsStartMarker`n127.0.0.1 management.azure.com`n$hostsEndMarker`n", "")
    if ($updated -ne $content) {
        Set-Content -Path $hostsPath -Value $updated
    }
}

New-Item -ItemType Directory -Force $runtimeRoot,$dataRoot,$shimDir | Out-Null
$terraformExe = Resolve-TerraformExe
Write-Verbose ("Using terraform: " + $terraformExe)
$shimPowerShellExe = (Get-Process -Id $PID).Path

if (-not $requiresTinyCloudRuntime) {
    & $terraformExe @TerraformArgs
    exit $LASTEXITCODE
}

$azShimLauncher = @'
@echo off
"{0}" -NoProfile -File "%~dp0azshim.ps1" %*
exit /b %ERRORLEVEL%
'@ -f $shimPowerShellExe
Set-Content -Path (Join-Path $shimDir "az.cmd") -Value $azShimLauncher

$azShimScript = @'
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
'@
Set-Content -Path (Join-Path $shimDir "azshim.ps1") -Value $azShimScript

$env:GOCACHE = $goCache
$env:TINYCLOUD_DATA_ROOT = $dataRoot
$env:TINYCLOUD_ADVERTISE_HOST = "management.azure.com"
$env:TINYCLOUD_MGMT_HTTP_PORT = "4566"
$env:TINYCLOUD_MGMT_HTTPS_PORT = "443"
$env:ARM_SUBSCRIPTION_ID = "11111111-1111-1111-1111-111111111111"
$env:ARM_TENANT_ID = "00000000-0000-0000-0000-000000000001"
$env:ARM_USE_CLI = "true"
$env:TINYTERRAFORM_AZ_LOG = $shimLog
$env:PATH = "$shimDir;$env:PATH"

if ($requiresPrivilegedRuntime -and (Get-NetTCPConnection -LocalPort 443 -ErrorAction SilentlyContinue)) {
    throw "port 443 is already in use; tinyterraform cannot bind management.azure.com locally"
}

Push-Location $repoRoot
try {
    & go build -o $tinycloudExe .\cmd\tinycloud
    if ($LASTEXITCODE -ne 0) {
        throw "failed to build tinycloud"
    }

    if ($terraformSubcommand -eq "init") {
        Write-Host "Resetting TinyCloud runtime state for terraform init"
        & $tinycloudExe reset
        if ($LASTEXITCODE -ne 0) {
            throw "failed to reset tinycloud state"
        }
    }

    & $tinycloudExe init
    if ($LASTEXITCODE -ne 0) {
        throw "failed to initialize tinycloud state"
    }

    $envOutput = & $tinycloudExe env terraform
    if ($LASTEXITCODE -ne 0) {
        throw "failed to load TinyCloud Terraform environment"
    }
} finally {
    Pop-Location
}

$envMap = @{}
foreach ($line in ($envOutput -split "`r?`n")) {
    if ([string]::IsNullOrWhiteSpace($line) -or -not $line.Contains("=")) {
        continue
    }
    $parts = $line.Split("=", 2)
    $envMap[$parts[0]] = $parts[1]
}

$terraformEnvAllowList = @(
    "ARM_SUBSCRIPTION_ID",
    "ARM_TENANT_ID"
)

$terraformEnvClearList = @(
    "ARM_ENDPOINT",
    "ARM_ENVIRONMENT",
    "ARM_METADATA_HOST",
    "ARM_METADATA_HOSTNAME",
    "ARM_MSI_ENDPOINT",
    "ARM_USE_MSI"
)

$requiredKeys = @(
    "ARM_SUBSCRIPTION_ID",
    "ARM_TENANT_ID",
    "TINY_MGMT_HTTPS_CERT"
)
foreach ($key in $requiredKeys) {
    if (-not $envMap.ContainsKey($key) -or [string]::IsNullOrWhiteSpace($envMap[$key])) {
        throw "TinyCloud Terraform environment is missing $key"
    }
}

foreach ($key in $terraformEnvClearList) {
    Remove-Item -Path "Env:$key" -ErrorAction SilentlyContinue
}

$certPath = $envMap["TINY_MGMT_HTTPS_CERT"]
if (-not (Test-Path $certPath)) {
    throw "TinyCloud HTTPS certificate was not found at $certPath"
}

$cert = [System.Security.Cryptography.X509Certificates.X509Certificate2]::new($certPath)
$trusted = Get-ChildItem Cert:\CurrentUser\Root | Where-Object { $_.Thumbprint -eq $cert.Thumbprint }
if (-not $trusted) {
    Import-Certificate -FilePath $certPath -CertStoreLocation Cert:\CurrentUser\Root | Out-Null
}

$hostsContent = Get-Content -Raw $hostsPath
if ($hostsContent.Contains($hostsStartMarker)) {
    throw "hosts file already contains TinyCloud Terraform markers"
}

$hostsBlock = "`r`n$hostsStartMarker`r`n127.0.0.1 management.azure.com`r`n$hostsEndMarker`r`n"
Set-Content -Path $hostsPath -Value ($hostsContent + $hostsBlock)

$overrideBody = @"
provider "azurerm" {
  features {}
  use_cli = true
  resource_provider_registrations = "none"

  enhanced_validation {
    locations = false
    resource_providers = false
  }
}
"@

Set-Content -Path $overridePath -Value $overrideBody

$server = $null
try {
    $server = Start-Process -FilePath $tinycloudExe -ArgumentList "start" -PassThru -RedirectStandardOutput $serverStdout -RedirectStandardError $serverStderr -WorkingDirectory $repoRoot

    $healthy = $false
    for ($i = 0; $i -lt 40; $i++) {
        try {
            Invoke-RestMethod $healthEndpoint -TimeoutSec 2 | Out-Null
            $healthy = $true
            break
        } catch {
            Start-Sleep -Milliseconds 500
        }
    }
    if (-not $healthy) {
        throw "TinyCloud did not become healthy on $healthEndpoint"
    }

    foreach ($key in $terraformEnvAllowList) {
        Set-Item -Path "Env:$key" -Value $envMap[$key]
    }

    & $terraformExe @TerraformArgs
    exit $LASTEXITCODE
} finally {
    if ($server -and -not $server.HasExited) {
        Stop-Process -Id $server.Id -Force
    }
    Remove-Item -LiteralPath $overridePath -ErrorAction SilentlyContinue
    Remove-HostsBlock
}
