# Contributor License Agreement Policy

This repository requires a Contributor License Agreement (CLA) for significant contributions.

## Why This Exists

TinyCloud keeps the public MVP in this open repository while also maintaining a separate private commercial expansion.

The CLA requirement exists to keep contribution provenance clear and to ensure the project can continue to:

- distribute the open-source MVP under Apache License 2.0
- accept external contributions without ownership ambiguity
- develop and distribute separate commercial editions without unclear rights around inbound contributions

## When A CLA Is Required

A CLA is required before merge for significant contributions, including:

- new features
- substantial code changes
- non-trivial architectural changes
- major documentation additions or rewrites

Maintainers may waive the CLA requirement for narrow changes such as:

- typo fixes
- small wording corrections
- minor documentation cleanup
- mechanical refactors with no material design change

## What The CLA Must Cover

The signed CLA process for this project should confirm that the contributor:

- has the right to submit the contribution
- grants the project the necessary copyright license to use, modify, distribute, and sublicense the contribution
- grants the necessary patent license for the contribution
- understands that the public repository remains Apache-2.0 licensed
- understands that the project may also use accepted contributions in separate commercial distributions

## Current State

Wave 1 manual CLA handling is now the active process.

Current public repo files:

- policy: [CLA.md](CLA.md)
- individual agreement text: [cla/individual-cla.md](cla/individual-cla.md)
- pull request review prompt: [.github/pull_request_template.md](.github/pull_request_template.md)
- maintainer runbook: [MAINTAINERS.md](MAINTAINERS.md)

Current maintainer workflow:

1. decide whether the contribution is significant
2. if it is significant, send the contributor the individual CLA text
3. obtain explicit written agreement before merge
4. store the signed or accepted CLA in a private maintainer-controlled location, not in the public repository
5. merge only after the maintainer has confirmed the CLA is on file

The repository still does not have an automated signing flow or CLA bot. Until that exists, maintainers should continue using the manual process above for significant external contributions.
