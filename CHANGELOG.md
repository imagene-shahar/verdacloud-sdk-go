# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Added
- Add spot volume removal policy (`on_spot_discontinue`) and related constants
- Add `delete_permanently` field to `InstanceActionRequest`
- Add `InstanceActionResult` type for action responses and handle response code properly

### Changed
- **Breaking**: `Action` now takes `InstanceActionRequest` struct and returns `([]InstanceActionResult, error)`
- **Breaking**: `Delete` and `Discontinue` add `deletePermanently bool` parameter
- `Action` properly handles 202 (success), 204 (already in state), 207 (partial failure), 400, 404
- Remove `omitempty` from `VolumeIDs` to distinguish nil (API default) from empty array (no volumes)
- Bump minimum Go version to 1.25 (fixes GO-2026-4602, GO-2026-4601 stdlib vulnerabilities)
- Update CI test matrix from Go 1.23/1.24 to Go 1.24/1.25

## [v1.2.2] - 2026-02-25
### Changed
- chore(volumes): Add missing attributes for shared filesystem volumes

## [v1.2.1] - 2026-02-05
### Added
- feat(client): Add User-Agent header support with `WithUserAgent()` option
- feat(version): Add SDK version detection via `runtime/debug.ReadBuildInfo()`
- feat(release): Auto-update SDK version in `make release`

## [v1.2.0] - 2026-02-02
### Added
- feat(workflow): Establish changelog and automated release workflow
- feat(clusters): Integrate with cluster APIs
- feat(example): Add comprehensive cluster API usage examples

### Fixed
- fix(serverless-jobs): Fix serverless job validation
- fix(volumes): Migrate volume operations to PUT-based API (attach, detach, clone, resize, rename)
- fix(volumes): Fix CloneVolume to handle array response format
- fix(linter): Enable staticcheck for test files (ensure local/CI consistency)
- fix(test): Convert if-else chain to switch statement in volumes_test.go
- fix(types): Fix format specifiers after ComputeResource.Size type change (int)

### Changed
- refactor(instances): Remove legacy support for instance
- refactor(Makefile): Streamline development workflow (25 → 12 targets, -260 lines)
  - Single `setup` target: smart detection, installs only missing tools
  - `lint` and `fmt` kept for CI only (pre-commit handles locally)
  - Remove redundant targets: install-tools, setup-hooks, gosec, govulncheck, check, check-security, test, ci-local, clean-all
  - Remove unused Docker targets (200+ lines)
- refactor(CI): Fix workflow to install golangci-lint directly (removed make install-tools)
- test(integration): Add resource cleanup timers (2-15s based on resource type)
- test(integration): Improve integration tests

### Removed
- docs: Remove outdated coverage improvement plan

## [v1.1.3] - 2025-12-30
### Fixed
- fix(types): Change size of ComputeResource to an integer, add location and contract to volumes
- fix(serverless-jobs): Use correct type in scaling endpoints
### CI
- ci: add govulncheck to CI pipeline for branch protection

## [v1.1.2] - 2025-12-10
### Fixed
- fix(serverless-jobs): Change an incorrect scaling struct from container scaling options to job scaling options
### Other
- add release template

## [v1.1.1] - 2025-12-05
### Changed
- Container type restructuring

## [v1.1.0] - 2025-12-03
### Added
- feat!: Standardize location codes and complete serverless APIs
### Fixed
- Fix location_code in volume creation

## [v1.0.2] - 2025-11-19
### Fixed
- Fix instance actions to support single ID only and correct golangci-lint installation
- Fix GitHub workflows and upgrade to Go 1.24

## [v1.0.1] - 2025-11-17
### Fixed
- fix: resolve linting issues for CI compatibility

## [v1.0.0] - 2025-11-17
### Added
- Initial release
