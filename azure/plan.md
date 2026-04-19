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

### Repo layout status
- The effective Go workspace/build root is now the top-level `tinycloud` directory.
- Repo-root Docker builds and top-level `cmd\...` entrypoints now exist.
- Shared CLI/runtime code now lives outside the provider-specific Azure tree.
- Azure implementation code still lives under `tinycloud\azure\...`.
- Remaining wrapper/product work is now about standalone `tinyaz`, final wrapper-contract locking, and later portability cleanup rather than repo-root migration itself.

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
- `setup`
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
- `setup` should validate and prepare the local TinyCloud environment before normal runtime use.
- `setup --full` should become the first-run install command that validates or installs the full local suite, including the runtime image, config/data roots, wrapper binaries, and supported toolchain prerequisites.
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
- `services list` should remain the inventory/config view, distinct from the runtime-oriented `status services` view.
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
- PowerShell must not remain a hard product dependency for normal `tinycloud`, `tinyterraform`, or `tinyaz` usage; compiled CLI binaries should be the primary user-facing surface across Windows, macOS, and Linux.
- Any remaining PowerShell scripts should be treated as transitional compatibility shims rather than the long-term product command surface.
- The target compatibility model for both `tinyterraform` and `tinyaz` is Model 2 where TinyCloud officially supports the command/resource family: the wrapper should classify the command family, resolve the correct TinyCloud endpoint or service endpoint, and preserve the normal upstream command shape instead of requiring users to fall back to manual endpoint helpers for supported flows.
- The wrapper parity target should track the current TinyCloud emulation scope rather than only the runtime listener list. Today that means the 18 emulator areas documented in the README current-emulation-scope table.
- `tinyaz` should target full wrapper coverage across all 18 current implemented TinyCloud emulation-scope areas, with the wrapper responsible for whatever TinyCloud compatibility behavior is needed to preserve a coherent Azure CLI-shaped workflow for each area.
- `tinyterraform` should target full wrapper coverage only for the parts of the current implemented TinyCloud emulation scope that have credible real Terraform provider/resource coverage and that TinyCloud can satisfy accurately.
- In practice, the strongest future `tinyterraform` targets are ARM and resource-oriented families first: resource groups, storage accounts, Blob containers, storage queues, storage tables and table entities, Key Vault resources and secrets, virtual networks, subnets, network security groups and rules, private DNS zones and A records, Service Bus namespaces/queues/topics/subscriptions, Event Hubs namespaces/hubs/consumer groups, and selective App Configuration, Cosmos DB, and limited deployment-template-backed resources once the real provider contract is verified against TinyCloud.
- Live operational objects such as queue messages, Service Bus messages, event payload publishing/consumption, and Cosmos document CRUD are not the primary `tinyterraform` target even when the underlying emulator service exists, because they are weaker Terraform fits than resource-oriented control-plane or nested resource contracts.
- Final per-tool command-family guarantees should still be documented and locked only after the standalone wrapper surfaces exist in real code and can be verified against actual behavior.

### CLI product structure
- The LocalStack analogue should be:
  - `tinycloud` as the main product/runtime CLI, analogous to `localstack`
  - `tinyterraform` as the Terraform wrapper, analogous to `tflocal`
  - `tinyaz` as the Azure CLI wrapper, analogous to `azlocal`
- `tinycloud` should own lifecycle, status, diagnostics, endpoint discovery, configuration, state helpers, and wrapper discovery/orchestration.
- `tinycloud` should eventually own the official install/bootstrap workflow through `setup` and `setup --full`, with the bootstrap scripts only installing the `tinycloud` CLI itself.
- `tinycloud` should discover and manage the active TinyCloud runtime so `status`, `logs`, `stop`, and `wait` work against the same instance with minimal manual wiring.
- `tinycloud start` should be able to launch the runtime through the supported backend for the environment rather than assuming only a direct foreground Go server process.
- The default local-developer product flow should be container-oriented, with the CLI responsible for initializing and managing the TinyCloud container/runtime in the same way the LocalStack CLI manages its local runtime.
- The CLI should keep enough runtime metadata to reconnect later for `status`, `logs`, `wait`, `stop`, and `restart` without requiring the user to manually pass process IDs, ports, or container names.
- `tinyterraform` and `tinyaz` should remain compatibility wrappers around real upstream tools, not bespoke reimplementations of Terraform or Azure CLI, but they may still need command-family classification and endpoint routing for the officially supported subset.

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
- `start` should default to detached mode and should print the runtime identifier, backend, selected services, management URL when available, and the next useful commands such as `tinycloud status`, `tinycloud logs -f`, and `tinycloud stop`.
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

## 13. Distribution and installation

### Artifact model
- Publish native CLI binaries for `tinycloud`, `tinyterraform`, and later `tinyaz`.
- Publish checksums and release notes with every release.
- Publish the runtime image separately from the CLI binaries.
- Prefer GitHub Releases for versioned binary artifacts.
- Prefer GHCR for the runtime image.

### Bootstrap model
- Serve small bootstrap scripts from a TinyCloud-controlled domain such as:
  - `https://get.tinycloud.dev/install.sh`
  - `https://get.tinycloud.dev/install.ps1`
- The bootstrap scripts should install the `tinycloud` CLI only.
- The bootstrap scripts should not contain the full suite install logic.
- The bootstrap scripts should direct users to run `tinycloud setup --full`.

### Setup command model
- `tinycloud setup` should validate and prepare the local TinyCloud environment.
- `tinycloud setup --full` should own the first-run install and validation flow for the full local suite.
- `tinycloud setup --full` should verify Docker, initialize config/data roots, prepare runtime metadata, validate wrapper prerequisites, and later manage supported upstream tool versions where appropriate.

### Toolchain model
- `tinyterraform` should continue to use a real Terraform binary rather than a reimplementation.
- `tinyaz` should continue to use a real Azure CLI `az` binary rather than a reimplementation.
- Near-term, external dependency detection is acceptable.
- Longer-term, TinyCloud may manage tested upstream tool versions in a TinyCloud-owned tools directory rather than relying only on the global system `PATH`.

### Public install story
- The intended end-user install story should eventually be:
  1. bootstrap `tinycloud`
  2. run `tinycloud setup --full`
  3. run `tinycloud start`
- The manual source-build path may remain available, but it should not remain the primary onboarding story.

## 14. Compatibility

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

## 15. API contracts

### Management responses
- Use Azure-style `id`, `name`, `type`, `location`, `properties`, and `tags`.
- Use `properties.provisioningState` for resource groups and async operations.
- Return error payloads in Azure-compatible `CloudError` format.

### Async operations
- Create an operation record for every async control-plane change.
- Return `Azure-AsyncOperation` and/or `Location` headers.
- Support `InProgress`, `Succeeded`, and `Failed`.
- Expose a polling endpoint that returns status and terminal error details.

## 16. Implementation order

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

## 17. Acceptance criteria

- The project compiles cleanly.
- `docker run` starts the emulator successfully.
- Resource group create/update/get/list/delete work end to end.
- SQLite persistence survives restart.
- At least one data-plane service works end to end.
- Async operations poll to completion.
- Responses are Azure-compatible enough for basic SDK and Terraform workflows.
- No core flow contains placeholder-only stubs.

## 18. Local runtime expectations

- Local developer workflows must run without `sudo` or administrator privileges.
- Core TinyCloud runtime workflows must run without `sudo` or administrator privileges.
- Compatibility wrappers may temporarily require elevated privileges when local host routing or certificate/bootstrap behavior cannot yet be achieved in an unprivileged way. Removing that requirement should remain a compatibility goal.
- Normal installed CLI usage should not require PowerShell. If PowerShell wrappers still exist, they should remain optional compatibility paths until the underlying behavior is fully migrated into the Go command layer.
- Snapshot creation without an explicit path must write to a location that is writable by the active process.
- Container defaults may continue to use `/var/lib/tinycloud` because the image provisions that directory for the non-root runtime user.

## 19. Post-v1 next steps

The current codebase satisfies the v1 must-ship scope. The next steps should now optimize for complete local developer workflows rather than broad, shallow feature expansion.

### Command architecture status

Roadmap items `#1` through `#5` are now complete:

1. the effective repo/module/build root now lives at `tinycloud`
2. main product CLI and wrapper entrypoints now live at top-level `cmd\...` locations
3. shared cloud-agnostic CLI/runtime support now exists outside provider-specific trees
4. the cohesive LocalStack-style `tinycloud` main CLI now exists
5. `tinyterraform` now has one shared ownership model for the supported runtime-routing compatibility path

The next active wrapper/product steps are:

6. implement standalone `tinyaz` across all 18 current TinyCloud emulation-scope areas
7. then define and verify the final per-tool wrapper contract for the current TinyCloud emulation scope, including the narrower Terraform-feasible portion of that scope
8. then expand `tinyterraform` across the Terraform-feasible supported portion of the current TinyCloud emulation scope

### Remaining ordered roadmap

After `#6`, `#7`, and `#8`, the next remaining work should stay in this order:

9. verified Terraform integration once Terraform is available in CI
10. PowerShell-free wrapper/runtime orchestration for normal cross-platform CLI usage
11. Phase 1 distribution foundation: GitHub Releases, GHCR runtime image, bootstrap scripts, and `tinycloud setup` / `tinycloud setup --full`
12. Queue Storage poison/dead-letter behavior where it materially improves real worker workflows
13. Blob event notification hooks only if a real workflow needs them
14. Key Vault certificates only if a real workflow needs them
15. Private Endpoints for supported services
16. Azure Functions local trigger/runtime helpers
17. Function App ARM resource and deployment helpers
18. App Service / Web App resource shell
19. Container Registry subset
20. Compose-first local workflow
21. Managed identity scenario presets for app-to-service testing
22. Additional deployment-template coverage for the already implemented providers, but only when a real workflow needs it
23. Further Blob compatibility refinement, but only for concrete SDK/tooling gaps
24. Container Apps or deeper App Service workflow support only if real workflows require it
25. Load Balancer / public IP modeling only if real workflows require it

### Post-MVP deferred distribution work

The MVP should stop at the Phase 1 install and release model above. Later packaging/distribution polish is intentionally deferred until after MVP:

- package managers, managed tool cache, CLI-driven dependency installation/update flows, and environment diagnostics
- native installers, signing, update checks, and product-grade installer UX

### Roadmap philosophy

The roadmap should follow the same practical pattern used by successful local cloud emulators:

- start with a narrow set of services that unlock real application stacks
- prefer one complete workflow over multiple placeholder implementations
- add service families in tiers so the emulator stays coherent
- verify developer workflows through Compose, SDK, and IaC examples

### Service family roadmap

The current 18-area emulation scope is already implemented or intentionally partial. Post-v1 service work should therefore focus on deeper workflow coverage rather than introducing service families that already exist in a basic form.

#### Tier 0: wrapper completion and contract locking

1. Standalone `tinyaz` across all 18 current TinyCloud emulation-scope areas
2. Final per-tool wrapper contract locking for `tinyterraform` and `tinyaz`
3. Expand `tinyterraform` across the Terraform-feasible supported portion of the current TinyCloud emulation scope

#### Tier 1: verification and workflow polish

4. Verified Terraform integration in CI
5. Compose-first local workflow
6. Managed identity scenario presets for app-to-service testing

#### Tier 2: portability and distribution

7. PowerShell-free wrapper/runtime orchestration for normal cross-platform CLI usage
8. Phase 1 distribution foundation: GitHub Releases, GHCR runtime image, bootstrap scripts, and `tinycloud setup` / `tinycloud setup --full`

#### Tier 3: behavior refinements for implemented services

9. Queue Storage poison/dead-letter behavior where it materially improves real worker workflows
10. Blob event notification hooks only if a real workflow needs them
11. Key Vault certificates only if a real workflow needs them
12. Additional deployment-template coverage for already implemented providers, but only when a real workflow needs it
13. Further Blob compatibility refinement, but only for concrete SDK/tooling gaps

#### Tier 4: broader application-platform and networking additions

14. Private Endpoints for supported services
15. Azure Functions local trigger/runtime helpers
16. Function App ARM resource and deployment helpers
17. App Service / Web App resource shell
18. Container Registry subset

#### Tier 5: optional higher-complexity expansions

19. Container Apps or deeper App Service workflow support only if real workflows require it
20. Load Balancer / public IP modeling only if concrete workflows require it

### Why these are next

- LocalStack shows the value of Docker Compose-first workflows, ready-state initialization, and repeatable state handling for developer adoption.
- Azurite shows the value of deeper behavior within a narrow service boundary rather than many shallow placeholders.
- MiniStack shows the value of solving real end-to-end application workflows instead of only exposing control-plane resource shells.
- The next meaningful expansion for TinyCloud is now wrapper completion, verification depth, and workflow polish on top of the already-implemented 18-area surface.
- Azure’s own emulator ecosystem shows realistic family boundaries:
  - storage family: Blob, Queue, Table
  - secrets/config family: Key Vault, App Configuration
  - messaging/event family: Service Bus, Event Hubs
  - data family: Cosmos DB subset

### Scope of each next step

#### Standalone `tinyaz`

- Add `cmd/tinyaz` as the first-class Azure CLI compatibility entrypoint.
- Keep it as a wrapper around the real `az` binary rather than a reimplementation.
- Target full wrapper coverage across the current 18-area TinyCloud emulation scope.

#### Final wrapper contract

- Lock the exact supported command-family contract for `tinyterraform` and `tinyaz`.
- Keep `tinyterraform` explicitly limited to the Terraform-feasible portion of the current emulation scope.
- Add tests and docs that reflect the supported contract rather than implied behavior.

#### `tinyterraform` expansion

- Expand `tinyterraform` in a resource-oriented order: ARM resources first, then storage child resources, then Key Vault and messaging nested resources, and only then selective App Configuration, Cosmos DB, Blob-object, or limited deployment-template-backed resources where the real provider contract is validated against TinyCloud.
- Do not treat live queue messages, Service Bus messages, event payloads, or Cosmos document CRUD as the primary `tinyterraform` contract.
- Keep the broadened `tinyterraform` scope explicit and documented rather than implied by the underlying emulator surface.

#### Verified Terraform integration

- Add automated verification for the existing Terraform example once Terraform is available in CI.
- Keep the docs honest until that verification exists.
- Treat working `azurerm_resource_group` apply/destroy as the baseline acceptance target.

#### Distribution phases

- The MVP should add GitHub Releases, the GHCR runtime image, bootstrap scripts, and the first `tinycloud setup` / `tinycloud setup --full` install flow.
- Package managers, a managed tools directory for tested upstream tool versions where appropriate, CLI-driven dependency management, environment diagnostics, native installers, signing, update checks, and a polished product install/update UX are post-MVP work.

#### Compose-first local workflow

- Add a documented Docker Compose example with TinyCloud plus at least one app/worker pair.
- Add a ready-state bootstrap or seed pattern so resources can be created automatically at container startup.
- Make the intended local-dev startup sequence explicit and reproducible.

#### Behavior refinements for implemented services

- Add Queue Storage poison/dead-letter behavior only where it materially improves worker-style testing.
- Add Blob event notification hooks only when a real workflow needs them.
- Add Key Vault certificates only when a real workflow needs them.
- Expand deployment-template coverage only for already implemented providers and only when a real workflow needs it.
- Refine Blob compatibility only for concrete SDK/tooling gaps rather than speculative parity work.

#### Future platform additions

- Private Endpoints for supported services
- Azure Functions local trigger/runtime helpers
- Function App ARM resource and deployment helpers
- App Service / Web App resource shell
- Container Registry subset
- PowerShell-free wrapper/runtime orchestration so normal CLI usage is portable across Windows, macOS, and Linux without requiring PowerShell
- optional deeper Container Apps, App Service, and load-balancer-style modeling only when real workflows require them
