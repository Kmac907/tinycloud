<p align="center">
  <img src="./azure/assets/logo.png" width="300" />
</p>

<h1 align="center">TinyCloud</h1>

<p align="center">
  <a href="#"><img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.26" /></a>
  <a href="#"><img src="https://img.shields.io/badge/Docker-Single%20Container-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker Single Container" /></a>
  <a href="./azure/README.md"><img src="https://img.shields.io/badge/Azure-Implemented%20Today-0078D4?style=for-the-badge&logo=microsoftazure&logoColor=white" alt="Azure implemented today" /></a>
</p>

<p align="center"><sub>Local cloud emulator project with shared CLI infrastructure at the repo root and provider-specific implementations under dedicated platform folders.</sub></p>

TinyCloud is a local cloud emulator project. The repo root holds the shared command surfaces, wrappers, and runtime-management layers. Provider-specific emulator implementations live under dedicated folders such as [`azure/`](azure), which is the first implemented emulator in the current repo.

Normal product usage should converge on compiled `tinycloud`, `tinyterraform`, and future `tinyaz` binaries. The current PowerShell wrappers are transitional compatibility paths, not the intended long-term dependency model for normal cross-platform CLI usage.

## What This Repo Contains

At the top level:

- [`cmd/`](cmd): top-level user-facing command entrypoints such as `tinycloud`, `tinycloudd`, and `tinyterraform`
- [`cli/`](cli): shared command implementation layer used by those entrypoints
- [`scripts/`](scripts): repo-root wrapper scripts for the current CLI/runtime workflow
- [`azure/`](azure): the current implemented emulator, including its docs, runtime adapters, API handlers, examples, and roadmap

The repo structure is intentionally broader than Azure alone. `azure/` is the current provider implementation, and the project layout leaves room for additional emulator backends to be added under their own top-level folders later.

## Current State

Today, TinyCloud ships one implemented emulator backend:

- [`azure/`](azure): local Azure-compatible emulator with ARM, identity, storage, secrets, messaging, event streaming, network, and configuration workflows

Azure currently has:

- a full emulator landing page at [azure/README.md](azure/README.md)
- shared product docs under [docs/](docs)
- Azure-specific docs under [azure/docs/](azure/docs)
- examples under [azure/examples/](azure/examples)
- the active roadmap under [azure/plan.md](azure/plan.md)

## Quick Start

From the repo root:

```powershell
$env:TINYCLOUD_DATA_ROOT="$PWD\data"
go run .\cmd\tinycloud init
go run .\cmd\tinycloud start
go run .\cmd\tinycloud status runtime
```

Docker:

```powershell
docker build -t tinycloud-azure .
docker run --rm -p 4566:4566 -p 4577:4577 -p 4578:4578 -p 4579:4579 -p 4580:4580 -p 4581:4581 -p 4582:4582 -p 4583:4583 -p 4584:4584/udp -p 4585:4585 tinycloud-azure
```

Those commands currently start the Azure-backed TinyCloud runtime because Azure is the implemented emulator in this repo today.

## Installed CLI

To use real terminal commands like `tinycloud init` instead of `go run`, build the current binaries from the repo root and put them on `PATH`:

```powershell
New-Item -ItemType Directory -Force .\bin | Out-Null
go build -o .\bin\tinycloud.exe .\cmd\tinycloud
go build -o .\bin\tinyterraform.exe .\cmd\tinyterraform
$env:PATH = "$PWD\bin;$env:PATH"
```

Then:

```powershell
tinycloud init
tinycloud start
tinycloud status runtime
```

Standalone `tinyaz` is planned but not implemented yet, so there is no `cmd\tinyaz` build target today.

The current `tinycloud` help surface includes `start`, `stop`, `restart`, `wait`, `logs`, `status`, `config`, `services`, `init`, `reset`, `endpoints`, `snapshot`, `seed`, and `env`. It does not include `setup` or `setup --full` today.

The current install path is still manual. The intended official install story is:

1. bootstrap the `tinycloud` CLI from your own TinyCloud domain
2. run `tinycloud setup --full`
3. let the CLI validate or install the full local suite

That bootstrap-plus-setup flow is planned, not implemented today.

## Dependency Matrix

| Command | Current State | External Dependency |
| --- | --- | --- |
| `tinycloud` | implemented today | Go for source builds; Docker is the typical local runtime backend |
| `tinyterraform` | implemented today | Terraform must be installed locally |
| `tinyaz` | planned, not implemented yet | Azure CLI `az` is expected to be installed locally under the current wrapper model |

Current note:

- some Windows wrapper flows still use PowerShell during the ongoing transition
- the intended product direction is binary-first and cross-platform, so PowerShell should not remain a hard dependency for normal CLI usage

## Commands

Current repo-root command surfaces:

- `tinycloud`: runtime lifecycle, status, endpoints, config, logs, services, and environment helpers
- `tinycloudd`: local daemon entrypoint for the managed process backend
- `tinyterraform`: Terraform compatibility wrapper for the current Azure-backed TinyCloud runtime
- `tinyaz`: planned Azure CLI compatibility wrapper, not implemented yet

Planned install and distribution command surface:

- `tinycloud setup`: validate and prepare the local TinyCloud environment
- `tinycloud setup --full`: bootstrap the full local suite, including runtime image, config/data roots, and supported toolchain validation or management

Those `setup` commands are part of the planned distribution model only. They are not in the current `tinycloud` CLI help output.

## Model 2 Direction

TinyCloud's current command direction is Model 2 for both `tinycloud` and `tinyterraform`:

- `tinycloud` is the native Model 2 TinyCloud CLI: users keep a normal product command shape while the CLI manages the local runtime, status, endpoints, and environment wiring
- `tinyterraform` is the Terraform-facing Model 2 compatibility command: for supported flows it preserves normal Terraform command shape while routing to the correct TinyCloud-managed runtime and endpoints

The planned `tinyaz` command is intended to follow that same Model 2 direction once it exists as a standalone command.

## Where To Read Next

- Start with [azure/README.md](azure/README.md) for the currently implemented emulator
- Use [azure/docs/overview.md](azure/docs/overview.md) for the current Azure emulation scope
- Use [docs/installation.md](docs/installation.md) for installed CLI setup
- Use [docs/distribution.md](docs/distribution.md) for the planned bootstrap, packaging, and release model
- Use [docs/cli.md](docs/cli.md) for shared command behavior
- Use [azure/docs/terraform.md](azure/docs/terraform.md) for current `tinyterraform` behavior and limits
- Use [docs/development.md](docs/development.md) for repo-wide development workflow, smoke tests, and Docker validation

## Project Direction

TinyCloud is structured so the repo root can host shared command and wrapper infrastructure while individual emulator backends evolve under provider-specific folders.

Current direction:

- Azure is the implemented backend today
- the shared repo-root CLI/runtime layer is already cloud-agnostic in shape
- additional emulator backends can be added later without making the repo root Azure-specific

## License

This project is licensed under the Apache License 2.0. See [LICENSE](LICENSE).

The TinyCloud name and branding are reserved for the project and are not granted by the software license except for reasonable descriptive use.

## Contributing

For contribution expectations, including the current CLA requirement for significant contributions, see [CONTRIBUTING.md](CONTRIBUTING.md).

For current development workflow and smoke tests, start with [docs/development.md](docs/development.md). For Azure-emulator-specific docs, start with [azure/README.md](azure/README.md).
