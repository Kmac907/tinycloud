# Pulumi Environment Example

From `tinycloud\`, start TinyCloud and print the local environment values through the top-level command entrypoint:

```powershell
go run .\cmd\tinycloud env pulumi
```

The repo-root wrapper also exposes the same environment output from `tinycloud\`:

```powershell
.\scripts\tinycloud.ps1 env pulumi
```

Use the printed values to configure a local Azure-native or ARM-based Pulumi program. The minimum values are:

- `ARM_ENDPOINT`
- `ARM_METADATA_HOST`
- `ARM_SUBSCRIPTION_ID`
- `ARM_TENANT_ID`

The Blob endpoint and mock OAuth token URL are also printed so SDK-based code can discover the implemented local services.
