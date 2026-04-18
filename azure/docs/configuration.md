# Configuration

## Core Environment Variables

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

## Persistence

- State is stored in SQLite at `state.db` under `TINYCLOUD_DATA_ROOT`.
- Snapshots default to `TINYCLOUD_DATA_ROOT\tinycloud.snapshot.json` on Windows or the equivalent path on other platforms.
- Local runs are intentionally unprivileged; the default non-Windows path is under the user home directory.
- Container runs use `/var/lib/tinycloud`.
- Managed local CLI runtime metadata, daemon logs, and persisted CLI service configuration live under `.tinycloud-runtime`.
