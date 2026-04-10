[CmdletBinding()]
param(
    [Parameter(ValueFromRemainingArguments = $true)]
    [string[]]$TinyCloudArgs
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot

function Resolve-TinyCloudGoWorkdir {
    if (-not [string]::IsNullOrWhiteSpace($env:TINYCLOUD_GO_WORKDIR)) {
        return (Resolve-Path -LiteralPath $env:TINYCLOUD_GO_WORKDIR).Path
    }

    return $repoRoot
}

function Resolve-TinyCloudRuntimeRoot {
    if (-not [string]::IsNullOrWhiteSpace($env:TINYCLOUD_RUNTIME_ROOT)) {
        return $env:TINYCLOUD_RUNTIME_ROOT
    }

    return (Join-Path $repoRoot ".tinycloud-runtime")
}

if ([string]::IsNullOrWhiteSpace($env:GOCACHE)) {
    $env:GOCACHE = Join-Path (Join-Path $repoRoot "azure") ".gocache"
}

$tinycloudGoWorkdir = Resolve-TinyCloudGoWorkdir
$runtimeRoot = Resolve-TinyCloudRuntimeRoot
$tinycloudExe = Join-Path $runtimeRoot "tinycloudd.exe"

New-Item -ItemType Directory -Force $runtimeRoot | Out-Null

Push-Location $tinycloudGoWorkdir
try {
    & go build -o $tinycloudExe tinycloud/cmd/tinycloudd
    if ($LASTEXITCODE -ne 0) {
        throw "failed to build tinycloudd"
    }

    & $tinycloudExe @TinyCloudArgs
    exit $LASTEXITCODE
} finally {
    Pop-Location
}
