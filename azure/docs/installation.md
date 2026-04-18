# Installation

TinyCloud can be used in two ways:

- repo-local development usage through `go run .\cmd\...`
- installed CLI usage through built binaries on `PATH`

This page covers the installed CLI path so you can run commands like:

```powershell
tinycloud init
tinycloud start
tinycloud status runtime
```

## Prerequisites

- Go installed locally
- PowerShell on Windows
- Docker installed locally if you want the default Docker-backed runtime flow

## Build The Current CLI Binaries

From `tinycloud\`:

```powershell
New-Item -ItemType Directory -Force .\bin | Out-Null
go build -o .\bin\tinycloud.exe .\cmd\tinycloud
go build -o .\bin\tinyterraform.exe .\cmd\tinyterraform
```

Current state:

- `tinycloud.exe` is the main installed runtime CLI
- `tinyterraform.exe` is the installed Terraform compatibility wrapper
- `tinyaz.exe` is not buildable yet because standalone `cmd\tinyaz` does not exist today

When standalone `tinyaz` is implemented, it should be built separately like the other commands:

```powershell
go build -o .\bin\tinyaz.exe .\cmd\tinyaz
```

## Add The Binaries To PATH

For the current PowerShell session only:

```powershell
$env:PATH = "$PWD\bin;$env:PATH"
```

For a permanent user-level setup, add the repo `bin` directory to your Windows `PATH` through your normal shell or system settings workflow.

## Verify The Install

After `.\bin` is on `PATH`:

```powershell
tinycloud init
tinycloud start
tinycloud status runtime
```

If you also built `tinyterraform.exe`, verify it separately:

```powershell
tinyterraform version
```

## Installed CLI Versus Repo-Local Usage

Installed usage:

```powershell
tinycloud init
tinycloud start
```

Repo-local usage:

```powershell
go run .\cmd\tinycloud init
go run .\cmd\tinycloud start
```

Both paths use the same command entrypoint code. The difference is only whether you invoke a built binary from `PATH` or run directly from source.

## Notes

- The installed CLI shape currently covers `tinycloud` and `tinyterraform`.
- `tinyterraform` runtime-routed flows still have the same current Windows privilege requirements documented in [terraform.md](terraform.md).
- Standalone `tinyaz` should be documented as an installable binary only after `cmd\tinyaz` actually exists.
