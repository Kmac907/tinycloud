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

function Resolve-TinyCloudMainPackage {
    if (-not [string]::IsNullOrWhiteSpace($env:TINYCLOUD_MAIN_PACKAGE)) {
        return $env:TINYCLOUD_MAIN_PACKAGE
    }

    $topLevelPackage = Join-Path $repoRoot "cmd\tinycloud\main.go"
    if (Test-Path $topLevelPackage) {
        return ".\cmd\tinycloud"
    }

    return ".\azure\cmd\tinycloud"
}

if ([string]::IsNullOrWhiteSpace($env:GOCACHE)) {
    $env:GOCACHE = Join-Path $repoRoot ".gocache"
}

$tinycloudGoWorkdir = Resolve-TinyCloudGoWorkdir
$runtimeRoot = Resolve-TinyCloudRuntimeRoot
$tinycloudMainPackage = Resolve-TinyCloudMainPackage
$tinycloudExe = Join-Path $runtimeRoot "tinycloud.exe"

New-Item -ItemType Directory -Force $runtimeRoot | Out-Null

Push-Location $tinycloudGoWorkdir
try {
    & go build -o $tinycloudExe $tinycloudMainPackage
    if ($LASTEXITCODE -ne 0) {
        throw "failed to build tinycloud"
    }

    & $tinycloudExe @TinyCloudArgs
    exit $LASTEXITCODE
} finally {
    Pop-Location
}
