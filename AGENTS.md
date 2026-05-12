# Repository Guidelines

## Project Structure & Module Organization

This repository contains a small Go CLI for adding IPv6 Segment Routing routes with optional TLV space.

- `main.go` is the entry point and delegates to the `cmd` package.
- `cmd/root.go` defines the Cobra root command and shared logging setup.
- `cmd/add.go` implements the `add` subcommand, SRH encoding wrapper, flags, and netlink route creation.
- `go.mod` and `go.sum` define the Go module and dependencies.
- `ip-sr-tlv` is a local build artifact; do not treat it as source.

There are currently no test, fixture, or asset directories. Add tests beside the package they exercise.

## Build, Test, and Development Commands

- `go test ./...` runs all package tests. It currently reports no test files.
- `go build ./...` compiles all packages.
- `go build -o ip-sr-tlv .` builds the CLI binary in the repository root.
- `go run . add --prefix 2001:db8::/64 --segs 2001:db8::1 --dev eth0 --tlv-type 252 --tlv-len 1` runs the CLI locally.
- `gofmt -w main.go cmd/*.go` formats edited Go files.

Route-changing commands require Linux, suitable interfaces or namespaces, and privileges such as root or `CAP_NET_ADMIN`.

## Coding Style & Naming Conventions

Use standard Go formatting with tabs produced by `gofmt`. Keep package names short and lowercase. Export identifiers only for package contracts; otherwise prefer names such as `addCmd` and `addArgs`.

Follow existing Cobra patterns: define commands in `cmd/*.go`, register subcommands in `init()`, and keep flag storage close to the implementation. Wrap errors with context using `fmt.Errorf("action: %w", err)`.

## Testing Guidelines

Use Go's built-in `testing` package unless a stronger need appears. Name tests `TestXxx` in files ending with `_test.go`.

Prefer unit tests for pure behavior such as SRH encoding validation. Netlink mutation should be isolated behind helpers or guarded by integration tests that document required privileges and namespace setup.

## Commit & Pull Request Guidelines

No Git history is available in this checkout, so follow conventional, imperative commit subjects such as `add SRH encode tests` or `validate TLV length`.

Pull requests should include a behavior summary, test results such as `go test ./...`, and any manual networking setup used to verify route changes. Link related issues when available and include command output for CLI behavior changes.

## Security & Configuration Tips

Be careful with examples that alter host routes. Prefer network namespaces or disposable test interfaces for manual verification. Do not commit generated binaries, local environment files, or machine-specific network configuration.
