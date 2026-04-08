param(
    [switch]$IncludeOptionalRuntimes,
    [ValidateSet("user", "machine")]
    [string]$Scope = "user"
)

$ErrorActionPreference = "Stop"

function Assert-Winget {
    if (Get-Command winget -ErrorAction SilentlyContinue) {
        return
    }

    throw @"
winget is required but was not found.

Install App Installer first, then rerun this script:
  Add-AppxPackage -RegisterByFamilyName -MainPackage Microsoft.DesktopAppInstaller_8wekyb3d8bbwe
"@
}

function Install-Package {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Id
    )

    $args = @("install", "-e", "--id", $Id)
    if ($Scope -eq "machine") {
        $args += @("--scope", "machine")
    }

    Write-Host "Installing $Id..."
    & winget @args
}

function Write-Version {
    param(
        [Parameter(Mandatory = $true)]
        [string]$Label,
        [Parameter(Mandatory = $true)]
        [string]$Command,
        [Parameter(Mandatory = $true)]
        [string[]]$Arguments
    )

    $tool = Get-Command $Command -ErrorAction SilentlyContinue
    if (-not $tool) {
        Write-Warning "$Label is not on PATH yet."
        return
    }

    $output = & $tool.Source @Arguments 2>&1
    if ($LASTEXITCODE -ne 0) {
        Write-Warning "$Label did not report a version cleanly."
        return
    }

    Write-Host ("{0}: {1}" -f $Label, ($output | Select-Object -First 1))
}

Assert-Winget

$packages = @(
    "GoLang.Go",
    "Docker.DockerDesktop",
    "Hashicorp.Terraform",
    "Pulumi.Pulumi",
    "jqlang.jq"
)

if ($IncludeOptionalRuntimes) {
    $packages += @(
        "OpenJS.NodeJS.LTS",
        "Python.Python.3.13",
        "Microsoft.DotNet.SDK.8"
    )
}

foreach ($package in $packages) {
    Install-Package -Id $package
}

Write-Host ""
Write-Host "Verification"
Write-Host "------------"
Write-Version -Label "Go" -Command "go" -Arguments @("version")
Write-Version -Label "Docker" -Command "docker" -Arguments @("version", "--format", "{{.Client.Version}}")
Write-Version -Label "Docker Compose" -Command "docker" -Arguments @("compose", "version")
Write-Version -Label "Terraform" -Command "terraform" -Arguments @("version")
Write-Version -Label "Pulumi" -Command "pulumi" -Arguments @("version")
Write-Version -Label "jq" -Command "jq" -Arguments @("--version")

if ($IncludeOptionalRuntimes) {
    Write-Version -Label "Node.js" -Command "node" -Arguments @("--version")
    Write-Version -Label "Python" -Command "python" -Arguments @("--version")
    Write-Version -Label ".NET SDK" -Command "dotnet" -Arguments @("--version")
}
