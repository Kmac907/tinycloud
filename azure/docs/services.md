# Services

This page collects direct interaction examples for the current TinyCloud emulation surface.

## ARM Control Plane

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

## Blob Storage Data-Plane

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

## Queue Storage Data-Plane

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

## Table Storage Data-Plane

Table Storage runs on `http://127.0.0.1:4579`.

Create a table and upsert an entity:

```powershell
Invoke-RestMethod -Method Post "http://127.0.0.1:4579/storelocal/Tables" -Body '{"name":"customers"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4579/storelocal/customers" -Body '{"partitionKey":"retail","rowKey":"cust-001","properties":{"Name":"Tiny Cloud"}}' -ContentType "application/json"
```

## Key Vault Secrets Data-Plane

Key Vault secrets run on `http://127.0.0.1:4580`.

Set and read a secret:

```powershell
Invoke-RestMethod -Method Put "http://127.0.0.1:4580/vaultlocal/secrets/app-secret" -Body '{"value":"super-secret-value","contentType":"text/plain"}' -ContentType "application/json"
Invoke-RestMethod "http://127.0.0.1:4580/vaultlocal/secrets/app-secret"
```

## Service Bus Data-Plane

Service Bus runs on `http://127.0.0.1:4581`.

Create a namespace, queue, topic, and subscription:

```powershell
Invoke-RestMethod -Method Post "http://127.0.0.1:4581/namespaces" -Body '{"name":"local-messaging"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4581/namespaces/local-messaging/queues" -Body '{"name":"jobs"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4581/namespaces/local-messaging/topics" -Body '{"name":"events"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4581/namespaces/local-messaging/topics/events/subscriptions" -Body '{"name":"worker-a"}' -ContentType "application/json"
```

Send and receive queue messages:

```powershell
Invoke-RestMethod -Method Post "http://127.0.0.1:4581/namespaces/local-messaging/queues/jobs/messages" -Body '{"body":"{\"job\":\"sync\"}"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4581/namespaces/local-messaging/queues/jobs/messages/receive?maxMessages=1&visibilityTimeout=30"
```

Publish and receive topic messages:

```powershell
Invoke-RestMethod -Method Post "http://127.0.0.1:4581/namespaces/local-messaging/topics/events/messages" -Body '{"body":"{\"event\":\"created\"}"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4581/namespaces/local-messaging/topics/events/subscriptions/worker-a/messages/receive?maxMessages=1&visibilityTimeout=30"
```

## App Configuration Data-Plane

App Configuration runs on `http://127.0.0.1:4582`.

Create a config store and manage a key-value:

```powershell
Invoke-RestMethod -Method Post "http://127.0.0.1:4582/stores" -Body '{"name":"tiny-settings"}' -ContentType "application/json"
Invoke-RestMethod -Method Put "http://127.0.0.1:4582/stores/tiny-settings/kv/FeatureX:Enabled?label=prod" -Body '{"value":"true","contentType":"text/plain"}' -ContentType "application/json"
Invoke-RestMethod "http://127.0.0.1:4582/stores/tiny-settings/kv/FeatureX:Enabled?label=prod"
```

## Cosmos DB Data-Plane

Cosmos DB runs on `http://127.0.0.1:4583`.

Create an account, database, container, and document:

```powershell
Invoke-RestMethod -Method Post "http://127.0.0.1:4583/accounts" -Body '{"name":"local-cosmos"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4583/accounts/local-cosmos/dbs" -Body '{"id":"appdb"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4583/accounts/local-cosmos/dbs/appdb/colls" -Body '{"id":"customers","partitionKeyPath":"/tenantId"}' -ContentType "application/json"
Invoke-RestMethod -Method Post "http://127.0.0.1:4583/accounts/local-cosmos/dbs/appdb/colls/customers/docs" -Body '{"id":"cust-001","partitionKey":"tenant-a","tenantId":"tenant-a","name":"Tiny Cloud"}' -ContentType "application/json"
Invoke-RestMethod "http://127.0.0.1:4583/accounts/local-cosmos/dbs/appdb/colls/customers/docs/cust-001"
```

## Private DNS

Private DNS uses ARM routes on the management endpoint plus a live UDP resolver on `127.0.0.1:4584`.

Create a zone and A record:

```powershell
Invoke-RestMethod -Method Put "http://127.0.0.1:4566/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-local/providers/Microsoft.Network/privateDnsZones/internal.test?api-version=2024-01-01" -Body '{"tags":{"env":"dev"}}' -ContentType "application/json"
Invoke-RestMethod -Method Put "http://127.0.0.1:4566/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/rg-local/providers/Microsoft.Network/privateDnsZones/internal.test/A/api?api-version=2024-01-01" -Body '{"properties":{"TTL":60,"aRecords":[{"ipv4Address":"10.0.0.4"}]}}' -ContentType "application/json"
```

Any DNS client that supports a custom resolver port can then query `api.internal.test` against `127.0.0.1:4584/udp`.

## Metadata And Identity

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

## Admin Runtime Endpoints

These are TinyCloud-specific runtime controls, not Azure service APIs.

```powershell
Invoke-RestMethod http://127.0.0.1:4566/_admin/healthz
Invoke-RestMethod http://127.0.0.1:4566/_admin/metrics
Invoke-RestMethod http://127.0.0.1:4566/_admin/runtime
Invoke-RestMethod http://127.0.0.1:4566/_admin/services
Invoke-RestMethod -Method Post http://127.0.0.1:4566/_admin/snapshot
Invoke-RestMethod -Method Post http://127.0.0.1:4566/_admin/reset
```

## Event Hubs Data-Plane

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
