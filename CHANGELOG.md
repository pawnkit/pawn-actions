# Changelog

## 1.8.4 - 2026-07-29

- Use the exact signer workflow without passing a conflicting signer option.

## 1.8.3 - 2026-07-29

- Grant the release-set smoke job access to artifact attestations.

## 1.8.2 - 2026-07-29

- Verify the signed five-tool set in the compatibility workflow.

## 1.8.1 - 2026-07-29

- Pass the workflow token to GitHub's attestation verifier.

## 1.8.0 - 2026-07-29

- Cryptographically verify v3 build attestations against the recorded signer,
  tag, commit, and artifact.
- Require SBOM and provenance records for every v3 artifact.

## 1.7.1 - 2026-07-29

- Exercise CLI toolchain reporting in the compatibility smoke matrix.
- Keep release-set v2 immutable and validate supply-chain records in v3.

## 1.7.0 - 2026-07-29

- Validate release-set v2 SBOM and build-attestation records.
- Verify referenced SBOM release assets with their recorded checksums.

Notable changes are recorded here.

## 1.6.13 - 2026-07-29

### Changed

- Promote the hardened July 29 toolchain set.

## 1.6.12 - 2026-07-29

### Changed

- Test the hardened formatter and linter release candidate.

## 1.6.11 - 2026-07-29

### Fixed

- Pass the native temporary path to the run-protocol fixture on Windows.

## 1.6.10 - 2026-07-29

### Changed

- Promoted the tested July 29 release set to the default toolchain.

## 1.6.9 - 2026-07-29

### Fixed

- Build the run fixture from the checked-out Actions module.

## 1.6.8 - 2026-07-29

### Fixed

- Use the formatted golden-project release in the integration matrix.

## 1.6.7 - 2026-07-29

### Fixed

- Compare standalone lint with the same manifest entry used by `pawn check`.

## 1.6.6 - 2026-07-29

### Changed

- Run the July 29 toolchain candidate across the golden projects.
- Check project tests and the run protocol in the Linux integration job.

## 1.6.5 - 2026-07-29

### Added

- Added the run-backend fixture for toolchain integration tests.

## 1.6.4 - 2026-07-26

### Fixed

- Updated the installer CI matrix to the current tested linter release.

## 1.6.3 - 2026-07-26

### Changed

- Smoke-test the latest CLI, formatter, linter, language server, and test runner releases.
- Use the July 26 tested release set as the default for compatibility smoke tests.

## 1.6.2 - 2026-07-25

### Changed

- Publish the repository support record and validate it in CI.

## 1.6.1 - 2026-07-25

### Fixed

- Set up the validator's Go version inside the support-check action.

## 1.6.0 - 2026-07-25

### Added

- Added support-record validation against repository CI targets.

## 1.5.4 - 2026-07-25

### Fixed

- Avoid downloading every tool archive before the smoke matrix downloads them.

## 1.5.3 - 2026-07-25

### Changed

- Validate the release-set v2 dependency candidate in the toolchain smoke test.

## 1.5.2 - 2026-07-25

### Fixed

- Record separate dependency graphs when released tools use different module versions.

## 1.5.1 - 2026-07-25

### Fixed

- Verify every module graph tag and commit before publication.

## 1.5.0 - 2026-07-25

### Added

- Validate release-set v2 module graphs, dependency layers, and cycles.

## 1.4.8 - 2026-07-25

### Changed

- Smoke the current linter and language server release candidate.

## 1.4.7 - 2026-07-25

### Changed

- Smoke CLI v1.3.0 and build-backend schema v2.

## 1.4.6 - 2026-07-24

### Fixed

- Use the compiler archive's actual library directory.

## 1.4.5 - 2026-07-24

### Fixed

- Install the 32-bit runtime required by the pinned Pawn compiler.

## 1.4.4 - 2026-07-23

### Changed

- Smoke PawnKit CLI v1.2.0 on Linux, Windows, and macOS.

## 1.4.3 - 2026-07-23

### Fixed

- Load the compiler library from the verified Pawn release archive.

## 1.4.2 - 2026-07-23

### Fixed

- Print failed build reports in CI when a result file is configured.

## 1.4.1 - 2026-07-23

### Added

- Added compiler-backed build coverage for both golden corpus projects.

## 1.4.0 - 2026-07-23

### Added

- Added a build action for direct compilers and RFC 0012 backends.

## 1.3.3 - 2026-07-23

### Changed

- Switched the default smoke test to the promoted toolchain release set.

## 1.3.2 - 2026-07-23

### Fixed

- Fixed the pawntest include check in the release-set smoke workflow.

## 1.3.1 - 2026-07-23

### Changed

- Expanded the release-set smoke test to the primary command-line tools.

## 1.3.0 - 2026-07-23

### Added

- Added a verified installer for PawnKit tool archives.
- Added Linux, Windows, and macOS installer coverage.

## 1.2.2 - 2026-07-23

### Changed

- Made the v0.1.10 tested release set the smoke workflow default.

## 1.2.1 - 2026-07-23

### Fixed

- Fixed setup and check script paths on Windows runners.

## 1.2.0 - 2026-07-23

### Added

- Added release artifact selection for Actions outputs.
- Added a three-platform smoke workflow for the pinned CLI and corpus projects.

## 1.1.2 - 2026-07-23

### Added

- Added an action and reusable workflow for validating a pinned release set.

## 1.1.1 - 2026-07-23

### Added

- Added remote checks for component tags, schemas, workflow evidence, and
  release artifacts.

## 1.1.0 - 2026-07-23

### Added

- Added tested release-set validation and artifact checks.

## 1.0.1 - 2026-07-23

### Fixed

- Updated workflow examples to install PawnKit CLI `v1.0.3`.

## 1.0.0 - 2026-07-19

### Added

- Verified installation of published `pawn` archives.
- Check, format, lint, and test actions.
- Reusable check, SARIF, corpus, Go CI, and release workflows.
