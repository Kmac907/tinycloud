# Overview

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

## Scope Notes

- The current emulation scope is tracked as 18 emulator areas.
- That is broader than the runtime service-selection surface.
- The runtime service-selection model currently exposes 10 listener/service names:
  - `management`
  - `blob`
  - `queue`
  - `table`
  - `keyVault`
  - `serviceBus`
  - `appConfig`
  - `cosmos`
  - `dns`
  - `eventHubs`
