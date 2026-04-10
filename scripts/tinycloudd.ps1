[CmdletBinding()]
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$TinyCloudArgs
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot

if ([string]::IsNullOrWhiteSpace($env:GOCACHE)) {
    $env:GOCACHE = Join-Path (Join-Path $repoRoot "azure") ".gocache"
}

Push-Location $repoRoot
try {
    & go run .\azure\cmd\tinycloudd @TinyCloudArgs
    exit $LASTEXITCODE
} finally {
    Pop-Location
}
