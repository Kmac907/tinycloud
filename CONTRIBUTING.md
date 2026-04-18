# Contributing

Thanks for your interest in contributing to TinyCloud.

## Scope

The open repo is the public MVP for the project. Shared product and development docs live under [docs/](docs). Provider-specific implementation details and emulator-specific docs for the implemented Azure backend live under [azure/](azure).

## Before You Contribute

- open an issue or discussion first for non-trivial changes
- keep pull requests focused and narrowly scoped
- do not include generated runtime state, local validation artifacts, or secrets

## Basic Contribution Flow

1. Review the current product and implementation docs:
   - start with [README.md](README.md)
   - use [docs/development.md](docs/development.md) for local workflow and validation commands
   - for Azure-emulator-specific behavior, use [azure/README.md](azure/README.md) and [azure/docs/](azure/docs)
2. Open an issue or discussion before starting non-trivial work so scope and direction are clear.
3. Fork the repo or create a working branch from the current default branch.
4. Make a small, focused change that matches the documented roadmap and current product direction.
5. Run the relevant local validation before opening a pull request.
6. Update docs or examples in the same change when user-facing behavior changes.
7. Open a pull request with:
   - a clear summary of what changed
   - the validation you ran
   - any assumptions, limitations, or follow-up work
8. If the change is significant, complete the CLA process before merge.

## Pull Request Expectations

- keep the diff small and reviewable
- avoid unrelated cleanup in the same pull request
- include documentation updates when behavior or commands change
- include test or validation evidence for the changed behavior
- be explicit about anything still partial or intentionally deferred

## CLA Requirement

Significant contributions require a Contributor License Agreement (CLA) before they can be merged.

This requirement exists so the open-source MVP can stay cleanly licensed while the project also maintains a separate private commercial expansion.

Examples of significant contributions include:

- new features
- substantial code changes
- non-trivial design or architecture changes
- large documentation additions or rewrites

Small typo fixes, minor wording corrections, and similarly narrow cleanup changes may be accepted without a CLA at maintainer discretion.

See [CLA.md](CLA.md) for the current CLA policy.
See [cla/individual-cla.md](cla/individual-cla.md) for the current individual CLA text.
Maintainers should use [MAINTAINERS.md](MAINTAINERS.md) for the current manual CLA handling workflow.

## Maintainer CLA Flow

For significant external contributions:

1. decide whether the pull request is significant
2. if it is, do not merge until the contributor has completed the CLA
3. keep the signed or explicitly accepted CLA in a private maintainer-controlled location
4. confirm the CLA status in the pull request before merge

The public repository contains the policy and blank agreement text. Signed CLA records should not be committed to this repository.

## License

Unless explicitly agreed otherwise in writing, contributions accepted into this repository are made under the same Apache License 2.0 terms as the project. See [LICENSE](LICENSE).

## Trademarks And Branding

The Apache-2.0 license covers the code in this repository. It does not grant rights to use the TinyCloud name, logo, or project branding beyond reasonable descriptive use.

If you distribute modified versions, do not imply official project status, endorsement, or commercial affiliation through the TinyCloud name or branding.

## Development

For current local commands, smoke tests, and runtime validation flows, use [docs/development.md](docs/development.md).
