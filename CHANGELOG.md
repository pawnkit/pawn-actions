# Changelog

Notable changes are recorded here.

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
