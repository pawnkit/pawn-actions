# Architecture

The setup action installs a verified `pawn` binary. The other actions call the
same CLI commands used locally. Workflow YAML does not reproduce formatter,
linter, test, build, or project logic.

```text
setup -> checksum verification -> tool cache -> pawn
check -> pawn check
```

Paths and task names pass through environment variables and Bash arrays. The
scripts never evaluate them as shell source.

The `releaseset` package validates RFC 0009 documents. It checks pinned module
versions, public tag commits, workflow evidence, hosted schemas, and release
archives before a set is published. The schema remains in `pawnkit-spec`.
