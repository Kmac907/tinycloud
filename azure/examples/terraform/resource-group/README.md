# Terraform Resource Group Example

Use the LocalStack-style wrapper from an elevated PowerShell session:

```powershell
$env:GOCACHE="$PWD\.gocache"
go run ..\..\..\cmd\tinyterraform -- init
go run ..\..\..\cmd\tinyterraform -- apply -auto-approve
go run ..\..\..\cmd\tinyterraform -- destroy -auto-approve
```

Equivalent direct wrapper flow:

```powershell
$env:GOCACHE="$PWD\.gocache"
..\..\..\scripts\tinyterraform.ps1 init
..\..\..\scripts\tinyterraform.ps1 apply -auto-approve
..\..\..\scripts\tinyterraform.ps1 destroy -auto-approve
```

Prerequisites:

- Terraform installed locally
- Windows PowerShell running as Administrator so the wrapper can temporarily map `management.azure.com` to TinyCloud
- Go installed locally so the wrapper can build and run the current TinyCloud binary

`cmd/tinyterraform` is the current first-class launcher entrypoint. On Windows it locates and invokes `scripts/tinyterraform.ps1`, which is the TinyCloud equivalent of `tflocal`: it invokes the real `terraform` binary, starts TinyCloud when needed, injects Azure CLI compatibility for auth, temporarily maps `management.azure.com` to TinyCloud's local HTTPS management listener, and cleans up the temporary hosts-file mapping when Terraform exits. Commands that actually need TinyCloud runtime routing still require an elevated session today; pure local passthrough commands like `terraform help` and `terraform version` do not. Terraform global flags such as `-chdir=...` are preserved by the launcher and wrapper so standard CLI invocation patterns still work, including PowerShell invocation. Both entrypoints also honor `TERRAFORM_EXE` when you need to force a specific Terraform binary path, and direct wrapper usage now preserves Terraform stdout for machine-readable commands.

`tinyterraform.ps1 init` also resets the TinyCloud runtime state before Terraform init so stale emulator resources do not survive failed local applies.

The long-term direction is to keep this flow as close as practical to normal `terraform` and `az` usage so users can rely on familiar command-line habits inside the local emulator environment.
