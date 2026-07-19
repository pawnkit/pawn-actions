# Contributing

PawnKit is maintained by volunteers, so reviews may take a little time.

Small workflow fixes and platform-specific test cases are welcome. Keep actions
thin: project, formatter, linter, and test behavior belongs in the tool that
owns it, not in workflow YAML.

Run the Go and shell tests before opening a pull request:

```sh
task check
```

Treat setup and archive changes as supply-chain work. Add a failure case for
bad checksums, unsafe paths, or unexpected archive layouts when relevant.
