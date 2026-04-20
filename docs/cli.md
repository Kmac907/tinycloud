# CLI

## Main Commands

From `tinycloud\`, the built-in CLI now lives at the cloud-agnostic top-level command path and manages the local runtime plus environment settings for external tools:

```powershell
$env:TINYCLOUD_DATA_ROOT="$PWD\data"

go run .\cmd\tinycloud init
go run .\cmd\tinycloud start
go run .\cmd\tinycloud wait --timeout 30s
go run .\cmd\tinycloud status runtime --json
go run .\cmd\tinycloud status services --json
go run .\cmd\tinycloud logs -f
go run .\cmd\tinycloud config show --json
go run .\cmd\tinycloud services list --json
go run .\cmd\tinycloud endpoints
go run .\cmd\tinycloud env terraform
go run .\cmd\tinycloud env pulumi
go run .\cmd\tinycloud stop
```

From `tinycloud\`, the same control CLI is also available through the repo-root wrapper:

```powershell
.\scripts\tinycloud.ps1 init
.\scripts\tinycloud.ps1 start
.\scripts\tinycloud.ps1 status runtime
.\scripts\tinycloud.ps1 status services
.\scripts\tinycloud.ps1 env pulumi
.\scripts\tinycloudd.ps1
```

If you want real installed commands like `tinycloud init`, build the binaries under `.\bin` and add that directory to `PATH`. See [installation.md](installation.md).

The intended end state is that normal TinyCloud usage happens through compiled binaries on `PATH`, not through mandatory PowerShell wrappers. The current PowerShell scripts are transitional compatibility paths while the remaining wrapper/runtime orchestration is moved into the Go command layer.

Current implemented `tinycloud` help surface:

- `start [--attached|--detached] [--services <list>] [--json]`
- `stop`
- `restart [--detached|--attached]`
- `wait [--timeout <duration>]`
- `logs [-f]`
- `status [runtime|services] [--json]`
- `config show [--json]`
- `config validate`
- `services list [--json]`
- `services enable <names...>`
- `services disable <names...>`
- `init`
- `reset`
- `endpoints [--json]`
- `snapshot create [path]`
- `snapshot restore <path>`
- `seed apply <path>`
- `env terraform`
- `env pulumi`

Planned install and environment-preparation commands:

- `tinycloud setup`
- `tinycloud setup --full`

Those commands are part of the planned distribution model and are not implemented today. They are not part of the current `tinycloud` help surface. See [distribution.md](distribution.md).

## Runtime Model

The built-in `tinycloud` CLI is not an Azure CLI replacement. It is the local runtime manager plus endpoint, config, and service-control surface for both supported local runtime backends:

- Docker is the default backend when Docker is available locally. `tinycloud start` auto-builds the repo-root `tinycloud-azure` image if needed and then manages the active TinyCloud container for `status`, `logs`, `wait`, `restart`, and `stop`.
- `--backend process` keeps the managed local `tinycloudd` binary workflow available when you want to stay outside Docker.
- `tinycloud start` defaults to detached startup so it returns control to the shell; use `tinycloud start --attached` when you want the foreground log-streaming path instead.

`tinycloud start` accepts LocalStack-style bootstrap inputs for the current local runtime workflow:

- `--backend docker|process`
- `--services ...`
- `--env KEY=VALUE`
- `--publish HOSTPORT:CONTAINERPORT`
- `--volume HOSTPATH:CONTAINERPATH`
- `--network NAME`

## Service Selection

The runtime honors `TINYCLOUD_SERVICES` so listener startup is explicit instead of implicitly always-on. The current service-selection model accepts either individual services or family aliases:

- `management`
- `storage`
- `secrets-config`
- `data`
- `messaging`
- `networking`

For example, this keeps only the ARM/admin surface active while leaving the data-plane listeners disabled:

```powershell
$env:TINYCLOUD_SERVICES="management"
go run .\cmd\tinycloudd
```

When service selection is in use, `/_admin/runtime`, `/_admin/services`, `tinycloud endpoints`, and metadata discovery reflect the enabled service set rather than advertising listeners that were never started.

`tinycloud services enable ...` and `tinycloud services disable ...` persist the selected service set under `.tinycloud-runtime\tinycloud.env` so later `tinycloud start`, `tinycloud restart`, and `tinycloud config show` calls reconnect to the same intended local runtime configuration. Because the current runtime backends do not live-toggle listeners, service changes currently require a restart. The human-readable CLI prints a service-selection summary plus explicit restart guidance, while `--json` output remains stable for automation.

## UX Rules

The human-readable terminal UX follows a LocalStack-style shape:

- interactive `tinycloud start` is the only command that prints the approved TinyCloud ASCII banner
- default `tinycloud start` prints lifecycle steps, a runtime summary, and the next useful follow-up commands, then returns control to the shell
- `tinycloud start --attached` is the explicit foreground mode when you want startup output followed by live logs
- `tinycloud status runtime` and `tinycloud status services` render terminal tables instead of raw key=value lines
- `tinycloud status services` is the runtime-status view, while `tinycloud services list` is the config/catalog inventory view
- `tinycloud config show` renders grouped Runtime, Ports, and Services sections
- `tinycloud endpoints` renders a stable endpoint table
- interactive `tinycloud start` and `tinycloud logs -f` render known structured TinyCloud runtime/request log lines as terminal sections instead of raw JSON, while unknown lines still fall back to raw output
- status icons such as `✓`, `✗`, and `‼` are used in human-readable output, with color only on the icon glyph itself in interactive terminals
- `--json` output remains banner-free and machine-readable

For the Docker backend, `status runtime` still reports the active TinyCloud container identity and image.

## Model 2 Command Direction

TinyCloud's command direction is intentionally LocalStack-style and Model 2:

- `tinycloud` is the native Model 2 TinyCloud command surface for runtime lifecycle, status, endpoints, config, services, and environment helpers
- `tinyterraform` is the TinyCloud analogue to `tflocal`
- `tinyaz` is the planned TinyCloud analogue to `azlocal`
- users should be able to keep using normal TinyCloud and Terraform command habits with minimal TinyCloud-specific setup
- `tinyterraform` and future `tinyaz` should invoke the real upstream binaries under the hood rather than reimplementing their command sets
- for officially supported command and resource families, both `tinycloud` and `tinyterraform` target a Model 2 shape: preserve the normal command structure and let the CLI resolve the correct TinyCloud runtime, management endpoint, or service endpoint underneath
- wrapper parity is intended to track the current TinyCloud emulation scope rather than only the runtime listener list; today that means the 18 emulator areas listed in the current-emulation-scope table
- `tinyaz` is intended to grow toward full wrapper coverage across all 18 current implemented TinyCloud emulation-scope areas, with the wrapper responsible for whatever TinyCloud compatibility behavior is needed to preserve a coherent Azure CLI-shaped workflow for each area
- `tinyterraform` is intended to grow toward full wrapper coverage only for the parts of the current implemented TinyCloud emulation scope that have credible real Terraform provider/resource coverage and that TinyCloud can satisfy accurately
- the broader `tinyterraform` implementation work belongs to its own explicit roadmap step after the per-tool contract is locked, rather than remaining implied inside contract wording alone
- for `tinyterraform`, that future scope is expected to be resource-oriented first: ARM resources, storage accounts and child resources, Key Vault resources and secrets, networking resources, private DNS resources, Service Bus hierarchy resources, Event Hubs hierarchy resources, and selective App Configuration, Cosmos DB, or limited deployment-template-backed resources once verified
- live operational objects such as queue messages, Service Bus messages, event payload publishing/consumption, and Cosmos document CRUD are not the primary `tinyterraform` target
- the final per-tool command-family contract is a later lock step after standalone `tinyaz` exists and can be verified against real behavior

The shared product-command entry layer now lives in the repo-root `tinycloud\cli\...` packages, while the older `tinycloud\azure\cmd\...` paths remain compatibility shims over that cloud-agnostic layer and the current Azure-backed runtime adapters stay under `tinycloud\azure\runtime\...`.

The repo root also keeps the older Azure-backed command paths working as compatibility paths:

```powershell
$env:GOCACHE="$PWD\.gocache"
go test ./azure/...
go run .\azure\cmd\tinycloud env pulumi
docker build -t tinycloud-azure .
```

The repo root also exposes a thin `tinycloud` wrapper for the current transition layout:

```powershell
.\scripts\tinycloud.ps1 env pulumi
```

Those repo-root wrappers build through repo-root-relative command package paths, preferring the top-level `cmd\...` entrypoints and falling back to the Azure compatibility paths under `azure\cmd\...` when needed. They cache the built binaries under `.tinycloud-runtime` and default their Go build cache to `tinycloud\.gocache`.

Current installed-binary shape:

- `tinycloud.exe` can be built today from `cmd\tinycloud`
- `tinyterraform.exe` can be built today from `cmd\tinyterraform`
- `tinyaz.exe` should be documented as a separate build only after standalone `cmd\tinyaz` exists
- PowerShell should not remain a hard dependency for normal CLI usage once that wrapper/runtime convergence work is complete
- the planned bootstrap-plus-setup install story should eventually make the manual binary-build path optional rather than the default onboarding flow
