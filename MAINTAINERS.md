# Maintainer Workflow

This file documents the current manual maintainer process for contributions, with emphasis on CLA handling for significant external changes.

## Significant Contribution Check

Treat a pull request as significant if it includes any of the following:

- new features
- substantial code changes
- non-trivial design or architecture changes
- major documentation additions or rewrites

Small typo fixes, narrow wording corrections, and similar cleanup changes may be treated as non-significant at maintainer discretion.

## Manual CLA Process

For significant external contributions:

1. decide that the pull request requires a CLA
2. reply to the contributor with the standard CLA message below
3. send or link the contributor to [cla/individual-cla.md](cla/individual-cla.md)
4. obtain explicit written acceptance before merge
5. store the signed or accepted CLA in a private maintainer-controlled location
6. record the contributor as `CLA on file`
7. confirm CLA status in the pull request
8. merge only after the above is complete

Signed CLA records must not be committed to the public repository.

## Standard Maintainer Message

Use this reply for significant external pull requests:

> Thanks. This change requires a CLA before merge. Please review `cla/individual-cla.md` and reply with:
>
> “I have read and agree to the TinyCloud Individual Contributor License Agreement in `cla/individual-cla.md`, and I agree that it applies to my current and future contributions to TinyCloud.”
>
> Once that agreement is received and recorded, the pull request can continue through merge review.

## Private Record Template

Maintain a private CLA registry outside the public repository.

Recommended fields:

```text
github_handle,name,email,cla_type,date_received,record_location,notes
```

Example:

```text
jdoe,Jane Doe,jane@example.com,individual,2026-04-18,private://cla-records/individual/jdoe-2026-04-18.pdf,accepted by email
```

## Storage Rules

- use one private storage location only
- keep contributor personal information out of the public repository
- keep the raw signed document or written acceptance alongside the registry entry
- retain enough information to show which CLA text version was accepted

## Current Limits

The repository does not yet have:

- a CLA bot
- a hosted signature flow
- an automated merge gate tied to CLA status

Until those exist, maintainers must continue using the manual process above.
