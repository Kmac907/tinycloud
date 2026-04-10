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

$repoRoot = Split-Path -Parent $PSScriptRoot
$azureRoot = Join-Path $repoRoot "azure"
$azureWrapper = Join-Path $azureRoot "scripts\tinyterraform.ps1"

if (-not (Test-Path $azureWrapper)) {
    throw "could not locate the Azure tinyterraform wrapper at $azureWrapper"
}

if ([string]::IsNullOrWhiteSpace($env:TINYCLOUD_SOURCE_ROOT)) {
    $env:TINYCLOUD_SOURCE_ROOT = $azureRoot
}

if ([string]::IsNullOrWhiteSpace($env:TINYCLOUD_GO_WORKDIR)) {
    $env:TINYCLOUD_GO_WORKDIR = $repoRoot
}

if ([string]::IsNullOrWhiteSpace($env:TINYCLOUD_MAIN_PACKAGE)) {
    $env:TINYCLOUD_MAIN_PACKAGE = "tinycloud/cmd/tinycloud"
}

if ([string]::IsNullOrWhiteSpace($env:TINYTERRAFORM_RUNTIME_ROOT)) {
    $env:TINYTERRAFORM_RUNTIME_ROOT = Join-Path $repoRoot ".tinyterraform-runtime"
}

& $azureWrapper @TerraformArgs
exit $LASTEXITCODE
