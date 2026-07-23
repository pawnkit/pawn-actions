# pawn-actions

Use the same PawnKit checks in GitHub Actions that you run locally.

Start by installing a published `pawn` release, then run the check action:

```yaml
- uses: pawnkit/pawn-actions/setup@v1
  with:
    version: 1.0.0
    download-url: https://github.com/pawnkit/pawnkit-cli/releases/download/v1.0.3/pawn-linux-amd64.tar.gz
    sha256: 328acf396dff120b46cc41508d5f59c4a41852f2a85118260df65f635ca6c9c5
- uses: pawnkit/pawn-actions/check@v1
```

Setup requires an exact version, an HTTPS archive URL, and its SHA-256 digest.
The archive must contain only `pawn` at its root (`pawn.exe` on Windows). The
installed binary must report the requested version.

The `check`, `fmt`, and `lint` actions call `pawn check`; they do not maintain a
second copy of its behavior. The `test` action also needs a PawnKit-compatible
test backend such as `pawntest` already available on `PATH`.

Build with an explicit compiler:

```yaml
- uses: pawnkit/pawn-actions/build@v1
  with:
    project: .
    compiler: compiler/pawncc
    artifact: build/server.amx
```

Set `backend` instead of `compiler` to use an RFC 0012 build backend. The action
passes profile, build, runtime, and output choices to `pawn build`.

Reusable workflows are available under `.github/workflows`. They cover the
general check pipeline, SARIF upload, corpus conformance, Go library CI, and Go
release archives.

## Validate a release set

Validate a tested release set and the Go modules it names before publication:

```sh
go run ./cmd/pawn-release-set \
  -go-mod ../pawnkit-cli/go.mod \
  ../pawnkit-spec/examples/pawn-release-set/valid.json
```

Add `-verify-remote` before publication. It checks tag commits, hosted schemas,
workflow evidence, artifact sizes, and SHA-256 hashes. Use `-verify-artifacts`
when only the archives need checking.

The command also rejects unknown fields, local replacements, PawnKit
pseudo-versions, duplicate entries, and artifacts for untested targets.

Repositories can call the shared validation workflow:

```yaml
jobs:
  release-set:
    uses: pawnkit/pawn-actions/.github/workflows/release-set.yml@v1.2.2
    with:
      spec-ref: v0.1.10
      set-path: release-sets/preview-2026-07-23.json
```

The release-set action can also select an archive for a runner:

```yaml
- id: pawn
  uses: pawnkit/pawn-actions/release-set@v1.2.0
  with:
    release-set: release-sets/preview.json
    component: pawn
    target: linux-amd64
- uses: pawnkit/pawn-actions/setup@v1
  with:
    version: ${{ steps.pawn.outputs.version }}
    download-url: ${{ steps.pawn.outputs.url }}
    sha256: ${{ steps.pawn.outputs.sha256 }}
```

Use `setup-tool` for the other binaries selected from a release set:

```yaml
- id: pawnlint
  uses: pawnkit/pawn-actions/release-set@v1.3.0
  with:
    release-set: release-sets/preview.json
    component: pawnlint
    target: linux-amd64
- uses: pawnkit/pawn-actions/setup-tool@v1.3.0
  with:
    binary: pawnlint
    version: ${{ steps.pawnlint.outputs.version }}
    download-url: ${{ steps.pawnlint.outputs.url }}
    sha256: ${{ steps.pawnlint.outputs.sha256 }}
```

The installer accepts `.tar.gz` and `.zip` release archives. It verifies the
checksum and version before adding the tool to `PATH`. Pawntest includes are
installed beside its executable.

`release-set-smoke.yml` installs the selected CLI archive on Linux, Windows,
and macOS, then checks the small SA-MP and open.mp corpus projects.

Use the `v1` tag for compatible fixes, or pin a full commit when every action
update must be reviewed.

## Contributing

This is a community project. Workflow fixes and runner-specific tests are
welcome; see [CONTRIBUTING.md](CONTRIBUTING.md) before changing setup or archive
handling.

## Licence

[MIT](LICENSE)
