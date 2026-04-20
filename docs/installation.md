# Installation

TinyCloud can be used in two ways:

- repo-local development usage through `go run .\cmd\...`
- installed CLI usage through built binaries on `PATH`

The intended long-term product model is binary-first and cross-platform. PowerShell is part of the current Windows transition path, but it should not remain a hard dependency for normal installed CLI usage.

This page covers the installed CLI path so you can run commands like:

```powershell
tinycloud init
tinycloud start
tinycloud status runtime
```

## Prerequisites

- Go installed locally
- PowerShell on Windows for the current transitional wrapper and documentation path
- Docker installed locally if you want the default Docker-backed runtime flow

## Dependency Matrix

| Command | Current State | External Dependency |
| --- | --- | --- |
| `tinycloud` | implemented today | Go for source builds; Docker is the typical local runtime backend |
| `tinyterraform` | implemented today | Terraform must be installed locally |
| `tinyaz` | planned, not implemented yet | Azure CLI `az` is expected to be installed locally under the current wrapper model |

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
- the current `tinycloud` help surface does not include `setup` or `setup --full`

When standalone `tinyaz` is implemented, it should be built separately like the other commands:

```powershell
go build -o .\bin\tinyaz.exe .\cmd\tinyaz
```

## Current Install Story Versus Planned Install Story

Current install story:

- build the binaries locally
- add the binary directory to `PATH`
- run `tinycloud ...` directly

Planned official install story:

1. bootstrap the `tinycloud` CLI from a TinyCloud-hosted installer URL
2. run `tinycloud setup --full`
3. let the CLI validate or install the full local suite

That bootstrap-plus-setup flow is planned, not implemented today.

Today, the installed `tinycloud` CLI surface is the same one exposed by `go run .\cmd\tinycloud`: `start`, `stop`, `restart`, `wait`, `logs`, `status`, `config`, `services`, `init`, `reset`, `endpoints`, `snapshot`, `seed`, and `env`. `tinycloud setup` and `tinycloud setup --full` belong to the roadmap, not the current installed binary behavior.

See [distribution.md](distribution.md) for the broader packaging and release model.

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
- Both `tinycloud` and `tinyterraform` should be documented as Model 2 command surfaces: keep the normal command shape and have the CLI resolve TinyCloud-managed runtime or endpoint wiring on the user's behalf.
- `tinyterraform` runtime-routed flows still have the same current Windows privilege requirements documented in [../azure/docs/terraform.md](../azure/docs/terraform.md).
- Standalone `tinyaz` should be documented as an installable binary only after `cmd\tinyaz` actually exists.
- PowerShell should be treated as a transitional compatibility tool, not the long-term product dependency model for normal TinyCloud CLI usage.
- The planned `tinycloud setup` and `tinycloud setup --full` flow belongs to the roadmap and distribution model, not the current implemented install surface.
