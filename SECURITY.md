# Security Policy

## Reporting A Vulnerability

Do not open a public issue for suspected security vulnerabilities.

Report security issues privately to the project maintainer through the repository owner contact path you were given for this project. If you do not already have a private contact path, ask for one before disclosing vulnerability details publicly.

When reporting a vulnerability, include:

- affected component or file path
- steps to reproduce
- expected impact
- environment details
- any proof-of-concept material needed to reproduce the issue

## Disclosure Expectations

- give the maintainer reasonable time to investigate and remediate the issue before public disclosure
- avoid publishing exploit details while a fix is still pending
- keep reports focused, reproducible, and minimal

## Scope Notes

This repository contains local emulator and wrapper tooling. Security-relevant reports may include:

- unsafe local credential or token handling
- local TLS or certificate trust issues
- privilege escalation through wrapper or runtime behavior
- unsafe hosts-file or process-management behavior
- exposed admin endpoints or local data leakage

## Supported Scope

Security review should focus on the current main branch and currently documented command surfaces.
