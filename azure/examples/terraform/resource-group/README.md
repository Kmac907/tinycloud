# Terraform Resource Group Example

Use the LocalStack-style wrapper from an elevated PowerShell session:

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

`tinyterraform.ps1` is the TinyCloud equivalent of `tflocal`: it invokes the real `terraform` binary, starts TinyCloud, injects Azure CLI compatibility for auth, temporarily maps `management.azure.com` to TinyCloud's local HTTPS management listener, and cleans up the temporary hosts-file mapping when Terraform exits.

`tinyterraform.ps1 init` also resets the TinyCloud runtime state before Terraform init so stale emulator resources do not survive failed local applies.

The long-term direction is to keep this flow as close as practical to normal `terraform` and `az` usage so users can rely on familiar command-line habits inside the local emulator environment.
