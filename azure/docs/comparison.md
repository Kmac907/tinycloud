# Comparison

This is the practical comparison for current use, not a marketing claim. The point here is where TinyCloud fits in the broader local cloud-emulator landscape.

| Tool | Cloud focus | Product shape | Strength | Tradeoff | Best fit |
| --- | --- | --- | --- | --- | --- |
| TinyCloud | Azure | focused local cloud emulator | combines ARM-style control plane, identity metadata, storage, secrets, and messaging in one small runtime | Azure coverage is still intentionally narrow | testing Azure workflows that need ARM plus several real data-plane services |
| Azurite | Azure Storage | storage emulator | mature Blob/Queue/Table emulation from Microsoft | no ARM, no identity, no broader Azure control plane | storage-only local development |
| LocalStack | AWS | broad local cloud platform | large AWS surface area and established local-cloud workflow patterns | AWS-focused rather than Azure-focused | teams standardizing on AWS local emulation |
| MiniStack | AWS | lightweight local cloud platform | fast, small-footprint AWS emulator with broad service ambitions | AWS-focused rather than Azure-focused | developers who want a lighter AWS local-cloud setup |

## Interpretation

- TinyCloud is closer in spirit to LocalStack and MiniStack than to Azurite: it aims to emulate a cloud environment, not just a single storage service.
- Azurite is the better choice when you only need Azure Storage and want broader storage coverage today.
- TinyCloud is the better fit when you need Azure-style resource provisioning, metadata/identity endpoints, and multiple local data-plane services together in one runtime.
- LocalStack and MiniStack are relevant peers because they define the broader local-cloud developer experience category, even though they target AWS instead of Azure.

## Comparison Sources

The comparison notes above are based on current upstream docs and homepages:

- LocalStack docs: https://docs.localstack.cloud/getting-started/installation/
- LocalStack overview/docs: https://docs.localstack.cloud/aws/enterprise/kubernetes/
- Azurite docs: https://learn.microsoft.com/en-us/azure/storage/common/storage-use-azurite
- Azurite + Storage Explorer docs: https://learn.microsoft.com/en-us/azure/storage/common/storage-explorer-emulators
- MiniStack homepage: https://ministack.org/
