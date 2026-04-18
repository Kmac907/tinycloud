<p align="center">
  <img src="./assets/logo.png" width="300" />
</p>

<h1 align="center">TinyCloud Azure Emulator</h1>

<p align="center">
  <a href="#"><img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.26" /></a>
  <a href="#"><img src="https://img.shields.io/badge/Docker-Single%20Container-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker Single Container" /></a>
  <a href="./docs/overview.md"><img src="https://img.shields.io/badge/Azure-18%20Emulation%20Areas-0078D4?style=for-the-badge&logo=microsoftazure&logoColor=white" alt="Azure 18 emulation areas" /></a>
  <a href="https://x.com/TheOmniDev"><img src="https://img.shields.io/badge/X-@TheOmniDev-000000?style=for-the-badge&logo=x&logoColor=white" alt="X TheOmniDev" /></a>
</p>

<p align="center"><sub>Develop and test Azure-facing applications locally with a focused emulator for ARM, identity, storage, document data, private DNS, network security, secrets, messaging, and event streaming workflows.</sub></p>

TinyCloud is a local Azure-compatible emulator for ARM, identity, storage, secrets, messaging, event streaming, and selected network workflows. It is designed for targeted local Azure workflow testing, not full Azure parity.

## Supported Today

Current emulator status:

- `17` implemented emulator areas
- `1` partial emulator area: ARM deployments
- `0` listed areas not implemented

Current implemented areas include:

- ARM tenants, subscriptions, providers, resource groups, storage accounts, Key Vault resources, VNets, subnets, NSGs, private DNS zones, and private DNS A records
- metadata, OAuth, managed identity, and admin/runtime endpoints
- Blob, Queue, Table, Key Vault secrets, Service Bus, Event Hubs, App Configuration, and Cosmos data-plane workflows

The current emulation scope is tracked as 18 emulator areas. That scope is broader than the runtime service-selection surface.

See [docs/overview.md](docs/overview.md) for the full support table, current port layout, and exact emulated surface.

## Quick Start

From `tinycloud\`:

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

For direct API examples, per-service usage, and smoke-test flows, see [docs/services.md](docs/services.md) and [docs/development.md](docs/development.md).

## Install

To use real terminal commands like `tinycloud init` instead of repo-local `go run` commands, build the current CLI binaries and put them on `PATH`:

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

See [docs/installation.md](docs/installation.md) for the full install/setup flow, including the future separate `tinyaz.exe` build once standalone `tinyaz` exists.

## CLI

TinyCloud exposes three user-facing command surfaces:

- `tinycloud`: runtime lifecycle, status, endpoints, config, logs, services, and environment helpers
- `tinyterraform`: Terraform compatibility wrapper
- `tinyaz`: planned Azure CLI compatibility wrapper

The built-in `tinycloud` CLI manages the local runtime through the repo-root Go entrypoints under `cmd\...` and the repo-root wrappers under `scripts\...`.

See:

- [docs/cli.md](docs/cli.md)
- [docs/installation.md](docs/installation.md)
- [docs/terraform.md](docs/terraform.md)

## Wrapper Direction

TinyCloud follows a LocalStack-style wrapper model:

- `tinyterraform` is the TinyCloud analogue to `tflocal`
- `tinyaz` is the planned TinyCloud analogue to `azlocal`
- both wrappers should preserve normal upstream command shape as closely as practical
- both wrappers should invoke the real upstream tools rather than reimplementing their command sets

Current roadmap direction:

- `tinyaz` is intended to cover all 18 current emulation-scope areas
- `tinyterraform` is intended to cover the Terraform-feasible portion of that same scope

The current `tinyterraform` support is still narrow and ARM-first. See [docs/terraform.md](docs/terraform.md) for the current support statement and routing notes.

## Documentation

- [Overview](docs/overview.md)
- [Installation](docs/installation.md)
- [CLI](docs/cli.md)
- [Services](docs/services.md)
- [Terraform](docs/terraform.md)
- [Configuration](docs/configuration.md)
- [Development](docs/development.md)
- [Comparison](docs/comparison.md)

## Current Limitations

- ARM deployment execution is still partial and intentionally narrow
- `tinyterraform` support is still limited by real Terraform provider/resource coverage
- `tinyterraform` is still ARM-first today; broad automatic per-service Terraform routing is not yet verified
- standalone `tinyaz` is not implemented yet
- this is not a blanket Azure CLI, Terraform-provider, or SDK parity environment today

## Examples

- Terraform resource group example: [`examples/terraform/resource-group`](examples/terraform/resource-group)
- Pulumi environment notes: [`examples/pulumi`](examples/pulumi)

## Contributing

For local smoke tests, development commands, and Docker validation flows, start with [docs/development.md](docs/development.md).
