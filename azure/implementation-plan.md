# TinyCloud Local Implementation Plan

This file is intentionally local-only context. It is ignored by Git and should not be committed.

## Tracking Rules

- Update this file after completing a meaningful implementation slice.
- Record what shipped, what remains, and what should be done next.
- Keep the next steps concrete enough to resume work in a later session.

## Reference Scope

Primary product scope and acceptance criteria live in `plan.md`.

## Current Status

The project now appears to satisfy the `plan.md` v1 must-ship scope. Remaining work is now primarily in the post-v1 roadmap, plus optional v1 verification work such as automated Terraform CI coverage.

Roadmap note:

- `#1` Promote the effective repo/module/build root from `tinycloud\azure` to `tinycloud` is now complete via the repo-root Go workspace, repo-root Docker build, repo-root wrapper entrypoints, repo-root cache/runtime defaults, and validated repo-root `go run .\azure\cmd\...` command paths.
- `#2` Move the main CLI and wrapper entrypoints to cloud-agnostic `tinycloud\cmd` locations is now complete via top-level `tinycloud\cmd\tinycloud`, `tinycloud\cmd\tinycloudd`, and `tinycloud\cmd\tinyterraform` entrypoints, with Azure-backed compatibility shims still left in `tinycloud\azure\cmd\...` for compatibility.
- `#3` Introduce shared cloud-agnostic CLI/runtime support outside provider-specific trees is now complete via repo-root `tinycloud\cli\...` command entry packages, Azure-backed runtime adapters under `tinycloud\azure\runtime\...`, and validated top-level plus Azure-compatibility command paths that both route through the shared repo-root command layer.
- `#4` Implement the cohesive LocalStack-style `tinycloud` main CLI is now complete via the shared repo-root lifecycle/status/config/service command layer, the managed process backend, the default Docker-backed local workflow, backend/bootstrap flags, and validated repo-root runtime management flows.
- Roadmap item `#5` is now complete: `tinyterraform` has one shared ownership model for the supported runtime-routing compatibility path.
- The next active roadmap item is now `#6`: implement standalone `tinyaz` as the Azure CLI compatibility analogue to `azlocal`, explicitly targeting full wrapper coverage across all 18 current TinyCloud emulation-scope areas.

### Implemented

- Foundation/runtime
  - Config and port wiring
  - CLI entrypoints
  - Built-in `cmd/tinycloud` command with `start`, `init`, `reset`, `status`, `endpoints`, `env terraform`, `env pulumi`, `snapshot create`, `snapshot restore`, and `seed apply`
  - Local developer tooling bootstrap script for installing core dependencies
  - Docker image and non-root runtime
  - Azure Dockerfile now builds from either `tinycloud\azure` or the repo root via `TINYCLOUD_CONTEXT_ROOT`
  - Repo root now has a first-class Dockerfile for `docker build -t tinycloud-azure .`
  - Repo root now has a first-class `scripts\tinyterraform.ps1` wrapper path for Terraform compatibility through repo-root build defaults
  - Repo root now has a first-class `scripts\tinycloud.ps1` wrapper path for control CLI commands through the repo-root workspace
  - Repo root now has a first-class `scripts\tinycloudd.ps1` runtime wrapper path through the repo-root workspace
  - Repo-root `tinycloud` and `tinycloudd` wrappers now build cached binaries through the repo-root Go workspace instead of shelling through `go run .\azure\cmd\...`
  - Repo-root `tinyterraform` now also defaults its runtime artifacts to the top-level `.tinyterraform-runtime`
  - Repo-root `tinyterraform` no longer delegates through `azure\scripts\tinyterraform.ps1`; it now owns the compatibility flow directly while auto-detecting the current Azure-backed source tree from the repo root
  - The shared Go `tinyterraform` command layer now owns the local `terraform init` reset/bootstrap path directly, so both top-level and Azure-compatibility launchers can run `init` without locating `scripts\tinyterraform.ps1`, and direct `scripts\tinyterraform.ps1 init` now delegates into that same launcher-owned path instead of duplicating init/bootstrap logic inline
  - The shared Go `tinyterraform` command layer now also prebuilds and injects the current `tinycloud` helper for runtime-routed launcher flows before invoking the wrapper, so the first-class launcher owns helper-binary preparation for supported runtime commands beyond `init` while the wrapper still handles the remaining privileged hosts/cert/CLI-shim path
  - The shared Go `tinyterraform` command layer now also resolves and injects the real `terraform` binary for runtime-routed launcher flows before invoking the wrapper, so the first-class launcher owns upstream Terraform binary selection for supported runtime commands while the wrapper reuses that injected path
  - The shared Go `tinyterraform` command layer now also creates and cleans up the temporary Terraform provider override in the actual Terraform working directory for runtime-routed launcher flows, including `-chdir` launcher invocations, while the wrapper reuses that injected override path instead of owning the file lifecycle
  - The shared Go `tinyterraform` command layer now also starts, waits for, and tears down the isolated process-backed TinyCloud runtime for runtime-routed launcher flows, and it injects the resulting Terraform environment values back into the wrapper so the wrapper can focus on the remaining privileged hosts/cert/Azure-CLI-shim path
  - The shared Go `tinyterraform` command layer now also owns the temporary `management.azure.com` hosts-file mapping for launcher-driven runtime flows, and both the launcher and wrapper now honor `TINYTERRAFORM_HOSTS_PATH` for isolated validation or compatibility-path variation handling
  - The Azure-compatibility `tinyterraform` launcher-to-wrapper probe test now retries the local `/_admin/healthz` check for a short bounded window instead of treating one 2-second request as authoritative, which removes the residual validation flake without changing product behavior
  - TinyTerraform helper binaries under `.tinyterraform-runtime` now use collision-resistant unique executable names instead of a single fixed `tinycloud.exe` / `tinyterraform.exe` path, so repeated wrapper and launcher `init` runs can reuse the same runtime root without tripping Windows file-lock conflicts on helper rebuilds
  - Repo-root wrappers now resolve command build targets through repo-root-relative package paths, preferring the top-level `cmd\...` locations and falling back to the Azure compatibility paths under `azure\cmd\...` instead of hardcoded `tinycloud/cmd/...` package strings
  - Repo-root wrappers now also default their Go build cache to `tinycloud\.gocache`, and the repo root ignores that cache directory alongside the repo-root runtime artifact directories
  - Real repo-root `go run .\azure\cmd\tinycloud ...` and `go run .\azure\cmd\tinyterraform -- ...` entrypoints are now covered by automated tests so the current workspace-root command paths stay locked in during the remaining migration
  - The repo root now also has a first-class Go module shim plus top-level `cmd\tinycloud` and `cmd\tinycloudd` entrypoints, with the shared product-facing command entry layer now living at repo-root `tinycloud\cli\...` while the Azure-backed runtime adapters live under `tinycloud\azure\runtime\...`
  - The repo root now also has a first-class top-level `cmd\tinyterraform` launcher entrypoint, and all three top-level command entrypoints now share the repo-root `tinycloud\cli\...` command layer instead of treating Azure-owned command packages as the product command home
  - Admin endpoints for health, metrics, reset, snapshot, and seed
  - Admin runtime endpoints now also expose `/_admin/runtime` and `/_admin/services` for machine-readable runtime/service inspection
  - Metadata discovery endpoint with management, auth, and service URLs
  - Local HTTPS management listener with generated workspace-local certificate material
  - `env terraform` and `env pulumi` output with local endpoint settings
  - Public runtime config now lives at `azure\runtime\tinycloudconfig`, with the old internal config package reduced to a thin compatibility alias
  - Runtime service selection now exists through `TINYCLOUD_SERVICES`, including service-family aliases plus config validation for invalid entries and conflicting listener addresses
  - The runtime now starts only the enabled service listeners instead of always binding every port, and metadata plus CLI endpoint discovery now advertise only the enabled service set
  - The shared repo-root `tinycloud` CLI now owns a managed local process backend with `start`, `stop`, `restart`, `logs`, `wait`, `status runtime`, `status services`, `config show`, `config validate`, `services list`, `services enable`, `services disable`, and persisted runtime/service metadata under `.tinycloud-runtime`
  - `tinycloud start` now owns the polished human-readable terminal UX: interactive startup-only branding, lifecycle steps, runtime summaries, and next-step guidance, while `status runtime`, `status services`, `config show`, and `endpoints` now render terminal-friendly tables/sections with status icons and unchanged `--json` automation output
  - `tinycloud start` now defaults to detached startup so it returns control to the shell by default, while `tinycloud start --attached` is the explicit foreground/log-streaming mode
  - `services enable` / `services disable` now persist configuration and print explicit restart guidance against a running runtime
  - The shared repo-root `tinycloud` CLI now also owns the default Docker-backed local workflow, auto-builds the repo-root `tinycloud-azure` image when needed, reconnects to the active TinyCloud container for `status`/`logs`/`wait`/`stop`/`restart`, and still exposes `--backend process` as the managed non-container fallback
  - `tinycloud start` now supports LocalStack-style bootstrap flags for backend selection plus environment, port-publish, volume-mount, and network injection where relevant to the current Docker workflow

- HTTP platform
  - Shared JSON and CloudError responses
  - Request ID middleware
  - Structured logging middleware
  - Panic recovery middleware
  - CORS middleware
  - Azure header normalization middleware
  - `api-version` enforcement middleware

- Core ARM helpers
  - `internal/core/apiversion`
  - `internal/core/resourceid`

- Persistence
  - SQLite-backed state in `state.db`
  - JSON snapshot/restore still supported
  - Schema migration handling for the evolved resource group table
  - Blob container/blob tables and CRUD helpers
  - Snapshot/restore now preserves Blob containers and objects
  - Storage-account records and CRUD helpers
  - Key Vault records and snapshot persistence
  - Deployment records and snapshot persistence

- Bootstrap state
  - Default tenant
  - Default subscription
  - Default `Microsoft.Resources` provider record

- ARM control plane
  - `GET /tenants`
  - `GET /subscriptions`
  - `GET /subscriptions/{subscriptionId}`
  - `GET /providers`
  - Provider registration lifecycle:
    - list
    - get by namespace
    - register
  - Resource group CRUD:
    - list
    - create/update
    - get
    - delete
  - Resource-group create/update now returns synchronous Azure-style `201` or `200`
  - Storage account CRUD:
    - list
    - create/update
    - get
    - delete
  - Key Vault CRUD:
    - list
    - create/update
    - get
    - delete
  - Deployment list/create/get routes backed by persisted records
  - Deployment execution now supports a minimal static template subset for:
    - `Microsoft.Storage/storageAccounts`
    - `Microsoft.KeyVault/vaults`
  - Unsupported deployment inputs still return a failed deployment record plus failed async operation status
  - ARM storage-account responses now advertise Blob primary endpoints
  - ARM key-vault responses now advertise per-vault URIs
  - Azure-shaped resource group responses
  - Azure-style not-found and validation errors

- Async operations
  - Persistent operation records
  - Polling endpoint:
    - `GET /subscriptions/{subscriptionId}/providers/Microsoft.Resources/operations/{operationId}`
  - `Azure-AsyncOperation`, `Location`, and `Retry-After` headers on resource-group writes/deletes
  - Immediate-completion operation model for now
  - Failed deployment operations now surface terminal error details through the polling endpoint

- Identity and metadata
  - `/oauth/token`
  - `/metadata/identity`
  - `/metadata/identity/oauth2/token`
  - Signed local JWT issuance with configurable issuer, audience, tenant, and subject
  - IMDS-style `Metadata: true` enforcement on identity discovery and token routes
  - Discovery metadata now advertises ARM, auth, and service endpoints
  - Environment document now includes authentication, provider metadata, and additional SDK-facing endpoint fields

- Blob data-plane
  - Dedicated Blob listener on the Blob service port
  - Container create and list operations
  - Blob upload, download, list, and delete
  - SQLite-backed Blob objects and container records
  - Azure-style Blob compatibility headers for versioning, encryption, blob type, content length, and `HEAD`
  - Container XML listings now advertise the full Blob service endpoint URL

- Queue Storage data-plane
  - Dedicated Queue listener on the Queue service port
  - Queue create and list operations
  - Queue message send, receive, and delete behavior with visibility timeouts and pop receipts
  - SQLite-backed queue and message records with snapshot persistence

- Table Storage data-plane
  - Dedicated Table listener on the Table service port
  - Table create, list, and delete operations
  - Entity upsert, list, get, and delete behavior
  - SQLite-backed table and entity records with snapshot persistence

- Service Bus queueing
  - Dedicated Service Bus listener on the Service Bus port
  - Namespace and queue create/list operations
  - Queue message send, receive, and delete behavior with lock tokens and visibility timeouts
  - SQLite-backed namespace, queue, and message records with snapshot persistence
- Service Bus topic/subscription persistence
  - SQLite-backed topic, subscription, and subscription-message records
  - Topic publish fan-out to persisted subscription-bound messages
  - Snapshot/restore support for topics, subscriptions, and topic messages
- Service Bus topics/subscriptions data-plane
  - Topic and subscription create/list routes on the dedicated Service Bus listener
  - Topic publish and subscription receive/delete behavior
  - Publish now succeeds even when a topic has no subscriptions yet
- App Configuration persistence
  - SQLite-backed config-store and key-value records
  - Key, label, value, and content-type persistence
  - Snapshot/restore support for app configuration state
- App Configuration data-plane
  - Dedicated App Configuration listener on its own service port
  - Config store create/list routes
  - Key-value put/get/list/delete routes
  - Metadata and CLI endpoint advertisement for the new service
- Cosmos DB persistence
  - SQLite-backed account, database, container, and document records
  - Document upsert/get/list/delete helpers
  - Snapshot/restore support for Cosmos state
- Cosmos DB data-plane
  - Dedicated Cosmos listener on its own service port
  - Account, database, container, and document CRUD routes
  - Metadata and CLI endpoint advertisement for the new service
- Event Hubs subset
  - SQLite-backed event hub namespace, hub, and event records
  - Dedicated Event Hubs listener on its own service port
  - Namespace and hub create/list routes
  - Event publish and ordered read-from-sequence behavior
  - Metadata and CLI endpoint advertisement for the new service
  - Snapshot/restore support for event hub state
- Virtual Networks and subnets
  - SQLite-backed virtual network and subnet records
  - ARM CRUD for `Microsoft.Network/virtualNetworks`
  - ARM CRUD for nested subnets
  - Address prefix persistence for virtual networks and subnets
  - Snapshot/restore support for virtual network state
  - Azure-style async headers on virtual network and subnet writes/deletes
- Network Security Groups
  - SQLite-backed network security group and security-rule records
  - ARM CRUD for `Microsoft.Network/networkSecurityGroups`
  - ARM CRUD for nested security rules
  - Basic security rule modeling for access, direction, protocol, prefixes, ports, and priority
  - Snapshot/restore support for NSG state
  - Azure-style async headers on NSG and security-rule writes/deletes
- Private DNS persistence
  - SQLite-backed private DNS zone and A-record records
  - Snapshot/restore support for private DNS state
- Private DNS subset
  - `Microsoft.Network/privateDnsZones` ARM CRUD
  - Private DNS A-record ARM CRUD
  - Live UDP resolver for A-record lookups
  - Metadata and CLI endpoint advertisement for the DNS resolver

- Key Vault secrets data-plane
  - Dedicated Key Vault listener on the Key Vault service port
  - Secret set, get, list, and delete behavior
  - SQLite-backed secret records and snapshot persistence
  - ARM-created vault resources are now usable through the data-plane endpoint

- Tests
  - Middleware tests
  - Parser tests
  - State tests
  - ARM route tests
  - CLI env output tests
  - Manual end-to-end verification of the LocalStack-style Terraform wrapper for `init`, `apply`, and `destroy` against `azurerm_resource_group`
  - Current wrapper shape already follows the intended `tflocal`-style model closely enough to validate the compatibility direction

## Remaining Major Gaps

### Optional v1 coverage

- Deployment execution remains intentionally limited:
  - no parameter resolution
  - no template expressions
  - no broader ARM resource-type coverage beyond storage accounts and Key Vault resources

### Post-v1 workflow gaps

- No Compose-first example for a real app plus worker stack
- Terraform example is manually verified but not automatically verified in CI
- No bootstrap installer yet for `tinycloud`
- No `tinycloud setup` or `tinycloud setup --full` command yet
- No public release/distribution flow yet for GitHub Releases plus GHCR-backed bootstrap
- `tinyterraform` now has a first-class launcher entrypoint at `cmd/tinyterraform`, and the shared Go command layer now owns the local `init` reset/bootstrap path for both launcher and direct wrapper entrypoints, but the broader privileged runtime-routing implementation still delegates to the Windows-specific script wrapper
- the shared launcher now also owns `tinycloud` helper-binary preparation, real `terraform` binary resolution, temporary provider-override lifecycle, isolated TinyCloud runtime startup/readiness/env bootstrap, launcher-driven hosts-file routing, local HTTPS certificate trust/bootstrap, and launcher-driven local `az` shim preparation for runtime-routed wrapper calls, and the direct-wrapper path now reuses that same shared `az` shim source instead of carrying a separate inline shim contract
- roadmap item `#5` is complete; the next wrapper roadmap step is now standalone `tinyaz` under `#6`
- the helper-binary collision issue inside `.tinyterraform-runtime` is now resolved, but the broader privileged runtime-routing implementation still lives in the Windows-specific wrapper path
- No standalone `tinyaz` helper yet; the Azure CLI compatibility path currently lives inside `tinyterraform.ps1`
- Compatibility behavior is now documented, and launcher passthrough has been validated for `init` plus non-runtime commands like `help`, `version`, `login`, `logout`, and `console`, including `go run .\cmd\tinyterraform -- version -json`, PowerShell `-chdir=...` invocation through both the Go launcher and `scripts/tinyterraform.ps1`, repo-root `.\scripts\tinyterraform.ps1 -chdir=.\azure\examples\terraform\resource-group init` through the top-level workspace, `TERRAFORM_EXE` override handling through both entrypoints, direct wrapper preservation of machine-readable stdout for commands like `version -json`, direct passthrough of `terraform help` through both entrypoints without requiring the privileged runtime path, direct passthrough of subcommand help requests like `terraform apply -help` through both entrypoints, direct passthrough of `terraform login` and `terraform logout` through both entrypoints, direct passthrough of `terraform console` through both entrypoints, explicit `TINYCLOUD_SOURCE_ROOT`, `TINYCLOUD_GO_WORKDIR`, `TINYCLOUD_MAIN_PACKAGE`, `TINYTERRAFORM_SCRIPT`, `TINYTERRAFORM_SCRIPT_RELATIVE_PATH`, and `TINYTERRAFORM_RUNTIME_ROOT` overrides for compatibility and repo-layout variation handling, automatic upward source-root discovery from nested wrapper paths like `azure\scripts`, and a local-only `init` path that resets/bootstrap TinyCloud without needing cert trust or hosts-file routing and is now shared by both the launcher and direct wrapper entrypoints, but the full Model 2 command-family routing contract is not yet validated across broader officially supported command coverage
- Repo-root `go.work` migration now also validates repo-root `tinyterraform` usage from `tinycloud\`, including direct non-runtime launcher usage plus a real `go run .\azure\cmd\tinyterraform -- -chdir=.\azure\examples\terraform\resource-group init` flow, and test coverage now locks in repo-root wrapper discovery through the Azure source tree
- The current `tinyterraform` path still requires an elevated PowerShell session because of temporary hosts-file routing; removing that privilege requirement remains an open compatibility goal
- PowerShell scripts still exist for some wrapper and developer paths, but the product direction is now explicit: normal `tinycloud`, `tinyterraform`, and future `tinyaz` usage should converge on compiled cross-platform CLI binaries so PowerShell is not a hard dependency
- The approved `tinycloud` terminal UX polish is now complete: interactive startup-only branding, runtime-summary-first detached startup output, human-readable tables for runtime/services/endpoints, grouped config display, and status icons with interactive-only green/red/yellow color handling now ship while preserving JSON output unchanged

### LocalStack-style command compatibility track

The intended user model is:

- `tinycloud` should become the main cohesive product CLI, analogous to `localstack`
- `tinyterraform` should be the TinyCloud analogue to `tflocal`
- `tinyaz` should be the TinyCloud analogue to `azlocal`
- users should be able to keep familiar `terraform` and `az` command habits while the wrappers inject local TinyCloud compatibility
- both wrappers should invoke the real upstream binaries rather than reimplementing command groups
- for officially supported command/resource families, both wrappers should follow the Model 2 shape: classify the command family, resolve the correct TinyCloud management or service endpoint, and preserve the normal upstream command structure
- wrapper parity should now be interpreted against the current TinyCloud emulation scope rather than only the runtime listener list; today that means the 18 emulator areas documented in the README current-emulation-scope table
- `tinyaz` should target full wrapper coverage across all 18 current implemented TinyCloud emulation-scope areas, with the wrapper responsible for whatever TinyCloud compatibility behavior is needed to preserve a coherent Azure CLI-shaped workflow for each area
- `tinyterraform` should target full wrapper coverage only for the parts of the current implemented TinyCloud emulation scope that have credible real Terraform provider/resource coverage and that TinyCloud can satisfy accurately
- the final per-tool command-family contract should still be locked only after the standalone wrapper surfaces exist and can be verified against real behavior
- `tinycloud` itself should own runtime discovery and lifecycle operations like `start`, `stop`, `restart`, `status`, `logs`, and `wait`
- `tinycloud` should expose config inspection/validation so runtime wiring can be understood without reading environment variables by hand
- `tinycloud` should manage the primary local container/runtime workflow rather than assuming users will always bootstrap the runtime manually
- `tinycloud` should expose a first-class service-selection and service-status UX comparable to LocalStack's runtime-management model
- `tinycloud` should provide a polished terminal UX with readable tables/summaries for humans and explicit machine-readable modes for automation

The intended repo structure is:

- `tinycloud\cmd\tinycloud`
- `tinycloud\cmd\tinyterraform`
- `tinycloud\cmd\tinyaz`
- `tinycloud\azure\...`
- future provider trees like `tinycloud\aws\...`

The required migration constraint is:

- `tinycloud\cmd` cannot become the real command home until the top-level `tinycloud` directory is also made into the effective build root for Go and Docker
- this affects `go.mod` or `go.work`, `Dockerfile`, wrapper build paths, and documented `go run` / `go build` commands
- the migration should be treated as one command-architecture slice family, not as an isolated folder rename

That means the wrapper layer should own:

- TinyCloud startup and health checks
- endpoint routing and host mapping
- cert trust/bootstrap for local HTTPS management traffic
- Azure CLI compatibility shims or delegation
- local state alignment and cleanup behavior

And the user-facing compatibility goals should be:

- pass through normal tool arguments with minimal surprises
- pass through stdout, stderr, and exit codes as closely as practical
- avoid requiring manual environment exports for common flows
- keep Terraform configuration as close as possible to normal Azure usage
- keep Azure CLI usage as close as possible to normal `az` usage
- make supported data-plane and control-plane command families feel first-class under `tinyterraform` and `tinyaz`, rather than forcing users onto separate manual helper flows for those supported cases
- document where Azure tooling prevents exact parity

And the main `tinycloud` CLI should additionally own:

- runtime/container initialization in the primary local workflow
- runtime reconnection and runtime identity tracking for later status/log/stop commands
- service-selection inputs such as `TINYCLOUD_SERVICES` and/or `tinycloud start --services ...`
- per-service readiness reporting similar to `status services`
- service enable/disable UX, whether live or restart-mediated
- terminal presentation rules for interactive summaries, tables, logs, and machine-readable output

### Full realistic service roadmap

The current 18-area emulation scope is already implemented or intentionally partial. The remaining realistic roadmap is therefore not “add Blob/Queue/Table/Cosmos/etc.” It is:

1. finish standalone `tinyaz`
2. lock the final per-tool wrapper contract
3. add Terraform CI verification
4. remove PowerShell as a hard dependency for normal CLI usage by moving remaining wrapper/runtime orchestration into the Go command layer
5. deepen selected behaviors where real workflows need them
6. add later platform-oriented expansions such as private endpoints, Functions, App Service, and Container Registry only when they are justified by real workflows

### Common service roadmap after v1

The next broad additions should focus on:

1. wrapper completeness and contract locking
2. Terraform verification
3. PowerShell-free portability work so normal CLI usage is cross-platform and binary-first
4. distribution and install flow, including bootstrap scripts plus `tinycloud setup` / `tinycloud setup --full`
5. worker/event behavior refinements such as Queue poison/dead-letter handling or Blob event hooks when real workflows justify them
6. platform-oriented additions such as private endpoints, Azure Functions helpers, Function App, App Service, and Container Registry

## Suggested Next Sequence

This section is a short execution window into the authoritative `Full Remaining Ordered Sequence` below. It should stay short and should reference the same ordering rather than becoming a second independent roadmap.
It must always be a contiguous prefix of the next unfinished items from `Full Remaining Ordered Sequence`, with no ad hoc entries inserted here that do not also exist there.

Current near-term window:

1. `#6` Implement standalone `tinyaz` as the Azure CLI compatibility analogue to `azlocal`, with full wrapper coverage across all 18 current TinyCloud emulation-scope areas
2. `#7` Define and document the final per-tool command-surface contract for the current TinyCloud emulation scope, including the narrower Terraform-feasible portion of that scope
3. `#8` Verified Terraform integration once Terraform is available in CI
4. `#9` PowerShell-free wrapper/runtime orchestration for normal cross-platform CLI usage
5. `#10` Phase 1 distribution foundation: GitHub Releases, GHCR runtime image, bootstrap scripts, and `tinycloud setup` / `tinycloud setup --full`

## Full Remaining Ordered Sequence

This section is the full remaining roadmap tail. Completing the short `Suggested Next Sequence` does not mean the roadmap is finished unless every item below is also completed, explicitly deferred, or removed from `plan.md`.

6. Implement standalone `tinyaz` as the Azure CLI compatibility analogue to `azlocal`, with full wrapper coverage across all 18 current TinyCloud emulation-scope areas
7. Define and document the final per-tool command-surface compatibility contract for `tinyterraform` and `tinyaz` across the current TinyCloud emulation scope and implement changes if needed, with the Terraform contract explicitly limited to the Terraform-feasible portion of that scope
8. Verified Terraform integration once Terraform is available in CI
9. PowerShell-free wrapper/runtime orchestration for normal cross-platform CLI usage
10. Phase 1 distribution foundation: GitHub Releases, GHCR runtime image, bootstrap scripts, and `tinycloud setup` / `tinycloud setup --full`
11. Phase 2 distribution convenience: package managers, managed tool cache, CLI-driven dependency installation/update flows, and environment diagnostics
12. Phase 3 polished distribution: native installers, signing, update checks, and product-grade installer UX
13. Queue Storage poison/dead-letter behavior where it materially improves real worker workflows
14. Blob event notification hooks only if a real workflow needs them
15. Key Vault certificates only if a real workflow needs them
16. Private Endpoints for supported services
17. Azure Functions local trigger/runtime helpers
18. Function App ARM resource and deployment helpers
19. App Service / Web App resource shell
20. Container Registry subset
21. Compose-first local workflow
22. Managed identity scenario presets for app-to-service testing
23. Additional deployment-template coverage for the already implemented providers, but only when a real workflow needs it
24. Further Blob compatibility refinement, but only for concrete SDK/tooling gaps
25. Container Apps or deeper App Service workflow support only if real workflows require it
26. Load Balancer / public IP modeling only if real workflows require it

## Current Recommendation

The next smallest useful product step is now roadmap item `#6`: implement standalone `tinyaz` with full wrapper coverage across all 18 current TinyCloud emulation-scope areas now that `tinyterraform` convergence under `#5` is complete. The follow-on `#7` contract step should then lock the final per-tool guarantees for that same emulation scope, with `tinyterraform` explicitly limited to the Terraform-feasible portion of it. After that, roadmap item `#9` should remove PowerShell as a hard dependency for normal CLI usage by moving the remaining wrapper/runtime orchestration into the Go command layer, and roadmap item `#10` should introduce the first public bootstrap and `tinycloud setup` / `tinycloud setup --full` install flow.

Migration note:

- the repo root now works as the effective Go workspace/build root, first-class Docker build home, and first-class wrapper-script home for `tinycloud`, `tinycloudd`, and `tinyterraform`, with top-level runtime-artifact and cache defaults for the transition wrappers
- the top-level command entrypoints now exist at `tinycloud\cmd`
- the shared product-command entry layer now exists at repo-root `tinycloud\cli\...`, with Azure-backed runtime adapters now under `tinycloud\azure\runtime\...`
- a new public runtime config/service-selection contract now exists at `tinycloud\azure\runtime\tinycloudconfig`, and the runtime now exposes machine-readable `/_admin/runtime` plus `/_admin/services` state for later CLI reconnection
- the shared `tinycloud` lifecycle/status/config/service UX now exists for both the managed process backend and the default Docker-backed local workflow, including persisted runtime metadata plus restart-mediated service selection
- the repo-root Dockerfile and `tinycloud start` path now agree on a real container-oriented local workflow
- the approved terminal-only polish slice is now complete: startup banner only on `start`, tables for runtime/services/endpoints, grouped config output, and interactive status-icon color treatment without changing JSON contracts
- next continue contiguous roadmap execution from `#6` through `#10`
- then resume the remaining ordered sequence from `#11` onward without introducing side tracks

Terraform note:

- The repo contains Terraform example material and `tinycloud env terraform` output.
- Real `terraform init`, `apply`, and `destroy` have now been manually verified for the `azurerm_resource_group` example through `scripts/tinyterraform.ps1`.
- Documentation should describe Terraform support as wrapper-driven and manually verified until automated CI coverage is added.

Distribution note:

- The public packaging/install direction is now documented:
  - bootstrap `tinycloud` from a TinyCloud-hosted installer URL
  - run `tinycloud setup --full`
  - let the CLI validate or install the full local suite
- That install story is planned, not implemented today.
- Roadmap items `#10` through `#12` now track the staged distribution work:
  - Phase 1 foundation
  - Phase 2 convenience
  - Phase 3 polished installers and update UX

## Completed Slices

- `072c55c` Converge tinyterraform az shim ownership
- `e02428f` Move terraform az shim into launcher
- `9ca6cad` Move terraform cert trust into launcher
- `20d17f5` Harden terraform wrapper health probe retry
- `20b9cea` Prebuild tinycloud helper for runtime wrapper flow
- `0b96877` Inject terraform exe into runtime wrapper flow
- `262259c` Own terraform override lifecycle in launcher
- `7f979db` Move terraform runtime startup into launcher
- `01a5764` Move terraform hosts routing into launcher
- `6adb0b6` Use unique tinyterraform helper binaries
- `cecfb4e` Delegate wrapper terraform init to launcher
- `f99b960` Make tinyterraform init launcher-owned
- `c71eca7` Trim endpoints from tinycloud start output
- `210cb90` Differentiate service status and inventory views
- `1a456fe` Default tinycloud start to detached mode
- `0b3f737` Format interactive TinyCloud logs
- `d30ffc9` Colorize terminal status icons only
- `605fc13` Polish tinycloud terminal UX
- `040dcaf` Add Docker tinycloud runtime backend
- `8f0442b` Implement tinycloud managed process CLI
- `80f581e` Add TinyCloud service selection runtime contract
- `d690a69` Fix terraform example command paths
- `6a09db4` Remove obsolete Azure command packages
- `eb8b85c` Extract shared command entry layer
- `ea76317` Add top-level tinyterraform command entrypoint
- `7d03006` Add top-level tinycloud command entrypoints
- `09b3094` Validate repo-root Go run entrypoints
- `f408cd9` Default repo-root wrapper Go cache
- `3ffb3c7` Resolve repo-root wrapper command paths
- `f316200` Make repo-root tinyterraform self-contained
- `afced6b` Default repo-root tinyterraform runtime
- `b22f221` Build repo-root wrapper binaries locally
- `67e9a4a` Add repo-root tinycloudd wrapper
- `bd1f919` Add repo-root tinycloud wrapper
- `550353b` Add repo-root tinyterraform wrapper
- `1e5fe2c` Add repo-root Dockerfile
- `f0ff5a2` Support repo-root Docker builds
- `6756471` Document repo-root tinyterraform init flow
- `9065179` Document repo-root tinyterraform usage
- `9cdbdc7` Add repo-root Go workspace
- `f775638` Track plan as the repo roadmap
- `faeea2b` Document runtime defaults and smoke tests
- `c4ec80c` Add HTTP middleware and shared responses
- `dc8270c` Add ARM core parsers
- `11ce562` Use SQLite for persistent state
- `a48e05f` Seed default ARM bootstrap records
- `56df100` Add ARM route skeleton
- `e1c3e7f` Implement resource group CRUD
- `d89f88c` Add ARM async operations
- `808d786` Add provider registration endpoints
- `8aa3ae4` Add identity and token endpoints
- `074973c` Expand metadata discovery
- `8a8b011` Add blob persistence layer
- `c09538c` Add blob data plane
- `11561bb` Snapshot blob state
- `0ecc1e8` Add storage account persistence
- `5169bcc` Add storage ARM routes
- `f295769` Add compatibility examples
- `e1ea1f9` Add deployment unsupported responses
- `bff4337` Add deployment persistence
- `e88136a` Track deployment failures
- `a47ace3` Refine metadata discovery
- `065d13c` Improve blob compatibility headers
- `121cf46` Add ARM tenant listing
- `a6ea05e` Refine endpoint discovery metadata
- `1a2f7aa` Enforce IMDS metadata headers
- `8fc0cf0` Add Key Vault persistence
- `c5c55bd` Add Key Vault ARM routes
- `d26a2cf` Add minimal deployment template support
- `ddd043d` Add Key Vault secret persistence
- `e07b663` Add Key Vault secret data plane
- `bba431a` Add queue storage persistence
- `a30d421` Add queue storage data plane
- `ce3596f` Add table storage persistence
- `fd6d752` Add table storage data plane
- `af6b7b3` Add service bus persistence
- `6a2cc12` Add service bus data plane
- `8e21f2b` Add service bus topic persistence
- `028bca0` Add service bus topics data plane
- `b3215e2` Add app configuration persistence
- `919ba0a` Add app configuration data plane
- `576995a` Add cosmos persistence
- `0452039` Add cosmos data plane
- `c301274` Add private DNS persistence
- `f778852` Add private DNS routes and resolver
- `f7ea2c7` Add event hubs persistence
- `9785cf0` Add event hubs data plane
- `b94b42a` Add virtual network persistence
- `6cd6656` Add virtual network ARM routes
- `d20ff73` Add network security group persistence
- `a3beef8` Add network security group ARM routes
- `279f6d2` Add Terraform examples and dev environment setup script
- `ad74b94` Add Terraform compatibility runtime
- `9c5e47e` Document LocalStack-style CLI compatibility
- `9f13d4c` Add first-class `cmd/tinyterraform` launcher entrypoint
- `660ecf2` Ignore generated Terraform runtime artifacts
- `e131301` Improve tinyterraform command passthrough
- `2ecb6bb` Normalize tinyterraform `go run` argument handling
- `45ef638` Preserve PowerShell `-chdir=...` argument handling in `tinyterraform`
- `af65054` Honor `TERRAFORM_EXE` in the first-class `tinyterraform` launcher
- `792425e` Preserve machine-readable stdout in direct `tinyterraform.ps1` usage
- `324b23f` Keep `terraform help` on the non-runtime passthrough path
- `1903549` Keep subcommand help flags on the non-runtime passthrough path
- `eba1990` Keep `terraform login` and `terraform logout` on the non-runtime passthrough path
- `8324f3b` Keep `terraform console` on the non-runtime passthrough path
- `162ae97` Add migration-friendly source-root and script overrides to `tinyterraform`
- `9fe3fbc` Add migration-friendly main-package and runtime-root overrides to `tinyterraform`
- `4b0558a` Add migration-friendly Go-workdir override to `tinyterraform`
- `76accf1` Add migration-friendly wrapper relative-path override to `tinyterraform`
- `6fa106c` Auto-discover TinyCloud source root from nested wrapper paths

## Session Resume Notes

- Use `GOCACHE=$PWD/.gocache` for Go commands in this environment.
- Repo-root Docker build command: `docker build -t tinycloud-azure .`
- Repo-root tinycloud wrapper command: `.\scripts\tinycloud.ps1 env pulumi`
- Repo-root tinycloudd wrapper command: `.\scripts\tinycloudd.ps1`
- Repo-root tinyterraform wrapper command: `.\scripts\tinyterraform.ps1 -chdir=.\azure\examples\terraform\resource-group init`
- Keep commits small and focused.
- Run a local review pass plus `go test ./...` before every commit.
