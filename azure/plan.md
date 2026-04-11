# TinyCloud Plan

## 1. Product goal

TinyCloud is a local Azure-compatible emulator written in Go and packaged as a single binary inside a single Docker container. It provides an ARM-first control plane, a metadata endpoint, minimal identity support, and separate service endpoints for a small set of data-plane services to support local development and CI.

## 2. Scope

### v1 must ship
- ARM control plane for subscriptions, tenants, provider registration, and resource groups.
- Resource ID parsing and api-version parsing.
- ARM long-running operations with Azure-style polling headers.
- Metadata endpoint and minimal IMDS-style identity endpoint.
- SQLite-backed persistence.
- At least one working data-plane service with real behavior.
- Admin runtime endpoints for health, metrics, reset, snapshot, and seed.
- Docker-based local runtime with no TODO placeholders in core flows.

### v1 should ship if feasible
- ARM deployments with a limited template subset.
- Microsoft.Storage provider routing.
- Microsoft.KeyVault provider routing.
- Blob service data-plane compatibility.
- Terraform example for `azurerm_resource_group`.
- Pulumi environment-based endpoint configuration.
- SDK-friendly endpoint discovery responses.
- LocalStack-style compatibility wrappers:
  - `tinyterraform` as the TinyCloud analogue to `tflocal`
  - `tinyaz` as the TinyCloud analogue to `azlocal`

### Out of scope for v1
- Full Azure parity.
- Full Entra ID.
- Full RBAC and policy.
- Full service coverage across Azure.

## 3. Non-goals

- Not a full Azure replacement.
- Not a production cloud.
- Not a complete implementation of every Azure SDK feature.
- Not a general-purpose authentication authority.

## 4. Architecture

### Runtime model
- Single Go process.
- Single container.
- Multiple listening ports are allowed.
- One management endpoint acts as the ARM and metadata front door.
- Separate local ports expose each data-plane service.

### Ports
- Management: 4566 HTTP, 4567 HTTPS.
- Blob: 4577.
- Queue: 4578.
- Table: 4579.
- Key Vault: 4580.
- Service Bus: 4581.
- App Configuration: 4582.
- Cosmos DB: 4583.
- Private DNS: 4584 UDP.
- Event Hubs: 4585.

### Request routing
- The management endpoint handles:
  - `/metadata/endpoints`
  - `/metadata/identity`
  - `/oauth/token`
  - `/subscriptions/*`
  - `/providers/*`
  - `/resourceGroups/*`
  - `/deployments/*`
  - `/_admin/*`
- Data-plane requests go directly to service endpoints.
- Do not route data-plane traffic through a single generic proxy.

### Repo layout target
- The current working implementation lives under the Azure emulator tree, but the long-term command surface must not remain trapped under `tinycloud\azure`.
- The target multi-cloud repo layout should be:
  - `tinycloud\cmd\tinycloud`
  - `tinycloud\cmd\tinyterraform`
  - `tinycloud\cmd\tinyaz`
  - `tinycloud\azure\...`
  - `tinycloud\aws\...` if an AWS emulator is added later
- `tinycloud\cmd` should hold cloud-agnostic product entrypoints.
- `tinycloud\azure\...` should hold Azure-specific emulator implementation.
- Shared CLI/runtime code should live outside provider-specific trees so Azure is one backend, not the repository root.

### Repo layout constraint
- Today, the active Go module root and Docker build context are the current `tinycloud\azure` tree.
- Moving command entrypoints to `tinycloud\cmd` is therefore not just a folder move; it requires making the top-level `tinycloud` directory a valid build root.
- That migration must explicitly cover:
  - Go module or workspace root migration
  - Docker build-context migration
  - wrapper/build script path migration
  - documentation and example command-path migration
  - keeping the Azure emulator buildable throughout the transition

## 5. Control plane

### Core ARM features
- Parse resource IDs into subscription, resource group, provider, type, and name segments.
- Enforce `api-version` presence and basic validation.
- Return Azure-style JSON errors.
- Implement `Microsoft.Resources/resourceGroups` CRUD.
- Implement provider registration records for selected namespaces.
- Implement ARM LROs using `Azure-AsyncOperation` and `Location` headers.

### Resource groups
- Create or update a resource group.
- Get a resource group.
- List resource groups for a subscription.
- Delete a resource group.
- Persist `location`, `tags`, `managedBy`, timestamps, and provisioning state.

### Deployments
- Support a minimal deployment record and async status tracking.
- Support a small template subset only if it can be implemented without stubs.
- If template execution is not ready, return a clear unsupported operation error rather than a fake success.

## 6. Identity and metadata

### Metadata endpoint
- Expose a custom cloud environment document.
- Return management and data-plane endpoints used by SDKs and tools.

### Token issuer
- Expose a mock OAuth2/OIDC token endpoint.
- Issue signed JWTs with configurable issuer, audience, tenant, and subject.
- Support only the claims needed for local SDK auth.

### IMDS
- Support minimal instance metadata and managed identity token retrieval.
- Require metadata request headers consistent with Azure-style clients where practical.
- Return stable JSON shapes for SDK compatibility.

## 7. Data-plane services

### Required v1 service
Implement Blob storage first if possible, because it has the broadest ecosystem support and aligns well with Azure local emulation patterns. If Blob is chosen, implement containers, blobs, and basic upload/download/list behavior.

### Optional v1 services
- Queue storage.
- Table storage.
- Key Vault secrets.
- Service Bus queues.

### Endpoint strategy
- Each service gets its own port and host base URL.
- ARM responses must advertise these service endpoints where the resource type expects them.
- Preserve endpoint consistency across metadata, SDK discovery, and CLI examples.

## 8. State and persistence

### Backends
- SQLite is the default persistent backend.
- Memory backend is test-only.
- Filesystem backend is optional later.
- Postgres is out of scope for v1.

### Stored entities
- Tenants.
- Subscriptions.
- Resource groups.
- Providers.
- ARM resources.
- Async operations.
- Storage namespaces and objects.
- Key Vault secrets if implemented.
- Service Bus queues and messages if implemented.

### Data directory
- In the container image, use `/var/lib/tinycloud` as the default data root.
- For local developer runs, use an unprivileged per-user or workspace-local data root.
- SQLite file default inside the container: `/var/lib/tinycloud/state.db`.

## 9. Middleware

Apply middleware in this order:
- Request ID.
- Structured JSON logging.
- Panic recovery.
- CORS.
- Azure header normalization.
- API version parsing.

## 10. Internal package layout

- `cmd/tinycloudd`
- `cmd/tinycloud`
- `internal/api/arm`
- `internal/api/admin`
- `internal/auth`
- `internal/metadata`
- `internal/identity`
- `internal/core/resourceid`
- `internal/core/apiversion`
- `internal/core/async`
- `internal/providers/storage`
- `internal/providers/queue`
- `internal/providers/table`
- `internal/providers/keyvault`
- `internal/providers/servicebus`
- `internal/providers/appconfig`
- `internal/providers/cosmos`
- `internal/providers/dns`
- `internal/providers/eventhubs`
- `internal/httpx`
- `internal/state`
- `internal/telemetry`

## 11. CLI

### Commands
- `start`
- `stop`
- `restart`
- `logs`
- `wait`
- `init`
- `reset`
- `snapshot create`
- `snapshot restore`
- `seed apply`
- `status`
- `status runtime`
- `status services`
- `endpoints`
- `config show`
- `config validate`
- `services list`
- `services enable`
- `services disable`
- `env terraform`
- `env pulumi`
- `tinyterraform`
- `tinyaz`

### CLI behavior
- `tinycloud` should be the main cohesive product CLI, analogous to the `localstack` CLI.
- `start` launches TinyCloud locally and should support both attached and detached runtime operation.
- `start` should default to detached startup, print the startup summary and next commands, and return control to the shell.
- `start --attached` should be the explicit foreground/log-streaming mode.
- `start` should be able to initialize the supported TinyCloud runtime backend for the environment, including the primary container-based local workflow.
- `start` should support LocalStack-style runtime bootstrap flags for environment variables, port publishing, volume mounts, and backend/network selection where relevant.
- `stop` stops the active TinyCloud runtime started by the CLI.
- `restart` restarts the active TinyCloud runtime while preserving the selected runtime configuration.
- `logs` shows or follows logs for the active TinyCloud runtime.
- `wait` blocks until the active TinyCloud runtime is healthy and ready to serve requests.
- `init` creates required directories and default configuration.
- `reset` clears persisted state.
- `snapshot create` writes a snapshot file.
- `snapshot restore` loads a snapshot file.
- `status runtime` prints runtime/backend/container/process status for the active TinyCloud instance.
- `status services` prints per-service readiness and enablement state for management and data-plane services.
- `endpoints` prints all local URLs.
- `config show` prints the effective TinyCloud runtime configuration.
- `config validate` validates the current TinyCloud runtime configuration and startup prerequisites.
- `services list` prints the configured and currently active service set.
- `services enable` and `services disable` should let users turn service families on and off through the supported TinyCloud configuration model. If a restart is required, the CLI should say so explicitly and offer the next command to run.
- `env terraform` prints Terraform variables and provider settings.
- `env pulumi` prints Pulumi-compatible environment settings.
- `tinyterraform` wraps standard Terraform with TinyCloud compatibility behavior, should invoke the real `terraform` binary, and should preserve normal Terraform argument passing as closely as practical.
- `tinyaz` wraps standard Azure CLI usage with TinyCloud compatibility behavior, should invoke the real `az` binary, and should preserve normal `az` argument passing as closely as practical.
- Both wrappers should pass through stdout, stderr, and exit codes as closely as practical.
- Compatibility logic should live in the wrapper layer when possible so user Terraform and Azure CLI workflows stay close to their normal cloud equivalents.

### CLI product structure
- The LocalStack analogue should be:
  - `tinycloud` as the main product/runtime CLI, analogous to `localstack`
  - `tinyterraform` as the Terraform wrapper, analogous to `tflocal`
  - `tinyaz` as the Azure CLI wrapper, analogous to `azlocal`
- `tinycloud` should own lifecycle, status, diagnostics, endpoint discovery, configuration, state helpers, and wrapper discovery/orchestration.
- `tinycloud` should discover and manage the active TinyCloud runtime so `status`, `logs`, `stop`, and `wait` work against the same instance with minimal manual wiring.
- `tinycloud start` should be able to launch the runtime through the supported backend for the environment rather than assuming only a direct foreground Go server process.
- The default local-developer product flow should be container-oriented, with the CLI responsible for initializing and managing the TinyCloud container/runtime in the same way the LocalStack CLI manages its local runtime.
- The CLI should keep enough runtime metadata to reconnect later for `status`, `logs`, `wait`, `stop`, and `restart` without requiring the user to manually pass process IDs, ports, or container names.
- `tinyterraform` and `tinyaz` should remain thin compatibility wrappers around real upstream tools, not bespoke reimplementations of Terraform or Azure CLI.

### Runtime and service model
- TinyCloud service activation should be an explicit CLI/config concern rather than an accidental byproduct of whatever listeners happen to be compiled into the binary.
- TinyCloud should support a startup-time service-selection model analogous to LocalStack service selection.
- The CLI/config surface should include a service-selection input such as `TINYCLOUD_SERVICES` and/or `tinycloud start --services ...`.
- The service-selection model should cover at least:
  - management/control-plane services
  - storage family services
  - secrets/config services
  - messaging/event services
  - networking services
- `status services` should distinguish:
  - configured services
  - enabled services
  - healthy services
  - disabled services
  - failed-to-start services
- If TinyCloud cannot truly enable or disable a service live at runtime, the CLI should still expose the operation through config plus restart and should be explicit about that UX.

### Terminal UX
- The `tinycloud` CLI should present a polished terminal UX comparable in spirit to the LocalStack CLI rather than only dumping raw process output.
- Default interactive output should be human-readable and should explain what the CLI is doing during startup, shutdown, waiting, and failure cases.
- The approved TinyCloud brand banner may appear only on `tinycloud start`, and only in interactive human-readable terminal output.
- The approved interactive startup banner is:

  ```text
     __  _                  __                __
    / /_(_)___  __  _______/ /___  __  ______/ /
   / __/ / __ \/ / / / ___/ / __ \/ / / / __  / 
  / /_/ / / / / /_/ / /__/ / /_/ / /_/ / /_/ /  
  \__/_/_/ /_/\__, /\___/_/\____/\__,_/\__,_/   
             /____/
  ```

- Other commands such as `status`, `config`, `endpoints`, `stop`, `restart`, `wait`, and `logs` should not print the brand banner by default.
- `start` should default to detached mode and should print the runtime identifier, backend, selected services, exposed endpoints, and the next useful commands such as `tinycloud status`, `tinycloud logs -f`, and `tinycloud stop`.
- `start --attached` should print a concise runtime summary before streaming logs.
- `status runtime`, `status services`, `config show`, and `endpoints` should have stable human-readable output by default and support machine-readable output formats such as JSON.
- `status services` should render clearly in the terminal, for example as a compact table with service name, enabled state, health state, endpoint, and notes.
- Human-readable `status services` output should use a table by default.
- Human-readable `status runtime` output should use a stable summary table or grouped key/value layout by default.
- Human-readable `endpoints` output should use a stable table rather than an arbitrary key-order dump.
- Human-readable `config show` output should use grouped sections such as runtime, ports, and services rather than a flat unstructured dump.
- `logs` should support tailing/following output, and interactive log output should preserve timestamps and service/source context where practical.
- when the runtime emits known structured TinyCloud JSON logs, interactive log output should present them as readable terminal sections while preserving raw fallback for unknown lines and non-interactive output
- Progress indicators, colors, and spinners should be used only when writing to an interactive terminal and should degrade cleanly in non-interactive output.
- Human-readable lifecycle and status output may use compact status icons where helpful, such as:
  - green `✓` for success/ready/running
  - red `✗` for failure
  - yellow `‼` for warnings or required follow-up such as restart-required state
- color only the icon glyph itself; do not color the following label text
- Colors must be emitted only for interactive terminal output; non-interactive output must stay readable without ANSI color support.
- Non-interactive and machine-readable modes should avoid banners, progress animations, or mixed diagnostic chatter that would break piping and automation.
- Error messages should be actionable: they should identify the failing service/runtime step and suggest the next command or fix.

### Full `tinycloud` CLI target
- The first-class `tinycloud` CLI should eventually cover:
  - runtime lifecycle: `start`, `stop`, `restart`, `status`, `logs`, `wait`
  - state lifecycle: `init`, `reset`, `snapshot create`, `snapshot restore`, `seed apply`
  - service control: `status services`, `services list`, `services enable`, `services disable`
  - configuration: `config show`, `config validate`
  - environment/discovery: `endpoints`, `env terraform`, `env pulumi`
  - compatibility helpers: launching or locating `tinyterraform` and `tinyaz`
  - diagnostics: health, port usage, data root, cert/trust status, routing status, runtime backend/container identity, and per-service readiness where relevant
- The CLI should present a cohesive user-facing product even when the underlying emulator implementation is provider-specific.

## 12. Docker

### Build requirements
- Multi-stage Docker build.
- Small final runtime image.
- Expose management and data-plane ports.
- Support volume mounts for persistence.

### Runtime example
- `docker run -p 4566:4566 -p 4567:4567 -p 4577:4577 -p 4578:4578 -p 4579:4579 -p 4580:4580 -p 4581:4581 -p 4582:4582 -p 4583:4583 -p 4584:4584/udp -v tinycloud-data:/var/lib/tinycloud tinycloud:latest`
- `docker run -p 4566:4566 -p 4567:4567 -p 4577:4577 -p 4578:4578 -p 4579:4579 -p 4580:4580 -p 4581:4581 -p 4582:4582 -p 4583:4583 -p 4584:4584/udp -p 4585:4585 -v tinycloud-data:/var/lib/tinycloud tinycloud:latest`

## 13. Compatibility

### Terraform
- Support `azurerm_resource_group` against the ARM endpoint.
- Document `ARM_ENDPOINT`, `ARM_METADATA_HOST`, and any custom provider host overrides.
- Provide a working example that creates one resource group.

### Pulumi
- Support environment-based ARM endpoint configuration.
- Ensure provider registration and resource group CRUD work without special code changes.

### Azure SDKs
- Return ARM responses using expected Azure shapes.
- Advertise correct data-plane endpoints for Blob, Key Vault, and other implemented services.
- Keep JSON casing and headers Azure-compatible where possible.

## 14. API contracts

### Management responses
- Use Azure-style `id`, `name`, `type`, `location`, `properties`, and `tags`.
- Use `properties.provisioningState` for resource groups and async operations.
- Return error payloads in Azure-compatible `CloudError` format.

### Async operations
- Create an operation record for every async control-plane change.
- Return `Azure-AsyncOperation` and/or `Location` headers.
- Support `InProgress`, `Succeeded`, and `Failed`.
- Expose a polling endpoint that returns status and terminal error details.

## 15. Implementation order

1. Foundation: config, logging, HTTP server, state layer, Docker.
2. Metadata and token issuer.
3. Resource ID and api-version parsing.
4. Resource groups CRUD.
5. Async operations.
6. Blob storage service.
7. Endpoint discovery and compatibility responses.
8. CLI commands.
9. Terraform/Pulumi examples.
10. Snapshot and seed admin flows.

## 16. Acceptance criteria

- The project compiles cleanly.
- `docker run` starts the emulator successfully.
- Resource group create/update/get/list/delete work end to end.
- SQLite persistence survives restart.
- At least one data-plane service works end to end.
- Async operations poll to completion.
- Responses are Azure-compatible enough for basic SDK and Terraform workflows.
- No core flow contains placeholder-only stubs.

## 17. Local runtime expectations

- Local developer workflows must run without `sudo` or administrator privileges.
- Core TinyCloud runtime workflows must run without `sudo` or administrator privileges.
- Compatibility wrappers may temporarily require elevated privileges when local host routing or certificate/bootstrap behavior cannot yet be achieved in an unprivileged way. Removing that requirement should remain a compatibility goal.
- Snapshot creation without an explicit path must write to a location that is writable by the active process.
- Container defaults may continue to use `/var/lib/tinycloud` because the image provisions that directory for the non-root runtime user.

## 18. Post-v1 next steps

The current codebase satisfies the v1 must-ship scope. The next steps should now optimize for complete local developer workflows rather than broad, shallow feature expansion.

### Command architecture comes first

Before adding more first-class wrapper/product surface, the command layer should be moved into a cloud-agnostic top-level location so the repository can grow beyond Azure cleanly.

This should happen in this order:

1. promote the effective repo/module/build root from `tinycloud\azure` to `tinycloud`, or introduce an equivalent top-level Go workspace structure
2. migrate the main product CLI entrypoints from the Azure tree to `tinycloud\cmd`
3. introduce shared cloud-agnostic CLI/runtime support outside provider-specific trees
4. complete the first-class `tinycloud` CLI as the LocalStack-style main product command
5. keep `tinyterraform` working through the migration
6. implement standalone `tinyaz`
7. then continue with wrapper contract tightening and broader workflow verification

### CLI migration plan

- Step 1: establish the top-level `tinycloud` directory as the buildable Go root via repo-root `go.mod`, repo-root `go.work`, or another equivalent structure
- Step 2: migrate Docker build context and output paths so builds no longer assume `tinycloud\azure` is the repository root
- Step 3: establish `tinycloud\cmd\tinycloud` as the long-term home of the main product CLI
- Step 4: move `tinycloud\cmd\tinyterraform` alongside it so wrappers also live at the top level
- Step 5: adapt the current Azure emulator to plug into the top-level CLI instead of owning it
- Step 6: extract shared CLI/runtime/bootstrap helpers out of the Azure tree
- Step 7: migrate wrapper scripts, docs, examples, and command references to the new top-level paths
- Step 8: leave Azure implementation code under `tinycloud\azure\...`
- Step 9: add future provider implementations like `tinycloud\aws\...` without restructuring the top-level command layer again

### Pipeline impact of the CLI move

The following project surfaces must move together with the command-layer migration:

- `go.mod` / Go workspace configuration
- `Dockerfile`
- wrapper build/run paths such as `scripts\tinyterraform.ps1`
- docs and example command references
- local development commands like `go run .\cmd\tinycloud ...`
- any future CI build/test/release jobs

The migration is only considered complete when those paths are internally consistent again.

### Roadmap philosophy

The roadmap should follow the same practical pattern used by successful local cloud emulators:

- start with a narrow set of services that unlock real application stacks
- prefer one complete workflow over multiple placeholder implementations
- add service families in tiers so the emulator stays coherent
- verify developer workflows through Compose, SDK, and IaC examples

### Service family roadmap

#### Tier 0: command architecture and product shell

1. Promote the effective repo/module/build root from `tinycloud\azure` to `tinycloud`
2. Move the main CLI and wrapper entrypoints to `tinycloud\cmd`
3. Complete the cohesive `tinycloud` CLI product surface
4. Keep `tinyterraform` parity during and after the migration
5. Implement standalone `tinyaz`

#### Tier 1: complete the core application workflow set

1. Key Vault secrets data-plane
2. One queueing workflow end to end:
   - Service Bus queues if Azure-native messaging is the main target
   - Queue Storage if a simpler storage-queue workflow is the higher-value first step
3. One second storage-style service:
   - Queue Storage
   - or Table Storage
4. Compose-first local workflow
5. Verified Terraform integration

#### Tier 2: cover common local-cloud building blocks

6. Table Storage
7. Service Bus queues
8. Service Bus topics and subscriptions
9. App Configuration key-values
10. Cosmos DB core API subset

#### Tier 3: broader event/data workflow support

11. Event Hubs producer/consumer subset
12. Queue Storage poison/dead-letter style local behavior where relevant
13. Blob event notification hooks only if required by real workflows
14. Key Vault certificates only if actual app stacks need them

#### Tier 4: app platform and networking foundations

15. DNS/private name resolution subset
16. Virtual Networks and subnets
17. NSGs and basic network rule modeling
18. Private Endpoints for supported services
19. Azure Functions local trigger/runtime helpers
20. Function App ARM resource and deployment helpers
21. App Service / Web App ARM resource shell
22. Container Registry subset

#### Tier 5: optional higher-complexity expansions

23. Managed identity scenario presets for app-to-service testing
24. Additional deployment-template coverage for the already implemented providers
25. Further Blob compatibility refinement as specific SDK gaps appear
26. Container Apps or App Service workflow depth
27. Load Balancer / public IP modeling only if concrete workflows require them

### Why these are next

- LocalStack shows the value of Docker Compose-first workflows, ready-state initialization, and repeatable state handling for developer adoption.
- Azurite shows the value of deeper behavior within a narrow service boundary rather than many shallow placeholders.
- MiniStack shows the value of solving real end-to-end application workflows instead of only exposing control-plane resource shells.
- The next meaningful expansion for TinyCloud should include additional real services so the project covers more than ARM plus Blob.
- Azure’s own emulator ecosystem shows realistic family boundaries:
  - storage family: Blob, Queue, Table
  - secrets/config family: Key Vault, App Configuration
  - messaging/event family: Service Bus, Event Hubs
  - data family: Cosmos DB subset

### Scope of each next step

#### Key Vault secrets data-plane

- Add secret set/get/list/delete behavior on the Key Vault service port.
- Persist secrets in SQLite and snapshots.
- Advertise secret endpoints consistently through metadata and ARM resource responses.
- Keep the implementation narrow and honest; do not attempt full Key Vault parity.

#### One queueing workflow

- Add either Service Bus queues or Queue Storage as the next real data-plane service.
- Include both provisioning and message send/receive/delete behavior needed for local worker-style development.
- Prefer one complete queueing path over partial implementations of both.

#### Additional service coverage

- Add one more implemented service beyond Blob and queueing so the emulator is not overly concentrated in a single workflow.
- Preferred order:
  - Queue Storage if simple storage queues are the most common target
  - Table Storage if document/key-value style storage is the next common workflow
  - Service Bus if Azure-native enterprise messaging is the higher priority
- Keep each service narrow but real:
  - CRUD plus the minimum data-plane behaviors required by actual application workflows
  - persist state in SQLite and snapshots
  - advertise endpoints consistently through metadata and resource responses

#### Broader common-service roadmap

After secrets plus one queueing path are in place, expand service coverage in the order most likely to unlock real local application stacks:

1. Queue Storage
2. Table Storage
3. Service Bus queues
4. Service Bus topics/subscriptions
5. Cosmos DB core API subset
6. Event Hubs producer/consumer subset
7. App Configuration key-value store
8. Key Vault certificates only if required by actual workflows
9. Optional Azure Functions integration helpers for local event-driven apps
10. DNS/private resolution
11. Virtual networking subset
12. Function App / App Service resource shells
13. Container Registry subset

The intent is not to implement every Azure service. The intent is to cover the common local-cloud building blocks that real applications use:

- object storage
- secrets
- queues
- pub/sub
- key-value or document storage
- application configuration
- event streaming
- basic networking
- application hosting and event compute
- image/artifact distribution

#### Service maturity model

Each new service should move through the same maturity path:

1. ARM resource shell if the service normally has one
2. Real data-plane behavior for the minimum useful workflow
3. SQLite persistence and snapshot support
4. Metadata/endpoint advertisement
5. Docker Compose example coverage
6. IaC or SDK verification where realistic

#### Compose-first local workflow

- Add a documented Docker Compose example with TinyCloud plus at least one app/worker pair.
- Add a ready-state bootstrap or seed pattern so resources can be created automatically at container startup.
- Make the intended local-dev startup sequence explicit and reproducible.

#### Verified Terraform integration

- Add an automated verification path for the existing Terraform example once Terraform is available in CI.
- Keep the docs honest until this is actually verified.
- Treat working `azurerm_resource_group` apply/destroy as the acceptance target.

#### Compose-first workflow maturity

To align with the way projects like LocalStack are actually used, the roadmap should eventually include:

- a first-party Docker Compose stack
- startup initialization hooks or seed bundles
- service readiness checks
- optional seeded demo environments for common app patterns

### Realistic service target list

The realistic next-service target list for TinyCloud is:

- Blob Storage
- Queue Storage
- Table Storage
- Key Vault secrets
- Service Bus queues
- Service Bus topics/subscriptions
- Cosmos DB core API subset
- Event Hubs subset
- App Configuration key-values
- DNS / private DNS subset
- Virtual Networks and subnets
- NSGs and private endpoints for supported services
- Azure Functions helpers plus Function App resource shell
- App Service / Web App resource shell
- Container Registry subset

The following are realistic only after the above are in place:

- Key Vault certificates
- Azure Functions integration helpers
- blob-trigger or queue-trigger local workflow helpers
- deeper Function App runtime behavior
- Container Apps workflow support
- load balancer style network front-door modeling

The following should remain out of scope until there is clear evidence of demand:

- full Azure parity for any service
- broad networking products
- RBAC/policy completeness
- managed Kubernetes
- full data-plane parity for databases and messaging systems

#### Deployment subset expansion

- Only expand deployment-template support if a real workflow needs it.
- Prefer static resource coverage for already-implemented providers before adding expressions, functions, or general ARM semantics.

#### Blob compatibility refinement

- Continue tightening headers, XML, and auth behavior only in response to concrete SDK or tooling failures.
- Avoid speculative parity work with no validating client.
