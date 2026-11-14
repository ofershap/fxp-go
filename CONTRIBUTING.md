# Contributing

Issues and pull requests are welcome.

## Setup

Go 1.26 or newer. Standard library only; no other dependencies for the
library or tests. Regenerating the golden corpus needs Node (pinned
fast-xml-parser 5.12.1 in testdata/gen/package-lock.json) and is not part of
the normal test run.

## Checks

Run everything CI runs before opening a PR:

```bash
go vet ./...
gofmt -l .
go test -race ./...
```

## Rules

- Behaviour must match fast-xml-parser 5.12.1 byte-exactly, quirks included.
  If upstream changes, extend the corpus and regenerate goldens in the same
  change; intentional divergences get a pinned regression test.
- Keep the public API small and the dependency tree empty.
- Claims in the README must match what the tests demonstrate.
