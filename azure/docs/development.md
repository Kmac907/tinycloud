# Development

## Installed CLI Versus Repo-Local Usage

Contributor docs default to repo-local commands so you exercise the current source tree directly while developing:

```powershell
go run .\cmd\tinycloud init
go run .\cmd\tinycloud start
```

If you want to validate the installed-binary path instead, build the current CLI binaries under `.\bin`, add that directory to `PATH`, and then run the installed commands directly:

```powershell
tinycloud init
tinycloud start
tinycloud status runtime
```

See [installation.md](installation.md) for the full installed CLI setup flow, including the current `tinyterraform.exe` build and the future separate `tinyaz.exe` build once standalone `cmd\tinyaz` exists.

Contributor workflows still mention PowerShell because the current Windows wrappers are part of the transition path. That is current-state documentation, not the long-term product dependency model. Normal TinyCloud usage is intended to converge on cross-platform compiled binaries without requiring PowerShell.

## Local Smoke Tests

```powershell
$env:TINYCLOUD_DATA_ROOT="$PWD\data"
go run .\cmd\tinycloud init
go run .\cmd\tinycloud start
go run .\cmd\tinycloud wait --timeout 30s
```

To use the direct non-container runtime path instead, force the managed process backend explicitly:

```powershell
$env:TINYCLOUD_DATA_ROOT="$PWD\data"
go run .\cmd\tinycloud start --backend process
```

From `tinycloud\`, the same control/runtime entrypoints are also available through the repo-root wrappers:

```powershell
$env:TINYCLOUD_DATA_ROOT="$PWD\azure\data"
.\scripts\tinycloud.ps1 start
.\scripts\tinycloudd.ps1
```

In another terminal:

```powershell
Invoke-RestMethod http://127.0.0.1:4566/metadata/endpoints
Invoke-RestMethod http://127.0.0.1:4566/tenants?api-version=2024-01-01
Invoke-RestMethod http://127.0.0.1:4566/subscriptions?api-version=2024-01-01
Invoke-RestMethod -Method Put "http://127.0.0.1:4566/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-local?api-version=2024-01-01" -Body '{"location":"westus2"}' -ContentType "application/json"
Invoke-WebRequest -Method Put "http://127.0.0.1:4577/devstoreaccount1/docs?restype=container"
Invoke-RestMethod -Method Post "http://127.0.0.1:4581/namespaces" -Body '{"name":"local-messaging"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4582/stores" -Body '{"name":"tiny-settings"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4583/accounts" -Body '{"name":"local-cosmos"}' -ContentType "application/json"
Invoke-RestMethod -Method Put "http://127.0.0.1:4566/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-local/providers/Microsoft.Network/virtualNetworks/vnet-local?api-version=2024-01-01" -Body '{"location":"westus2","properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]}}}' -ContentType "application/json"
Invoke-RestMethod -Method Put "http://127.0.0.1:4566/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-local/providers/Microsoft.Network/networkSecurityGroups/nsg-local?api-version=2024-01-01" -Body '{"location":"westus2"}' -ContentType "application/json"
Invoke-RestMethod -Method Put "http://127.0.0.1:4566/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-local/providers/Microsoft.Network/privateDnsZones/internal.test?api-version=2024-01-01" -Body '{}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4585/namespaces" -Body '{"name":"local-streaming"}' -ContentType "application/json"
```

## Docker Smoke Tests

```powershell
docker build -t tinycloud-azure .
docker run --rm `
  -p 4566:4566 `
  -p 4577:4577 `
  -p 4578:4578 `
  -p 4579:4579 `
  -p 4580:4580 `
  -p 4581:4581 `
  -p 4582:4582 `
  -p 4583:4583 `
  -p 4584:4584/udp `
  -p 4585:4585 `
  -v "${PWD}\data:/var/lib/tinycloud" `
  tinycloud-azure
```

During the repo-root migration, the repo root now also has a first-class Dockerfile, so the same image can be built directly from `tinycloud\`:

```powershell
docker build -t tinycloud-azure .
```
