<p align="center">
  <img src="./assets/logo.png" width="300" />
</p>

<h1 align="center">TinyCloud Azure Emulator</h1>

<p align="center">
  <a href="#"><img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.26" /></a>
  <a href="#"><img src="https://img.shields.io/badge/Docker-Single%20Container-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker Single Container" /></a>
  <a href="#current-emulation-scope"><img src="https://img.shields.io/badge/Azure-ARM%20%2B%20Storage%20%2B%20Data%20%2B%20Secrets%20%2B%20Messaging-0078D4?style=for-the-badge&logo=microsoftazure&logoColor=white" alt="Azure ARM Storage Data Secrets and Messaging" /></a>
  <a href="https://x.com/TheOmniDev"><img src="https://img.shields.io/badge/X-@TheOmniDev-000000?style=for-the-badge&logo=x&logoColor=white" alt="X TheOmniDev" /></a>
</p>

<p align="center"><sub>Develop and test Azure-facing applications locally with a focused emulator for ARM, identity, storage, document data, private DNS, network security, secrets, messaging, and event streaming workflows.</sub></p>

TinyCloud is a local Azure-compatible emulator written in Go and packaged as a single container. It provides a compact Azure development environment for local iteration and CI by combining:

- Azure Resource Manager support for tenants, subscriptions, providers, resource groups, storage accounts, and Key Vault resources
- Azure-style async operation polling for supported control-plane writes
- metadata, OAuth, minimal IMDS-style managed identity endpoints, and a local HTTPS management listener
- real Blob, Queue Storage, Table Storage, Cosmos DB, private DNS, App Configuration, Key Vault secrets, Service Bus, Event Hubs, and basic network-security behavior on dedicated service ports
- admin/runtime endpoints for health, metrics, reset, snapshot, and seed

TinyCloud is designed for targeted local Azure workflow testing, not full Azure parity.

## Current Emulation Scope

Current status across the listed emulator areas:

- `17` implemented
- `1` partial
- `0` not implemented yet

### Support Levels

| Area | Current level | Notes |
| --- | --- | --- |
| ARM tenants/subscriptions/providers | Implemented | Includes provider registration records and tenant listing |
| ARM resource groups | Implemented | CRUD with Azure-style create/update semantics and async deletes |
| ARM storage accounts | Implemented | CRUD with Blob endpoint advertisement |
| ARM Key Vault resources | Implemented | CRUD for `Microsoft.KeyVault/vaults` |
| ARM deployments | Partial | Deployment records and async status are implemented; a narrow static template subset works for storage accounts and Key Vault vaults |
| Blob data-plane | Implemented | Containers, upload/download/list/delete, `HEAD`, compatibility headers |
| Managed identity and token endpoints | Implemented | Minimal IMDS-style behavior and signed local JWTs |
| Admin/runtime endpoints | Implemented | Health, metrics, reset, snapshot, seed |
| Key Vault secrets data-plane | Implemented | Secret set/get/list/delete on the dedicated Key Vault listener |
| Service Bus | Implemented | Namespaces, queues, topics, subscriptions, send/publish, receive, delete |
| Event Hubs | Implemented | Namespaces, hubs, publish, and ordered event reads |
| Virtual Networks | Implemented | ARM CRUD for virtual networks and subnets |
| Network Security Groups | Implemented | ARM CRUD for NSGs and nested security rules |
| Queue Storage | Implemented | Queue create/list and message send/receive/delete |
| Table Storage | Implemented | Table create/list/delete and entity upsert/get/list/delete |
| Cosmos DB | Implemented | Account, database, container, and document CRUD on the dedicated Cosmos listener |
| Private DNS | Implemented | ARM zone/A-record CRUD plus a live UDP resolver for A-record lookups |
| App Configuration | Implemented | Config store and key-value CRUD on the dedicated App Configuration listener |

### What Is Actually Emulated Today

- Azure Resource Manager:
  - `GET /tenants`
  - `GET /subscriptions`
  - `GET /providers`
  - provider registration
  - resource group CRUD
  - storage account CRUD
  - Key Vault resource CRUD
  - virtual network CRUD
  - subnet CRUD
  - network security group CRUD
  - network security rule CRUD
  - private DNS zone CRUD
  - private DNS A-record CRUD
  - deployment record/status routes
  - async operation polling
- Metadata and identity:
  - `GET /metadata/endpoints`
  - `GET /metadata/identity`
  - `GET /metadata/identity/oauth2/token`
  - `POST /oauth/token`
- Blob service on its own port:
  - create/list containers
  - upload/download/list/delete blobs
  - `HEAD` blob metadata
- Queue Storage on its own port:
  - create/list queues
  - send/receive/delete messages
- Table Storage on its own port:
  - create/list/delete tables
  - upsert/get/list/delete entities
- Cosmos DB on its own port:
  - create/list accounts
  - create/list databases
  - create/list containers
  - create/get/list/delete documents
- Private DNS:
  - private DNS zone CRUD through ARM
  - private DNS A-record CRUD through ARM
  - live UDP resolver for A-record lookups
- Key Vault on its own port:
  - set/get/list/delete secrets
- Service Bus on its own port:
  - create/list namespaces and queues
  - send/receive/delete queue messages
  - create/list topics and subscriptions
  - publish/receive/delete topic subscription messages
- Event Hubs on its own port:
  - create/list namespaces
  - create/list hubs
  - publish events
  - read ordered event streams from a sequence number
- App Configuration on its own port:
  - create/list config stores
  - put/get/list/delete key-values
- Admin/runtime:
  - `/_admin/healthz`
  - `/_admin/metrics`
  - `/_admin/reset`
  - `/_admin/snapshot`
  - `/_admin/seed`

## Ports

All listed listeners are active today.

| Port | Status | Purpose |
| --- | --- | --- |
| `4566` | Active | management endpoint: ARM, metadata, identity, OAuth, admin |
| `4567` | Active | management HTTPS endpoint |
| `4577` | Active | Blob data-plane |
| `4578` | Active | Queue Storage data-plane |
| `4579` | Active | Table Storage data-plane |
| `4580` | Active | Key Vault secrets data-plane |
| `4581` | Active | Service Bus data-plane |
| `4582` | Active | App Configuration data-plane |
| `4583` | Active | Cosmos DB data-plane |
| `4584/udp` | Active | private DNS resolver |
| `4585` | Active | Event Hubs data-plane |

## How To Interact With The Emulated Azure Environment

### 1. ARM Control Plane

Use the management endpoint on `http://127.0.0.1:4566` with Azure-style paths and `api-version`.

Create a resource group:

```powershell
Invoke-RestMethod -Method Put `
  "http://127.0.0.1:4566/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-local?api-version=2024-01-01" `
  -Body '{"location":"westus2","tags":{"env":"dev"}}' `
  -ContentType "application/json"
```

Create a storage account:

```powershell
Invoke-RestMethod -Method Put `
  "http://127.0.0.1:4566/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-local/providers/Microsoft.Storage/storageAccounts/storelocal?api-version=2024-01-01" `
  -Body '{"location":"westus2","sku":{"name":"Standard_LRS"}}' `
  -ContentType "application/json"
```

Create a Key Vault resource:

```powershell
Invoke-RestMethod -Method Put `
  "http://127.0.0.1:4566/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-local/providers/Microsoft.KeyVault/vaults/vaultlocal?api-version=2024-01-01" `
  -Body '{"location":"westus2","properties":{"tenantId":"00000000-0000-0000-0000-000000000001","sku":{"name":"standard"}}}' `
  -ContentType "application/json"
```

Resource-group create/update is synchronous and returns Azure-style `201 Created` or `200 OK`. Resource-group deletes, storage-account writes, and Key Vault writes return `202 Accepted` plus `Azure-AsyncOperation`, `Location`, and `Retry-After`.

Create a virtual network and subnet:

```powershell
Invoke-RestMethod -Method Put `
  "http://127.0.0.1:4566/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-local/providers/Microsoft.Network/virtualNetworks/vnet-local?api-version=2024-01-01" `
  -Body '{"location":"westus2","properties":{"addressSpace":{"addressPrefixes":["10.0.0.0/16"]}}}' `
  -ContentType "application/json"

Invoke-RestMethod -Method Put `
  "http://127.0.0.1:4566/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-local/providers/Microsoft.Network/virtualNetworks/vnet-local/subnets/frontend?api-version=2024-01-01" `
  -Body '{"properties":{"addressPrefix":"10.0.1.0/24"}}' `
  -ContentType "application/json"
```

Create a network security group and rule:

```powershell
Invoke-RestMethod -Method Put `
  "http://127.0.0.1:4566/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-local/providers/Microsoft.Network/networkSecurityGroups/nsg-local?api-version=2024-01-01" `
  -Body '{"location":"westus2","tags":{"env":"dev"}}' `
  -ContentType "application/json"

Invoke-RestMethod -Method Put `
  "http://127.0.0.1:4566/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-local/providers/Microsoft.Network/networkSecurityGroups/nsg-local/securityRules/allow-https?api-version=2024-01-01" `
  -Body '{"properties":{"access":"Allow","direction":"Inbound","protocol":"Tcp","sourceAddressPrefix":"*","sourcePortRange":"*","destinationAddressPrefix":"*","destinationPortRange":"443","priority":100}}' `
  -ContentType "application/json"
```

### 2. Blob Storage Data-Plane

Blob runs on `http://127.0.0.1:4577`. Use the storage account name in the path.

Create a container:

```powershell
Invoke-WebRequest -Method Put "http://127.0.0.1:4577/storelocal/docs?restype=container"
```

Upload a blob:

```powershell
Invoke-WebRequest -Method Put `
  -Uri "http://127.0.0.1:4577/storelocal/docs/sample.pdf" `
  -Headers @{ "x-ms-blob-type" = "BlockBlob"; "x-ms-version" = "2023-11-03" } `
  -ContentType "application/pdf" `
  -InFile ".\sample.pdf"
```

Download a blob:

```powershell
Invoke-WebRequest `
  -Uri "http://127.0.0.1:4577/storelocal/docs/sample.pdf" `
  -OutFile ".\downloaded-sample.pdf"
```

### 3. Queue Storage Data-Plane

Queue Storage runs on `http://127.0.0.1:4578`.

Create a queue:

```powershell
Invoke-RestMethod -Method Post "http://127.0.0.1:4578/storelocal/queues" -Body '{"name":"jobs"}' -ContentType "application/json"
```

Send and receive a message:

```powershell
Invoke-RestMethod -Method Post "http://127.0.0.1:4578/storelocal/queues/jobs/messages" -Body '{"messageText":"run-report"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4578/storelocal/queues/jobs/messages/receive?maxMessages=1&visibilityTimeout=30"
```

### 4. Table Storage Data-Plane

Table Storage runs on `http://127.0.0.1:4579`.

Create a table and upsert an entity:

```powershell
Invoke-RestMethod -Method Post "http://127.0.0.1:4579/storelocal/Tables" -Body '{"name":"customers"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4579/storelocal/customers" -Body '{"partitionKey":"retail","rowKey":"cust-001","properties":{"Name":"Tiny Cloud"}}' -ContentType "application/json"
```

### 5. Key Vault Secrets Data-Plane

Key Vault secrets run on `http://127.0.0.1:4580`.

Set and read a secret:

```powershell
Invoke-RestMethod -Method Put "http://127.0.0.1:4580/vaultlocal/secrets/app-secret" -Body '{"value":"super-secret-value","contentType":"text/plain"}' -ContentType "application/json"
Invoke-RestMethod "http://127.0.0.1:4580/vaultlocal/secrets/app-secret"
```

### 6. Service Bus Data-Plane

Service Bus runs on `http://127.0.0.1:4581`.

Create a namespace, queue, topic, and subscription:

```powershell
Invoke-RestMethod -Method Post "http://127.0.0.1:4581/namespaces" -Body '{"name":"local-messaging"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4581/namespaces/local-messaging/queues" -Body '{"name":"jobs"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4581/namespaces/local-messaging/topics" -Body '{"name":"events"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4581/namespaces/local-messaging/topics/events/subscriptions" -Body '{"name":"worker-a"}' -ContentType "application/json"
```

Send/receive queue messages:

```powershell
Invoke-RestMethod -Method Post "http://127.0.0.1:4581/namespaces/local-messaging/queues/jobs/messages" -Body '{"body":"{\"job\":\"sync\"}"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4581/namespaces/local-messaging/queues/jobs/messages/receive?maxMessages=1&visibilityTimeout=30"
```

Publish/receive topic messages:

```powershell
Invoke-RestMethod -Method Post "http://127.0.0.1:4581/namespaces/local-messaging/topics/events/messages" -Body '{"body":"{\"event\":\"created\"}"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4581/namespaces/local-messaging/topics/events/subscriptions/worker-a/messages/receive?maxMessages=1&visibilityTimeout=30"
```

### 7. App Configuration Data-Plane

App Configuration runs on `http://127.0.0.1:4582`.

Create a config store and manage a key-value:

```powershell
Invoke-RestMethod -Method Post "http://127.0.0.1:4582/stores" -Body '{"name":"tiny-settings"}' -ContentType "application/json"
Invoke-RestMethod -Method Put "http://127.0.0.1:4582/stores/tiny-settings/kv/FeatureX:Enabled?label=prod" -Body '{"value":"true","contentType":"text/plain"}' -ContentType "application/json"
Invoke-RestMethod "http://127.0.0.1:4582/stores/tiny-settings/kv/FeatureX:Enabled?label=prod"
```

### 8. Cosmos DB Data-Plane

Cosmos DB runs on `http://127.0.0.1:4583`.

Create an account, database, container, and document:

```powershell
Invoke-RestMethod -Method Post "http://127.0.0.1:4583/accounts" -Body '{"name":"local-cosmos"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4583/accounts/local-cosmos/dbs" -Body '{"id":"appdb"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4583/accounts/local-cosmos/dbs/appdb/colls" -Body '{"id":"customers","partitionKeyPath":"/tenantId"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4583/accounts/local-cosmos/dbs/appdb/colls/customers/docs" -Body '{"id":"cust-001","partitionKey":"tenant-a","tenantId":"tenant-a","name":"Tiny Cloud"}' -ContentType "application/json"
Invoke-RestMethod "http://127.0.0.1:4583/accounts/local-cosmos/dbs/appdb/colls/customers/docs/cust-001"
```

### 9. Private DNS

Private DNS uses ARM routes on the management endpoint plus a live UDP resolver on `127.0.0.1:4584`.

Create a zone and A record:

```powershell
Invoke-RestMethod -Method Put "http://127.0.0.1:4566/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-local/providers/Microsoft.Network/privateDnsZones/internal.test?api-version=2024-01-01" -Body '{"tags":{"env":"dev"}}' -ContentType "application/json"
Invoke-RestMethod -Method Put "http://127.0.0.1:4566/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-local/providers/Microsoft.Network/privateDnsZones/internal.test/A/api?api-version=2024-01-01" -Body '{"properties":{"TTL":60,"aRecords":[{"ipv4Address":"10.0.0.4"}]}}' -ContentType "application/json"
```

Any DNS client that supports a custom resolver port can then query `api.internal.test` against `127.0.0.1:4584/udp`.

### 10. Metadata And Identity

Inspect local environment metadata:

```powershell
Invoke-RestMethod http://127.0.0.1:4566/metadata/endpoints
```

Request a managed identity token:

```powershell
Invoke-RestMethod `
  -Headers @{ Metadata = "true" } `
  "http://127.0.0.1:4566/metadata/identity/oauth2/token?api-version=2018-02-01&resource=https://management.azure.com/"
```

### 11. Admin Runtime Endpoints

These are TinyCloud-specific runtime controls, not Azure service APIs.

```powershell
Invoke-RestMethod http://127.0.0.1:4566/_admin/healthz
Invoke-RestMethod http://127.0.0.1:4566/_admin/metrics
Invoke-RestMethod http://127.0.0.1:4566/_admin/runtime
Invoke-RestMethod http://127.0.0.1:4566/_admin/services
Invoke-RestMethod -Method Post http://127.0.0.1:4566/_admin/snapshot
Invoke-RestMethod -Method Post http://127.0.0.1:4566/_admin/reset
```

### 12. Event Hubs Data-Plane

Event Hubs runs on `http://127.0.0.1:4585`.

Create a namespace and hub:

```powershell
Invoke-RestMethod -Method Post "http://127.0.0.1:4585/namespaces" -Body '{"name":"local-streaming"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4585/namespaces/local-streaming/hubs" -Body '{"name":"orders"}' -ContentType "application/json"
```

Publish and read events:

```powershell
Invoke-RestMethod -Method Post "http://127.0.0.1:4585/namespaces/local-streaming/hubs/orders/events" -Body '{"body":"{\"event\":\"created\"}","partitionKey":"tenant-a"}' -ContentType "application/json"
Invoke-RestMethod "http://127.0.0.1:4585/namespaces/local-streaming/hubs/orders/events?fromSequenceNumber=1&maxEvents=10"
```

## CLI Integration

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

The built-in `tinycloud` CLI is not an Azure CLI replacement. It is now the local runtime manager plus endpoint, config, and service-control surface for both supported local runtime backends:

- Docker is the default backend when Docker is available locally. `tinycloud start` auto-builds the repo-root `tinycloud-azure` image if needed and then manages the active TinyCloud container for `status`, `logs`, `wait`, `restart`, and `stop`.
- `--backend process` keeps the managed local `tinycloudd` binary workflow available when you want to stay outside Docker.
- `tinycloud start` now defaults to detached startup so it returns control to the shell; use `tinycloud start --attached` when you want the foreground log-streaming path instead.

`tinycloud start` now also accepts LocalStack-style bootstrap inputs for the current local runtime workflow:

- `--backend docker|process`
- `--services ...`
- `--env KEY=VALUE`
- `--publish HOSTPORT:CONTAINERPORT`
- `--volume HOSTPATH:CONTAINERPATH`
- `--network NAME`

The runtime now also honors `TINYCLOUD_SERVICES` so listener startup is explicit instead of implicitly always-on. The current service-selection model accepts either individual services or family aliases:

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

When service selection is in use, `/_admin/runtime`, `/_admin/services`, `tinycloud endpoints`, and metadata discovery now reflect the enabled service set rather than advertising listeners that were never started.

`tinycloud services enable ...` and `tinycloud services disable ...` persist the selected service set under `.tinycloud-runtime\tinycloud.env` so later `tinycloud start`, `tinycloud restart`, and `tinycloud config show` calls reconnect to the same intended local runtime configuration. Because the current runtime backends do not live-toggle listeners, service changes currently require a restart. The human-readable CLI now prints a service-selection summary plus explicit restart guidance, while `--json` output remains stable for automation.

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

Those repo-root wrappers now build through repo-root-relative command package paths, preferring the top-level `cmd\...` entrypoints and falling back to the Azure compatibility paths under `azure\cmd\...` when needed. They cache the built binaries under `.tinycloud-runtime` and default their Go build cache to `tinycloud\.gocache`.

The human-readable terminal UX now follows a more structured LocalStack-style shape:

- interactive `tinycloud start` is the only command that prints the approved TinyCloud ASCII banner
- default `tinycloud start` prints lifecycle steps, a runtime summary, and the next useful follow-up commands, then returns control to the shell
- `tinycloud start --attached` is the explicit foreground mode when you want startup output followed by live logs
- `tinycloud status runtime` and `tinycloud status services` render terminal tables instead of raw key=value lines
- `tinycloud status services` is the runtime-status view, while `tinycloud services list` is the config/catalog inventory view
- `tinycloud config show` renders grouped Runtime, Ports, and Services sections
- `tinycloud endpoints` renders a stable endpoint table
- interactive `tinycloud start` and `tinycloud logs -f` now render known structured TinyCloud runtime/request log lines as terminal sections instead of raw JSON, while unknown lines still fall back to raw output
- status icons such as `✓`, `✗`, and `‼` are used in human-readable output, with color only on the icon glyph itself in interactive terminals
- `--json` output remains banner-free and machine-readable

For the Docker backend, `status runtime` still reports the active TinyCloud container identity and image.

TinyCloud's compatibility direction is intentionally LocalStack-style:

- `tinyterraform` is the TinyCloud analogue to `tflocal`
- `tinyaz` is the planned TinyCloud analogue to `azlocal`
- users should be able to keep using normal Terraform and Azure CLI habits with minimal TinyCloud-specific setup
- both wrappers should invoke the real upstream binaries under the hood rather than reimplementing their command sets
- for officially supported command and resource families, both wrappers target a Model 2 shape: classify the command family, resolve the correct TinyCloud management or service endpoint, and preserve the normal upstream command structure
- parity is defined against an explicitly documented and verified supported subset, not as a blanket promise that every Terraform or Azure CLI command will work unchanged

The goal is to put compatibility behavior in wrappers around familiar tools rather than forcing users onto a custom control surface, while still keeping the compatibility claim bounded to the supported subset TinyCloud actually verifies.

## Terraform Example

The current repo includes a Terraform example for `azurerm_resource_group` under `examples/terraform/resource-group`.

Current status:

- the repo contains a Terraform example, `tinycloud env terraform` output, a first-class launcher at `cmd/tinyterraform`, and a Windows wrapper script at `scripts/tinyterraform.ps1`
- Terraform is required locally; TinyCloud does not bundle it
- the supported local flow is the first-class `tinyterraform` launcher plus the wrapper-backed privileged runtime path, not a raw `terraform apply` against `azurerm`
- the wrapper has been manually verified end to end for `init`, `apply`, and `destroy` against `azurerm_resource_group`
- the current officially supported Terraform compatibility subset is still narrow: the verified `azurerm_resource_group` example plus non-runtime passthrough commands such as `help`, `version`, `login`, `logout`, `console`, and subcommand help
- the roadmap direction is to keep promoting `tinyterraform` toward a first-class Model 2 compatibility command for the officially supported subset, expanding that verified subset over time instead of claiming blanket `azurerm` parity

The provider shape currently used in the repo is:

```hcl
terraform {
  required_version = ">= 1.6.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 4.0"
    }
  }
}

provider "azurerm" {
  features {}

  subscription_id = var.subscription_id
  tenant_id       = var.tenant_id
  resource_provider_registrations = "none"
}

variable "subscription_id" {
  type    = string
  default = "11111111-1111-1111-1111-111111111111"
}

variable "tenant_id" {
  type    = string
  default = "00000000-0000-0000-0000-000000000001"
}

resource "azurerm_resource_group" "example" {
  name     = "tinycloud-rg"
  location = "westus2"

  tags = {
    environment = "local"
    managed_by  = "tinycloud"
  }
}
```

Then print the low-level environment values directly:

```powershell
go run .\cmd\tinycloud env terraform
```

Typical local flow on Windows from `tinycloud\`:

```powershell
$env:GOCACHE="$PWD\.gocache"
go run .\cmd\tinyterraform -- init
go run .\cmd\tinyterraform -- apply -auto-approve
go run .\cmd\tinyterraform -- destroy -auto-approve
```

The older Azure-backed launcher path still works as a compatibility path:

```powershell
$env:GOCACHE="$PWD\.gocache"
go run .\azure\cmd\tinyterraform -- -chdir=.\azure\examples\terraform\resource-group init
```

The repo root now also exposes the wrapper script directly:

```powershell
$env:GOCACHE="$PWD\.gocache"
.\scripts\tinyterraform.ps1 -chdir=.\azure\examples\terraform\resource-group init
```

Equivalent direct wrapper flow:

```powershell
$env:GOCACHE="$PWD\.gocache"
.\scripts\tinyterraform.ps1 init
.\scripts\tinyterraform.ps1 apply -auto-approve
.\scripts\tinyterraform.ps1 destroy -auto-approve
```

`cmd/tinyterraform` is the current first-class launcher entrypoint. On Windows it now owns the local `terraform init` reset/bootstrap path directly in the shared Go command layer, and direct wrapper `init` through either `scripts\tinyterraform.ps1` entrypoint now delegates into that same launcher-owned path instead of keeping a separate inline init implementation in the wrapper. For the broader privileged runtime-routing path, the launcher now also prebuilds the current `tinycloud` helper, resolves the real `terraform` binary, and creates the temporary Terraform provider override in the actual Terraform working directory before invoking `scripts\tinyterraform.ps1`, while the wrapper remains responsible for the remaining privileged behavior: starting TinyCloud when needed, injecting the current Azure CLI compatibility required for supported Terraform flows, temporarily mapping `management.azure.com` to the local TinyCloud HTTPS listener, and removing the mapping on exit. The current Azure CLI compatibility layer is still embedded in that wrapper; the roadmap direction is to split that into a standalone `tinyaz` helper while keeping the same Model 2 supported-subset goal for both wrappers. Commands that actually need TinyCloud runtime routing beyond `init` still require an elevated PowerShell session today; pure local passthrough commands like `terraform help`, `terraform version`, `terraform login`, `terraform logout`, `terraform console`, and subcommand help requests like `terraform apply -help` do not. Terraform global flags such as `-chdir=...` are preserved by the launcher and wrapper so normal CLI invocation patterns continue to work, including PowerShell invocation. Both entrypoints also honor `TERRAFORM_EXE` when you need to point TinyCloud at a specific Terraform binary, and the wrapper preserves Terraform stdout for machine-readable commands like `version -json`.

`tinyterraform init` resets the TinyCloud runtime state before running Terraform init. That keeps emulator state and Terraform state aligned after failed local applies.
`tinyterraform init` uses that local reset/bootstrap path but does not need the HTTPS cert-trust and hosts-file routing that `apply` and `destroy` still require.

For compatibility and repo-layout variation handling, both entrypoints also support explicit path overrides:

- `TINYCLOUD_SOURCE_ROOT` points the wrapper at the TinyCloud source tree it should build and run
- `TINYTERRAFORM_SCRIPT` points the Go launcher at a specific `tinyterraform.ps1` script path
- `TINYTERRAFORM_SCRIPT_RELATIVE_PATH` points the Go launcher at the wrapper script relative to `TINYCLOUD_SOURCE_ROOT`, which defaults to `scripts\tinyterraform.ps1` today
- `TINYCLOUD_MAIN_PACKAGE` points the wrapper at the TinyCloud Go package it should build; the wrappers still accept the older `tinycloud/cmd/tinycloud` form for migration compatibility, but the default repo-root paths now build the top-level `.\cmd\tinycloud` launcher and only fall back to `.\azure\cmd\tinycloud` when needed
- `TINYCLOUD_GO_WORKDIR` points the wrapper at the Go build/workspace directory it should run `go build` from
- `TINYTERRAFORM_RUNTIME_ROOT` points the wrapper at an isolated runtime directory instead of the default `.tinyterraform-runtime`

Those overrides let the wrapper and launcher survive source-tree and workspace variations while the remaining CLI work continues.
The wrapper also now searches upward from its own location for `cmd\tinycloud\main.go`, so a script temporarily relocated under a provider path like `azure\scripts` can still find the real TinyCloud root without requiring `TINYCLOUD_SOURCE_ROOT`.
The repo-root wrapper is now a first-class script at `tinycloud\scripts\tinyterraform.ps1`. It auto-detects the current Azure-backed source tree from the repo root, builds through the repo-root Go workspace, defaults its Go build cache to `tinycloud\.gocache`, resolves the current command package path from the repo root, and keeps the same compatibility behavior without delegating through `azure\scripts\tinyterraform.ps1`.
When you use the repo-root wrapper, its runtime artifacts now default to `tinycloud\.tinyterraform-runtime` unless you override `TINYTERRAFORM_RUNTIME_ROOT`.

Compatibility goal:

- preserve normal `terraform` argument passing and user expectations
- preserve normal Azure CLI habits as much as practical
- invoke real `terraform` and `az` binaries under the hood
- pass through stdout, stderr, and exit codes as closely as practical
- keep TinyCloud-specific wiring in the wrapper layer instead of in user Terraform code
- for officially supported command/resource families, preserve normal upstream command structure and use wrapper-side endpoint routing rather than requiring manual helper flows
- expand parity through a documented and verified supported subset rather than promising every upstream Terraform or Azure CLI workflow unchanged

## Configuration

### Core Environment Variables

| Variable | Default | Purpose |
| --- | --- | --- |
| `TINYCLOUD_DATA_ROOT` | Windows: `.\data` non-Windows: `~/.tinycloud/data` | writable local state root |
| `TINYCLOUD_LISTEN_HOST` | Windows: `127.0.0.1`, non-Windows: `0.0.0.0` | bind host |
| `TINYCLOUD_ADVERTISE_HOST` | `127.0.0.1` | host used in advertised URLs |
| `TINYCLOUD_MGMT_HTTP_PORT` | `4566` | management listener |
| `TINYCLOUD_MGMT_HTTPS_PORT` | `4567` | management HTTPS listener |
| `TINYCLOUD_BLOB_PORT` | `4577` | Blob listener |
| `TINYCLOUD_QUEUE_PORT` | `4578` | Queue Storage listener |
| `TINYCLOUD_TABLE_PORT` | `4579` | Table Storage listener |
| `TINYCLOUD_KEYVAULT_PORT` | `4580` | Key Vault listener |
| `TINYCLOUD_SERVICEBUS_PORT` | `4581` | Service Bus listener |
| `TINYCLOUD_APPCONFIG_PORT` | `4582` | App Configuration listener |
| `TINYCLOUD_COSMOS_PORT` | `4583` | Cosmos DB listener |
| `TINYCLOUD_DNS_PORT` | `4584` | private DNS UDP listener |
| `TINYCLOUD_EVENTHUBS_PORT` | `4585` | Event Hubs listener |
| `TINYCLOUD_SERVICES` | empty = all services | comma-separated service or family selection such as `management`, `storage`, `messaging`, or `management,storage` |
| `TINYCLOUD_TENANT_ID` | `00000000-0000-0000-0000-000000000001` | default tenant ID |
| `TINYCLOUD_SUBSCRIPTION_ID` | `11111111-1111-1111-1111-111111111111` | default subscription ID |
| `TINYCLOUD_TOKEN_ISSUER` | empty | optional token issuer override |
| `TINYCLOUD_TOKEN_AUDIENCE` | `https://management.azure.com/` | default token audience |
| `TINYCLOUD_TOKEN_SUBJECT` | `tinycloud-local-user` | token subject |
| `TINYCLOUD_TOKEN_KEY` | `tinycloud-dev-signing-key` | local JWT signing key |

### Persistence

- State is stored in SQLite at `state.db` under `TINYCLOUD_DATA_ROOT`.
- Snapshots default to `TINYCLOUD_DATA_ROOT\tinycloud.snapshot.json` on Windows or the equivalent path on other platforms.
- Local runs are intentionally unprivileged; the default non-Windows path is under the user home directory.
- Container runs use `/var/lib/tinycloud`.
- Managed local CLI runtime metadata, daemon logs, and persisted CLI service configuration live under `.tinycloud-runtime`.

## Local And Docker Smoke Tests

### Local

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

### Docker

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

## How TinyCloud Compares

This is the practical comparison for current use, not a marketing claim. The point here is where TinyCloud fits in the broader local cloud-emulator landscape.

| Tool | Cloud focus | Product shape | Strength | Tradeoff | Best fit |
| --- | --- | --- | --- | --- | --- |
| TinyCloud | Azure | focused local cloud emulator | combines ARM-style control plane, identity metadata, storage, secrets, and messaging in one small runtime | Azure coverage is still intentionally narrow | testing Azure workflows that need ARM plus several real data-plane services |
| Azurite | Azure Storage | storage emulator | mature Blob/Queue/Table emulation from Microsoft | no ARM, no identity, no broader Azure control plane | storage-only local development |
| LocalStack | AWS | broad local cloud platform | large AWS surface area and established local-cloud workflow patterns | AWS-focused rather than Azure-focused | teams standardizing on AWS local emulation |
| MiniStack | AWS | lightweight local cloud platform | fast, small-footprint AWS emulator with broad service ambitions | AWS-focused rather than Azure-focused | developers who want a lighter AWS local-cloud setup |

### Interpretation

- TinyCloud is closer in spirit to LocalStack and MiniStack than to Azurite: it aims to emulate a cloud environment, not just a single storage service.
- Azurite is the better choice when you only need Azure Storage and want broader storage coverage today.
- TinyCloud is the better fit when you need Azure-style resource provisioning, metadata/identity endpoints, and multiple local data-plane services together in one runtime.
- LocalStack and MiniStack are relevant peers because they define the broader local-cloud developer experience category, even though they target AWS instead of Azure.

## Current Limitations

- Deployment template execution is intentionally narrow; only a small static subset is implemented today
- Private DNS uses UDP on a non-default port (`4584`) by default, so standard system DNS tools that assume port `53` need an explicit custom resolver configuration
- Not a full Azure CLI, Terraform-provider, or SDK parity environment; compatibility is still defined by a narrow verified supported subset
- `tinyterraform` still relies on a Windows-specific wrapper beneath the top-level launcher for runtime-routed flows
- No standalone `tinyaz` helper yet; the Azure CLI compatibility layer currently lives inside `tinyterraform.ps1`

## Examples

- Terraform resource group example: `examples/terraform/resource-group`
- Pulumi environment notes: `examples/pulumi`

## Comparison Sources

The comparison notes above are based on current upstream docs/homepages:

- LocalStack docs: https://docs.localstack.cloud/getting-started/installation/
- LocalStack overview/docs: https://docs.localstack.cloud/aws/enterprise/kubernetes/
- Azurite docs: https://learn.microsoft.com/en-us/azure/storage/common/storage-use-azurite
- Azurite + Storage Explorer docs: https://learn.microsoft.com/en-us/azure/storage/common/storage-explorer-emulators
- MiniStack homepage: https://ministack.org/
