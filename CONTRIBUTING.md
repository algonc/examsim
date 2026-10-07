# Contributing to examsim

Contributions are welcome. Keep changes focused, preserve existing command behavior unless the change is intentional, and include tests for new behavior or bug fixes.

## Requirements

- Go 1.25 or newer
- Git
- GNU Make and `golangci-lint` for the Makefile checks (optional when running Go commands directly)

## Development Setup

CLI source and tests live in `cmd/examsim/`. The Go module and dependency files remain at the repository root.

Clone the repository and download its dependencies:

```sh
git clone https://github.com/algonc/examsim.git
cd examsim
go mod download
```

Build the CLI into the ignored `bin/` directory:

```sh
make build
```

Without Make, run `go build -trimpath -o bin/ ./cmd/examsim`.

Run it directly from source:

```sh
go run ./cmd/examsim -e examples/exam1.yaml
```

## Quality Checks

Run `make` to verify dependencies, run vet, lint, and tests, then build the CLI. Use `make check` for checks only, `make test-cover` for coverage, and `make clean` to remove build output. `GO`, `GOLANGCI_LINT`, and `BIN_DIR` can be overridden, for example `make build BIN_DIR=dist`.

Format changed Go files and run the project checks before submitting a change:

```sh
gofmt -w path/to/changed.go
go test ./...
go vet ./...
```

On a system with CGO enabled, also run the race detector for changes that affect concurrency, signals, input handling, or session persistence:

```sh
go test -race ./...
```

## Pull Requests

- Explain the user-visible behavior and motivation.
- Keep unrelated refactoring out of the change.
- Add or update tests in proportion to the behavioral risk.
- Update user documentation when flags, exam fields, output, or session behavior changes.
- Ensure generated binaries and release archives are not committed.

Release preparation and publication are documented separately in [RELEASING.md](RELEASING.md).
