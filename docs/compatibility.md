# Compatibility

The setup action supports GitHub-hosted Linux, macOS, and Windows runners with
Bash, tar, and either `sha256sum` or `shasum`. Release archives must contain
only `pawn` on Unix or `pawn.exe` on Windows at the archive root.

Public action inputs and outputs remain compatible within a maintained major
tag. Pin a full commit SHA when updates must be reviewed individually.

Check, format, and lint do not run native plugins or components. The test action
needs a negotiated backend on `PATH` when native runtime work is required.
