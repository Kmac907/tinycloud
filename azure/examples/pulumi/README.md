# Pulumi Environment Example

Start TinyCloud and print the local environment values:

```powershell
go run .\cmd\tinycloud env pulumi
```

Use the printed values to configure a local Azure-native or ARM-based Pulumi program. The minimum values are:

- `ARM_ENDPOINT`
- `ARM_METADATA_HOST`
- `ARM_SUBSCRIPTION_ID`
- `ARM_TENANT_ID`

The Blob endpoint and mock OAuth token URL are also printed so SDK-based code can discover the implemented local services.
