# Security

- Setup accepts HTTPS archives only.
- Every archive requires an exact SHA-256 digest.
- Cached binaries must match the archive digest marker and requested version.
- Archive paths are checked before extraction.
- Workflows use least-privilege permissions.
- Third-party actions are pinned to full commit SHAs.
- Normal checks do not execute native plugins or components.

Report vulnerabilities through GitHub's private
[security advisory form](https://github.com/pawnkit/pawn-actions/security/advisories/new).
Do not include live credentials in the report.
