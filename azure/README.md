# TinyCloud Azure Emulator

TinyCloud is a local Azure-compatible emulator written in Go and packaged as a single Docker container. The current repository includes the foundation runtime:

- local CLI entrypoint: `tinycloud`
- long-running daemon: `tinycloudd`
- admin endpoints for health, metrics, reset, snapshot, and seed
- ARM subscription/provider/resource-group control plane
- async operation polling for ARM resource-group writes and deletes
- metadata discovery at `/metadata/endpoints`
- identity endpoints at `/metadata/identity`, `/metadata/identity/oauth2/token`, and `/oauth/token`
- Blob data-plane on port `4577`
- Docker image with non-root runtime and persistent data root at `/var/lib/tinycloud`

## Local smoke test

Run from `azure/`:

```powershell
$env:TINYCLOUD_DATA_ROOT="$PWD\data"
go run .\cmd\tinycloud init
go run .\cmd\tinycloud status
go run .\cmd\tinycloud snapshot create
go run .\cmd\tinycloudd
```

In a second terminal:

```powershell
Invoke-RestMethod http://127.0.0.1:4566/_admin/healthz
Invoke-RestMethod http://127.0.0.1:4566/_admin/metrics
Invoke-RestMethod -Method Post http://127.0.0.1:4566/_admin/snapshot
Invoke-RestMethod http://127.0.0.1:4566/metadata/endpoints
Invoke-RestMethod http://127.0.0.1:4566/metadata/identity
Invoke-RestMethod -Method Post http://127.0.0.1:4566/oauth/token -Body "resource=https://management.azure.com/" -ContentType "application/x-www-form-urlencoded"
Invoke-RestMethod -Method Put "http://127.0.0.1:4566/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-local?api-version=2024-01-01" -Body '{"location":"westus2"}' -ContentType "application/json"
Invoke-RestMethod -Method Put "http://127.0.0.1:4566/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-local/providers/Microsoft.Storage/storageAccounts/storelocal?api-version=2024-01-01" -Body '{"location":"westus2","sku":{"name":"Standard_LRS"}}' -ContentType "application/json"
```

## Docker smoke test

Build and run:

```powershell
docker build -t tinycloud-azure .
docker run --rm -p 4566:4566 -p 4577:4577 -p 4578:4578 -p 4579:4579 -p 4580:4580 -p 4581:4581 tinycloud-azure
```

Then verify:

```powershell
Invoke-RestMethod http://127.0.0.1:4566/_admin/healthz
Invoke-RestMethod -Method Post http://127.0.0.1:4566/_admin/snapshot
Invoke-RestMethod http://127.0.0.1:4566/metadata/endpoints
Invoke-RestMethod http://127.0.0.1:4566/metadata/identity
```

Persist state across runs with a bind mount or volume:

```powershell
docker run --rm -p 4566:4566 -v "${PWD}\data:/var/lib/tinycloud" tinycloud-azure
```

## Acceptance test matrix

| Area | Environment | Check |
| --- | --- | --- |
| CLI init/status | local | `tinycloud init` and `tinycloud status` complete with a writable local `TINYCLOUD_DATA_ROOT` |
| Snapshot default path | local | `tinycloud snapshot create` writes under the configured data root |
| Admin health/metrics | local | `/_admin/healthz` and `/_admin/metrics` return `200` |
| Metadata discovery | local | `/metadata/endpoints` returns ARM, auth, provider, and service URLs |
| Identity endpoints | local | `/metadata/identity` and `/oauth/token` return stable local auth metadata |
| Blob data-plane | local or Docker | create a storage account, then create/list/upload/download blobs on port `4577` |
| Container boot | Docker | `docker run` starts `tinycloudd` successfully |
| Container snapshot | Docker | `POST /_admin/snapshot` succeeds without an explicit `path` |
| Persistent container data | Docker | mounted `/var/lib/tinycloud` survives restart |
| Path restrictions | local or Docker | admin snapshot and seed reject paths outside the data root |

## Current scope

The current codebase now covers the core local runtime, SQLite persistence, ARM subscription/provider/resource-group/storage-account flows, async polling, minimal identity/token endpoints, and a real Blob data-plane service. Deployment records and deeper compatibility polish are still outstanding.

## Examples

- Terraform resource group example: `examples/terraform/resource-group`
- Pulumi environment notes: `examples/pulumi`
