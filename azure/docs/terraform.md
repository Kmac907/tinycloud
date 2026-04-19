# Terraform

The current repo includes a Terraform example for `azurerm_resource_group` under `examples/terraform/resource-group`.

## Current Status

- the repo contains a Terraform example, `tinycloud env terraform` output, a first-class launcher at `cmd/tinyterraform`, and a Windows wrapper script at `scripts/tinyterraform.ps1`
- Terraform is required locally; TinyCloud does not bundle it
- the supported local flow is the first-class `tinyterraform` launcher plus the wrapper-backed privileged runtime path, not a raw `terraform apply` against `azurerm`
- the wrapper has been manually verified end to end for `init`, `apply`, and `destroy` against `azurerm_resource_group`
- the current officially supported Terraform compatibility subset is still narrow: the verified `azurerm_resource_group` example plus non-runtime passthrough commands such as `help`, `version`, `login`, `logout`, `console`, and subcommand help
- the roadmap direction is to keep promoting `tinyterraform` toward a first-class Model 2 compatibility command across the current TinyCloud emulation scope wherever credible Terraform provider/resource coverage exists, rather than claiming blanket `azurerm` parity beyond what real Terraform can support
- official Terraform provider coverage is broader than the current `tinyterraform` verified subset; many additional TinyCloud service/resource families are Terraform-feasible in principle, but TinyCloud has not yet validated and locked that broader contract
- the current PowerShell wrapper is a transitional compatibility path, not the intended long-term product dependency model; normal TinyCloud usage is intended to converge on compiled cross-platform CLI binaries

## Provider Shape

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

## Typical Flows

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

The repo root also exposes the wrapper script directly:

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

## Current Wrapper Behavior

`cmd/tinyterraform` is the current first-class launcher entrypoint. On Windows it now owns the local `terraform init` reset/bootstrap path directly in the shared Go command layer, and direct wrapper `init` through either `scripts\tinyterraform.ps1` entrypoint now delegates into that same launcher-owned path instead of keeping a separate inline init implementation in the wrapper. For the broader privileged runtime-routing path, the launcher now also prebuilds the current `tinycloud` helper, resolves the real `terraform` binary, creates the temporary Terraform provider override in the actual Terraform working directory, starts plus waits for the isolated TinyCloud runtime, manages the temporary `management.azure.com` hosts-file mapping, imports the generated local HTTPS management certificate into the current user trust store, and prepares the launcher-owned local `az` compatibility shim before invoking `scripts\tinyterraform.ps1`. The direct-wrapper path now reuses that same shared `az` shim source instead of carrying its own separate inline Azure CLI shim contract, so `tinyterraform` now has one shared ownership model for the supported runtime-routing compatibility path. The current Azure CLI compatibility layer still lives in `tinyterraform` today; the roadmap direction is now to split that into a standalone `tinyaz` helper while aiming `tinyaz` at all 18 current TinyCloud emulation-scope areas, and aiming `tinyterraform` at the Terraform-feasible portion of that same emulation scope. Commands that actually need TinyCloud runtime routing beyond `init` still require an elevated PowerShell session today because the flow still needs temporary hosts-file routing; pure local passthrough commands like `terraform help`, `terraform version`, `terraform login`, `terraform logout`, `terraform console`, and subcommand help requests like `terraform apply -help` do not. Terraform global flags such as `-chdir=...` are preserved by the launcher and wrapper so normal CLI invocation patterns continue to work, including PowerShell invocation. Both entrypoints also honor `TERRAFORM_EXE` when you need to point TinyCloud at a specific Terraform binary, and the wrapper preserves Terraform stdout for machine-readable commands like `version -json`.

That current PowerShell-backed privileged path is transitional. The explicit portability goal is to move the remaining wrapper/runtime orchestration into the Go command layer so normal `tinyterraform` usage does not depend on PowerShell.

With that convergence complete, roadmap item `#5` is complete and the next wrapper roadmap step is `#6`: introduce standalone `tinyaz` with full wrapper coverage across all 18 current TinyCloud emulation-scope areas.

## Endpoint Routing Note

- `tinyterraform` is currently ARM-first, not a universal per-service endpoint router.
- The generated provider override only injects `provider "azurerm" { use_cli = true ... }`; it does not rewrite every Azure service endpoint into one generic TinyCloud URL.
- The wrapper obtains local ARM credentials through the local `az` shim and the TinyCloud OAuth endpoint.
- For runtime-routed flows such as `apply` and `destroy`, the wrapper does not currently point AzureRM straight at `ARM_ENDPOINT`; instead it temporarily maps `management.azure.com` to `127.0.0.1`, starts TinyCloud HTTPS management on local port `443`, and trusts the generated certificate so the provider can keep using its normal Azure management host shape.
- `tinycloud env terraform` currently emits ARM management settings such as `ARM_ENDPOINT`, `ARM_METADATA_HOST`, `ARM_METADATA_HOSTNAME`, `ARM_MSI_ENDPOINT`, `ARM_SUBSCRIPTION_ID`, `ARM_TENANT_ID`, and `TINY_MGMT_HTTPS_CERT`, plus a small set of explicit service endpoint hints such as `TINY_BLOB_ENDPOINT`, `TINY_APPCONFIG_ENDPOINT`, `TINY_COSMOS_ENDPOINT`, `TINY_DNS_SERVER`, `TINY_EVENTHUBS_ENDPOINT`, and `TINY_OAUTH_TOKEN`.
- In the current wrapper implementation, `tinycloud env terraform` is used primarily to obtain subscription, tenant, and certificate/bootstrap state for the runtime path; those extra `TINY_*` service endpoint hints are available for manual tooling and future parity work, but are not yet broadly propagated by `tinyterraform` into the Terraform child process as automatic per-service provider routing.
- Separate service endpoints still exist and are advertised by TinyCloud itself. For example ARM storage-account responses include `properties.primaryEndpoints.blob`, and Key Vault ARM responses include `properties.vaultUri`.
- That means the current architecture can support provider flows that use ARM first and then discover a service-specific endpoint from ARM or explicit environment, but broad automatic per-service Terraform compatibility is not yet claimed or verified.
- The currently verified Terraform compatibility target remains the ARM-side `azurerm_resource_group` flow. Additional service-specific Terraform parity belongs to the explicit later `tinyterraform` expansion step after the per-tool contract is locked, and remains limited by what real Terraform provider/resource coverage can support.

## Future Scope Guidance

The strongest future `tinyterraform` targets are the resource-oriented Azure families that already have clear Terraform provider coverage and that TinyCloud already implements in some form. The most likely supported families are:

- ARM resource groups
- storage accounts
- Blob containers
- storage queues
- storage tables and table entities
- Key Vault resources and secrets
- virtual networks and subnets
- network security groups and rules
- private DNS zones and A records
- Service Bus namespaces, queues, topics, and subscriptions
- Event Hubs namespaces, hubs, and consumer groups
- selective App Configuration, Cosmos DB, and deployment-template-backed resources once the real provider contract is verified against TinyCloud

That future scope is broader than today's verified support, but it is still not the same as blanket AzureRM parity.

The primary things `tinyterraform` should not treat as its main future contract are live operational objects such as:

- queue messages
- Service Bus messages
- event payload publishing or consumption
- Cosmos document CRUD

Those workflows are generally better treated as application-client, SDK, direct API, or later command-specific compatibility scenarios rather than the primary Terraform compatibility surface.

## Notes And Overrides

`tinyterraform init` resets the TinyCloud runtime state before running Terraform init. That keeps emulator state and Terraform state aligned after failed local applies.
`tinyterraform init` uses that local reset/bootstrap path but does not need the hosts-file routing that `apply` and `destroy` still require.

For compatibility and repo-layout variation handling, both entrypoints also support explicit path overrides:

- `TINYCLOUD_SOURCE_ROOT` points the wrapper at the TinyCloud source tree it should build and run
- `TINYTERRAFORM_SCRIPT` points the Go launcher at a specific `tinyterraform.ps1` script path
- `TINYTERRAFORM_SCRIPT_RELATIVE_PATH` points the Go launcher at the wrapper script relative to `TINYCLOUD_SOURCE_ROOT`, which defaults to `scripts\tinyterraform.ps1` today
- `TINYCLOUD_MAIN_PACKAGE` points the wrapper at the TinyCloud Go package it should build; the wrappers still accept the older `tinycloud/cmd/tinycloud` form for migration compatibility, but the default repo-root paths now build the top-level `.\cmd\tinycloud` launcher and only fall back to `.\azure\cmd\tinycloud` when needed
- `TINYCLOUD_GO_WORKDIR` points the wrapper at the Go build/workspace directory it should run `go build` from
- `TINYTERRAFORM_RUNTIME_ROOT` points the wrapper at an isolated runtime directory instead of the default `.tinyterraform-runtime`
- `TINYTERRAFORM_HOSTS_PATH` points both the launcher and wrapper at an alternate hosts file path, which is mainly useful for isolated validation flows

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
