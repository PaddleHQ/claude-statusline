# Contributing

Thank you for helping with claude-statusline. This page covers what you need to change the code and get the change merged. For what the tool does, start with the [README](README.md).

## Prerequisites

- Go, at the version in `go.mod`.
- [golangci-lint](https://golangci-lint.run/) v2, for the linters and formatters in `.golangci.yml` (the build uses v2.14.0).

## Repository layout

| Path | What it is |
|---|---|
| `main.go` | A few lines of wiring: it hands the standard streams to `internal/cli`. It has no test, so that its wiring does not count against coverage. |
| `internal/cli` | Reads the session from standard input and prints the line. It never reports an error, because a blank status line is better than a broken one. |
| `internal/statusline` | Builds the line from the session. |
| `internal/payload` | Decodes the session JSON from Claude Code and strips control characters from its strings. |
| `segments/` | One folder per plugin, each providing the segment it names and owning its tests and generated mocks. A plugin declares the interfaces it needs and imports nothing from `internal/`. For now there is one, `segments/model`. |

## Verifying a change

CI runs the following on every pull request, and you can run the same commands locally.

```sh
go mod verify
go mod tidy -diff             # go.mod and go.sum are tidy
! grep -q '^toolchain ' go.mod   # no toolchain directive
test -z "$(gofmt -s -l .)"    # formatting
go generate ./... && git diff --exit-code   # generated mocks are up to date
go vet ./...
go test -race ./...
# coverage at or above requiredCoverage in build-config.yaml: see Coverage below
golangci-lint run
actionlint                    # the workflow files
editorconfig-checker          # whitespace and indentation against .editorconfig
```

The workflow files in `.github/workflows/` are the source of truth if this list and CI ever disagree. `golangci-lint run --fix` applies the formatters for you.

Dependabot opens weekly pull requests for Go modules (labelled `release:patch`) and GitHub Actions (labelled `norelease`), so they pass the SemVer label check. A pull request carrying a blocking label, such as `do not merge`, fails the "Check for blocking labels" check until the label is removed.

## Tests

Tests use [testify](https://github.com/stretchr/testify): `require` when the rest of the test cannot continue, `assert` otherwise. Test files live in external `_test` packages, so they exercise the public surface the way a caller would. Where a test needs to reach something unexported, the package has an `export_test.go` that exposes it to tests only, so production code gains no public API for the sake of testing.

Prefer table-driven tests (`t.Run` over a slice of cases). A plugin's tests use [mockery](#mocks) mocks, so they never build a payload; the parts under `internal/` use real values and, for files, a temporary directory, never your `HOME`.

### Coverage

CI's Coverage job runs the tests with `-coverpkg ./...`, so a package is credited for statements that other packages' tests run, and fails when total statement coverage is below `requiredCoverage` in `build-config.yaml`. It leaves out generated mocks (`/mocks/`) and `tools/`, and writes a per-package table to the job summary. `main` is kept free of tests, because a test file there would pull its untested wiring into the total. When coverage is a point or more above the requirement, the job prints a notice with the value to raise it to; raising it is a ratchet, so do it in the pull request that earned it. To reproduce the job locally:

```sh
mkdir -p covdata
go test -race -cover -covermode=atomic -coverpkg ./... ./... -args -test.gocoverdir="$PWD/covdata"
go tool covdata textfmt -i=covdata -o cover.out
grep -vE '/mocks/|/tools/' cover.out > cover.filtered.out
go tool cover -func cover.filtered.out | tail -n 1
```

### Fuzzing

The status line runs on every render and reads input it does not control, starting with the session JSON. Each such parser has a native Go fuzz target (`FuzzParse` in `internal/payload`) that checks a property rather than only that nothing panics: payload strings hold no control characters.

`go test ./...` runs every target on its seed inputs, so they are ordinary tests in CI. To let the fuzzer search for new inputs, run one target at a time:

```sh
go test -run='^$' -fuzz='^FuzzParse$' -fuzztime=60s ./internal/payload
```

The "Weekly fuzz" workflow does the same for every target on a schedule, and can be started by hand from the Actions tab. When a run fails, download the `crashers-*` artefact: it holds the new input as `testdata/fuzz/<Target>/<id>`. Copy it into the same path in your checkout, run `go test -run='FuzzParse/<id>' ./internal/payload` to reproduce it, and fix the code (or the property, if the property was wrong). Commit the input file with the fix, so it stays as a regression test. A new fuzz target also needs a line in the matrix of `.github/workflows/fuzz.yml`.

### Mocks

Plugin tests use mocks generated by [mockery](https://vektra.github.io/mockery/) v2, which is a `tool` in `go.mod`, so nothing needs installing. Each interface a plugin declares has a `//go:generate go run github.com/vektra/mockery/v2 --name=<Interface>` line above it, and `.mockery.yaml` holds the options. The mocks are written to a `mocks/` folder in the plugin's package and are committed. After adding or changing a plugin interface, regenerate them from the repository root:

```sh
go generate ./...
```

CI runs the same command and fails if it changes anything, so a stale mock cannot be merged. Do not edit the files in `mocks/`. Tests use the typed expecter API, for example `m.EXPECT().DisplayName().Return("Opus 4.7")`.

### Running the binary

To run the binary by hand, point `HOME` and `CLAUDE_CONFIG_DIR` at a temporary directory, so that it never touches your real Claude Code state:

```sh
tmp=$(mktemp -d)
echo '{"model":{"display_name":"Opus 4.7"}}' | HOME="$tmp" CLAUDE_CONFIG_DIR="$tmp" go run .
```

## Conventions

### Go style

- Methods on a struct use pointer receivers, never value receivers, so that a type has one consistent method set.
- Every receiver is named, with a short name taken from the type (`func (m *Model) Render()`, never `func (*Model) Render()`).
- Behaviour that operates on a struct's data is a method on that struct, so it is found by looking at the type rather than by searching the package for a free function. Constructors stay functions (`New`, `Parse`: anything that builds and returns a value for the caller to use), and so do stateless helpers with no natural owning type, such as `payload.Sanitize`. Do not force a helper onto a type just to avoid a package-level function.

### Comments and language

Comments explain why, not what; code that reads clearly needs none. Code, identifiers and comments use US English. Documentation, commit messages and pull request text use British English.

## Pull requests

Every pull request needs exactly one of these labels, checked by the `Check SemVer label` workflow: `release:major`, `release:minor`, `release:patch` or `norelease`. Use `norelease` for changes that need no release, such as documentation or tests.
