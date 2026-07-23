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

Add `-verify-artifacts` to download each listed archive and check its size and
SHA-256 hash. The command rejects unknown fields, local replacements, PawnKit
pseudo-versions, duplicate entries, and artifacts for untested targets.

Use the `v1` tag for compatible fixes, or pin a full commit when every action
update must be reviewed.

## Contributing

This is a community project. Workflow fixes and runner-specific tests are
welcome; see [CONTRIBUTING.md](CONTRIBUTING.md) before changing setup or archive
handling.

## Licence

[MIT](LICENSE)
