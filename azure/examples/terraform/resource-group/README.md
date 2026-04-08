# Terraform Resource Group Example

Run TinyCloud, then export the environment printed by:

```powershell
$env:GOCACHE="$PWD\.gocache"
go run .\cmd\tinycloud env terraform
```

Prerequisites:

- Terraform installed locally
- TinyCloud running on the local management endpoint

From this directory:

```powershell
terraform init
terraform apply
```

This example is intended to target the local TinyCloud ARM endpoint and create one resource group.
