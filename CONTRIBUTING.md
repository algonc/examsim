# Contributing to examsim

Contributions are welcome. Keep changes focused, preserve existing command behavior unless the change is intentional, and include tests for new behavior or bug fixes.

## Requirements

- Go 1.25 or newer
- Git

## Development Setup

Clone the repository and download its dependencies:

```sh
git clone https://github.com/algonc/examsim.git
cd examsim
go mod download
```

Build the CLI into the ignored `bin/` directory:

```sh
mkdir -p bin
go build -o bin/ .
```

Run it directly from source:

```sh
go run . -e examples/exam1.yaml
```

## Quality Checks

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
