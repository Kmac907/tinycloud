<p align="center">
  <img src="./assets/logo.png" width="300" />
</p>

<h1 align="center">TinyCloud Azure Emulator</h1>

<p align="center">
  <a href="#"><img src="https://img.shields.io/badge/Go-1.26-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go 1.26" /></a>
  <a href="#"><img src="https://img.shields.io/badge/Docker-Single%20Container-2496ED?style=for-the-badge&logo=docker&logoColor=white" alt="Docker Single Container" /></a>
  <a href="#current-emulation-scope"><img src="https://img.shields.io/badge/Azure-ARM%20%2B%20Blob-0078D4?style=for-the-badge&logo=microsoftazure&logoColor=white" alt="Azure ARM and Blob" /></a>
  <a href="https://x.com/Kyle_Andrew_Mac"><img src="https://img.shields.io/badge/X-@Kyle_Andrew_Mac-000000?style=for-the-badge&logo=x&logoColor=white" alt="X Kyle Andrew Mac" /></a>
</p>

<p align="center"><sub>Develop and test Azure-facing applications locally with a focused emulator for ARM, identity, and Blob workflows.</sub></p>

TinyCloud is a local Azure-compatible emulator written in Go and packaged as a single container. It provides a compact Azure development environment for local iteration and CI by combining:

- Azure Resource Manager support for tenants, subscriptions, providers, resource groups, storage accounts, and Key Vault resources
- Azure-style async operation polling for supported control-plane writes
- metadata, OAuth, and minimal IMDS-style managed identity endpoints
- real Blob Storage container and blob behavior on a dedicated service port
- admin/runtime endpoints for health, metrics, reset, snapshot, and seed

TinyCloud is designed for targeted local Azure workflow testing, not full Azure parity.

## Current Emulation Scope

Current status across the listed emulator areas:

- `8` implemented
- `1` partial
- `3` not implemented yet

### Support Levels

| Area | Current level | Notes |
| --- | --- | --- |
| ARM tenants/subscriptions/providers | Implemented | Includes provider registration records and tenant listing |
| ARM resource groups | Implemented | CRUD with Azure-style shapes and async headers |
| ARM storage accounts | Implemented | CRUD with Blob endpoint advertisement |
| ARM Key Vault resources | Implemented | CRUD for `Microsoft.KeyVault/vaults`; no secrets data-plane yet |
| ARM deployments | Partial | Deployment records and async failure/status are implemented; template execution is not |
| Blob data-plane | Implemented | Containers, upload/download/list/delete, `HEAD`, compatibility headers |
| Managed identity and token endpoints | Implemented | Minimal IMDS-style behavior and signed local JWTs |
| Admin/runtime endpoints | Implemented | Health, metrics, reset, snapshot, seed |
| Key Vault secrets data-plane | Not implemented | ARM resource exists, secrets API does not |
| Service Bus | Not implemented | No ARM or queue/message data-plane yet |
| Queue Storage | Not implemented | Port reserved only |
| Table Storage | Not implemented | Port reserved only |

### What Is Actually Emulated Today

- Azure Resource Manager:
  - `GET /tenants`
  - `GET /subscriptions`
  - `GET /providers`
  - provider registration
  - resource group CRUD
  - storage account CRUD
  - Key Vault resource CRUD
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
- Admin/runtime:
  - `/_admin/healthz`
  - `/_admin/metrics`
  - `/_admin/reset`
  - `/_admin/snapshot`
  - `/_admin/seed`

## Ports

Only two listeners are active today. The remaining service ports are reserved in config for future providers.

| Port | Status | Purpose |
| --- | --- | --- |
| `4566` | Active | management endpoint: ARM, metadata, identity, OAuth, admin |
| `4567` | Reserved | management HTTPS URL is advertised/configurable, but no TLS listener is started yet |
| `4577` | Active | Blob data-plane |
| `4578` | Reserved | Queue Storage placeholder |
| `4579` | Reserved | Table Storage placeholder |
| `4580` | Reserved | Key Vault data-plane placeholder |
| `4581` | Reserved | Service Bus placeholder |

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

Resource-group, storage-account, and Key Vault writes return `202 Accepted` plus `Azure-AsyncOperation`, `Location`, and `Retry-After`.

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

### 3. Metadata And Identity

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

### 4. Admin Runtime Endpoints

These are TinyCloud-specific runtime controls, not Azure service APIs.

```powershell
Invoke-RestMethod http://127.0.0.1:4566/_admin/healthz
Invoke-RestMethod http://127.0.0.1:4566/_admin/metrics
Invoke-RestMethod -Method Post http://127.0.0.1:4566/_admin/snapshot
Invoke-RestMethod -Method Post http://127.0.0.1:4566/_admin/reset
```

## CLI Integration

The built-in CLI manages the local runtime and prints environment settings for external tools:

```powershell
$env:TINYCLOUD_DATA_ROOT="$PWD\data"

go run .\cmd\tinycloud init
go run .\cmd\tinycloud status
go run .\cmd\tinycloud endpoints
go run .\cmd\tinycloud env terraform
go run .\cmd\tinycloud env pulumi
go run .\cmd\tinycloud start
```

The CLI is not an Azure CLI replacement. It is a local runtime helper plus endpoint/config printer.

## Terraform Example

The current repo includes a Terraform example for `azurerm_resource_group` under `examples/terraform/resource-group`.

Current status:

- the repo contains a Terraform example and `tinycloud env terraform` output for it
- Terraform is required locally; TinyCloud does not bundle it
- this repo does not currently include an automated Terraform integration test

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

Then print the local environment values directly:

```powershell
go run .\cmd\tinycloud env terraform
```

Typical local flow:

```powershell
$env:GOCACHE="$PWD\.gocache"
go run .\cmd\tinycloudd
```

In another terminal:

```powershell
$env:GOCACHE="$PWD\.gocache"
go run .\cmd\tinycloud env terraform
```

Export the printed values into your shell, then from `examples/terraform/resource-group` run:

```powershell
terraform init
terraform apply
```

The example material in this repo is under `examples/terraform/resource-group`, but successful `terraform apply` should be treated as environment-dependent until verified on a machine with Terraform installed.

## Configuration

### Core Environment Variables

| Variable | Default | Purpose |
| --- | --- | --- |
| `TINYCLOUD_DATA_ROOT` | Windows: `.\data` non-Windows: `~/.tinycloud/data` | writable local state root |
| `TINYCLOUD_LISTEN_HOST` | Windows: `127.0.0.1`, non-Windows: `0.0.0.0` | bind host |
| `TINYCLOUD_ADVERTISE_HOST` | `127.0.0.1` | host used in advertised URLs |
| `TINYCLOUD_MGMT_HTTP_PORT` | `4566` | management listener |
| `TINYCLOUD_MGMT_HTTPS_PORT` | `4567` | advertised HTTPS management port |
| `TINYCLOUD_BLOB_PORT` | `4577` | Blob listener |
| `TINYCLOUD_QUEUE_PORT` | `4578` | reserved Queue Storage port |
| `TINYCLOUD_TABLE_PORT` | `4579` | reserved Table Storage port |
| `TINYCLOUD_KEYVAULT_PORT` | `4580` | reserved Key Vault port |
| `TINYCLOUD_SERVICEBUS_PORT` | `4581` | reserved Service Bus port |
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

## Local And Docker Smoke Tests

### Local

```powershell
$env:TINYCLOUD_DATA_ROOT="$PWD\data"
go run .\cmd\tinycloud init
go run .\cmd\tinycloudd
```

In another terminal:

```powershell
Invoke-RestMethod http://127.0.0.1:4566/metadata/endpoints
Invoke-RestMethod http://127.0.0.1:4566/tenants?api-version=2024-01-01
Invoke-RestMethod http://127.0.0.1:4566/subscriptions?api-version=2024-01-01
Invoke-WebRequest -Method Put "http://127.0.0.1:4577/devstoreaccount1/docs?restype=container"
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
  -v "${PWD}\data:/var/lib/tinycloud" `
  tinycloud-azure
```

## How TinyCloud Compares

This is the practical comparison for current use, not a marketing claim. The point here is where TinyCloud fits in the broader local cloud-emulator landscape.

| Tool | Cloud focus | Product shape | Strength | Tradeoff | Best fit |
| --- | --- | --- | --- | --- | --- |
| TinyCloud | Azure | focused local cloud emulator | combines ARM-style control plane, identity metadata, and real Blob behavior in one small runtime | Azure coverage is still intentionally narrow | testing Azure workflows that need ARM plus at least one real data-plane service |
| Azurite | Azure Storage | storage emulator | mature Blob/Queue/Table emulation from Microsoft | no ARM, no identity, no broader Azure control plane | storage-only local development |
| LocalStack | AWS | broad local cloud platform | large AWS surface area and established local-cloud workflow patterns | AWS-focused rather than Azure-focused | teams standardizing on AWS local emulation |
| MiniStack | AWS | lightweight local cloud platform | fast, small-footprint AWS emulator with broad service ambitions | AWS-focused rather than Azure-focused | developers who want a lighter AWS local-cloud setup |

### Interpretation

- TinyCloud is closer in spirit to LocalStack and MiniStack than to Azurite: it aims to emulate a cloud environment, not just a single storage service.
- Azurite is the better choice when you only need Azure Storage and want broader storage coverage today.
- TinyCloud is the better fit when you need Azure-style resource provisioning, metadata/identity endpoints, and Blob behavior together in one local runtime.
- LocalStack and MiniStack are relevant peers because they define the broader local-cloud developer experience category, even though they target AWS instead of Azure.

## Current Limitations

- No Key Vault secrets data-plane yet
- No Service Bus emulation yet
- No Queue/Table storage emulation yet
- No deployment template execution; deployments are tracked honestly as unsupported
- No management TLS listener yet, even though an HTTPS URL can be advertised
- Not a full Azure CLI or full SDK parity environment

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
