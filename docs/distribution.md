# Distribution

This page documents the intended TinyCloud packaging, bootstrap, and release model.

Current state:

- `tinycloud` and `tinyterraform` can be built locally from source today
- the Docker runtime can be built and run locally today
- the official bootstrap and `tinycloud setup --full` flow is planned, not implemented yet

## Target Distribution Model

TinyCloud should ship as a layered product:

1. Docker image for the runtime
2. Native CLI binaries for the user-facing command surface
3. A one-command bootstrap that installs the `tinycloud` CLI
4. A `tinycloud setup --full` flow that installs or validates the rest of the local toolchain

PowerShell should not remain a hard dependency for normal product usage. It may remain as a transitional compatibility path, but the intended end state is binary-first and cross-platform.

## Release Artifacts

Each release should publish:

- `tinycloud` binary
- `tinyterraform` binary
- future `tinyaz` binary
- SHA256 checksums
- release notes
- Docker image tags

Target binary platforms:

- Windows x64
- Windows arm64
- macOS x64
- macOS arm64
- Linux x64
- Linux arm64

Docker image tags should include:

- `ghcr.io/<org>/tinycloud-azure:<version>`
- `ghcr.io/<org>/tinycloud-azure:latest`

## Official Install Story

The intended official install story is:

```bash
curl -fsSL https://get.tinycloud.dev/install.sh | sh
tinycloud setup --full
tinycloud start
```

Windows:

```powershell
irm https://get.tinycloud.dev/install.ps1 | iex
tinycloud setup --full
tinycloud start
```

Current note:

- this bootstrap-plus-setup flow is planned
- today, installation is still manual through source builds and local PATH setup

## Bootstrap Script Scope

The bootstrap scripts should stay small and auditable.

They should only:

1. detect OS and architecture
2. download the correct released `tinycloud` binary from GitHub Releases
3. place it in a user-local install directory
4. optionally add that directory to `PATH` for the current shell or print permanent PATH instructions
5. print the next command:
   - `tinycloud setup --full`

They should not contain the full install logic for Docker, Terraform, Azure CLI, or runtime configuration.

## Planned `tinycloud setup` Commands

The intended install command surface is:

- `tinycloud setup`
- `tinycloud setup --full`

Planned meaning:

- `tinycloud setup`: validate and prepare the local TinyCloud environment
- `tinycloud setup --full`: verify Docker, pull or validate the runtime image, initialize config/data directories, validate wrapper tool dependencies, and later manage supported upstream tool versions

These commands are part of the roadmap and are not implemented today.

## TinyCloud-Owned Tools Directory

A TinyCloud-owned tools directory means TinyCloud manages its own copies of supported upstream tools instead of depending entirely on whatever is globally installed on the machine.

Examples:

- Windows: `%LOCALAPPDATA%\tinycloud\tools\terraform\<version>\terraform.exe`
- macOS/Linux: `~/.local/share/tinycloud/tools/terraform/<version>/terraform`

Why this is useful:

- reproducible tool versions
- fewer support issues caused by user PATH drift
- simpler one-command onboarding
- better supportability for the tested toolchain

Recommended model:

- prefer TinyCloud-managed tested versions by default
- allow override with config or environment variables
- still permit advanced users to point TinyCloud at system-installed binaries

## Dependency Strategy

### Near-term

Use an external dependency model first:

- `tinyterraform` requires local Terraform
- future `tinyaz` requires local Azure CLI `az`

### Long-term

Move toward a managed toolchain model:

- `tinycloud setup --full` downloads or validates tested versions of required upstream tools
- TinyCloud stores them in its own managed tools directory
- system-installed binaries remain optional overrides

## Docker Runtime Distribution

Docker should be treated as the runtime artifact, not the main user UX.

Recommended model:

- publish the runtime image to GHCR
- keep `tinycloud start` as the normal way users start the runtime
- let the CLI pull and run the correct image automatically
- document raw `docker run` as an advanced path, not the primary install story

## Hosting Strategy

The bootstrap scripts should be served from the TinyCloud website domain.

Preferred URLs:

- `https://get.tinycloud.dev/install.sh`
- `https://get.tinycloud.dev/install.ps1`

Alternative if needed:

- `https://tinycloud.dev/install.sh`
- `https://tinycloud.dev/install.ps1`

The `get.` subdomain is preferred because it keeps the installer endpoint separate from the main site.

Recommended hosting split:

- Hostinger site or subdomain: bootstrap scripts only
- GitHub Releases: native CLI binaries and checksums
- GHCR: Docker runtime images

## Site Repo Layout

Bootstrap scripts should be version-controlled in the main site repo, then deployed as static files.

Recommended source layout:

- `public/install.sh`
- `public/install.ps1`

If using `get.tinycloud.dev`, the deployed folder for that subdomain should contain those files, or mirror those files from the main site build.

## Packaging Roadmap

### Phase 1

Goal:

Ship a usable open-source install and release story.

Deliver:

- GitHub Releases with `tinycloud` and `tinyterraform`
- checksums and release notes
- GHCR Docker image for the runtime
- install documentation for direct binary install
- bootstrap scripts:
  - `install.sh`
  - `install.ps1`
- initial `tinycloud setup`
- initial `tinycloud setup --full`
- dependency detection for:
  - Docker
  - Terraform
  - later `az`, once `tinyaz` exists

### Phase 2

Goal:

Reduce manual install friction and improve reproducibility.

Deliver:

- package manager support:
  - Homebrew tap
  - winget
  - Scoop
- managed tool cache for tested upstream tools
- CLI-driven install and update flow for:
  - Docker image
  - `tinyterraform` dependencies
  - future `tinyaz` dependencies
- version validation and compatibility reporting
- `tinycloud doctor` or equivalent environment diagnostics

### Phase 3

Goal:

Turn distribution into a polished product install story.

Deliver:

- native installers:
  - Windows MSI
  - macOS pkg
  - Linux deb/rpm if justified
- signed binaries and signed installers
- auto-update checks
- optional managed runtime and tool updates
- commercial-grade installer UX
- clearer split between open-source and paid-edition artifacts if needed

## Summary

Recommended end state:

- native `tinycloud` CLI on `PATH`
- Docker runtime managed by the CLI
- upstream tools managed by TinyCloud or explicitly detected and guided
- bootstrap scripts served from your own domain
- GitHub Releases for binaries
- GHCR for container images
- PowerShell optional only, not required for normal product usage
