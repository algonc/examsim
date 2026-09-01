# Releasing examsim

This document is for project maintainers. Releases are built and published by the [release workflow](.github/workflows/release.yml).

## Versioning

Use a new `vMAJOR.MINOR.PATCH` tag for each stable release, for example `v1.2.0`. A suffix such as `v1.2.0-rc.1` creates a GitHub prerelease.

Never move or reuse a version tag after pushing it. If released code needs a correction, make the change and publish a new version.

## Prepare the Release

1. Confirm the intended changes are committed on the default branch.
2. Update user documentation when behavior has changed.
3. Tidy and verify the module:

```sh
go mod tidy
git diff --exit-code -- go.mod go.sum
go mod verify
```

4. Run the local quality checks:

```sh
go test -count=1 ./...
go vet ./...
```

5. Ensure the working tree is clean:

```sh
git status --short
```

## Publish the Release

Create and push an annotated version tag:

```sh
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

The tag starts the release workflow. It performs the following operations:

1. Tests the tagged source on Linux, macOS, and Windows.
2. Runs the race detector on Linux.
3. Builds AMD64 and ARM64 binaries for Linux, macOS, and Windows.
4. Packages the executable with `LICENSE`, `README.md`, and `examples/exam1.yaml`.
5. Produces `checksums.txt` containing SHA-256 checksums.
6. Creates a GitHub release with automatically generated release notes.

Stable versions are eligible to become the latest release. Tags containing a prerelease suffix are published as prereleases.

## Verify the Release

After the workflow completes:

1. Confirm that all six platform archives and `checksums.txt` are attached to the GitHub release.
2. Download at least one archive and verify its checksum.
3. Extract the archive and run `examsim -help`.
4. Review the generated release notes before announcing the release.

The workflow does not overwrite an existing release. Infrastructure failures can be retried from GitHub Actions. If a code or packaging change is required, commit the correction and create a new version tag.
