# Contributing to Fossor

Thanks for your interest in improving Fossor. This guide covers what you need to develop, test, and submit changes.

## Prerequisites

| Tool   | Version | Why                                                |
| ------ | ------- | -------------------------------------------------- |
| Go     | 1.25+   | Matches `go.mod`.                                  |
| Git    | 2.20+   | Fossor shells out to `git` for every operation.    |
| Make   | any     | Driver for all common tasks.                       |
| Bash   | any     | `testdata/setup.sh` and `testdata/reset.sh`.       |

## Local Development Workflow

```bash
make build          # Build the fossor binary into the working dir
make run            # Build and launch against the current directory
make test           # Run the full test suite
make vet            # go vet ./...
make check          # vet + test + build
make deps           # go mod tidy
make update         # Bump all module deps to latest + tidy
make testdata       # Create 20 mock repos in testdata/repos with varied states
make testdata-reset # Wipe and recreate the mock repos (deterministic)
make clean          # Remove the binary and testdata/repos
```

For local UI work the typical loop is:

```bash
make testdata           # one-time setup
make build && ./fossor testdata/repos --no-fetch
```

`--no-fetch` skips the network round-trip during discovery so the TUI lands instantly.

## Commit Style

We use [Conventional Commits](https://www.conventionalcommits.org/). Look at `git log` for examples. Common types:

- `feat:` — new functionality
- `fix:` — bug fix
- `chore:` — maintenance, dependency bumps, repo hygiene
- `docs:` — documentation only
- `ci:` — workflow or release-tooling changes
- `refactor:` — internal cleanup with no behavior change
- `perf:` — performance improvement
- `style:` — formatting / gofmt only
- `test:` — adding or refactoring tests

Scope where it adds clarity: `feat(ui): …`, `fix(git): …`, `feat(cli): …`.

Keep commits small and focused — one logical change per commit. Branch from `main`.

## Pull Request Checklist

- [ ] `make check` is clean (vet + tests + build).
- [ ] `golangci-lint run ./...` is clean if you have it installed (CI runs it on every PR).
- [ ] `go test -race ./...` is clean if you touched concurrency-sensitive code (CI runs `-race` by default).
- [ ] User-facing changes have a `CHANGELOG.md` entry under `## [Unreleased]`.
- [ ] Behavior changes that affect documented keys/flags/states touch the matching reference page in the same PR (see `docs/style.md`).
- [ ] New keyboard shortcuts are reflected in `docs/reference/keybindings.md` and `README.md`.

## Documentation Placement

Fossor docs follow the [Diátaxis](https://diataxis.fr/) framework:

| You're writing…                                | Goes in              |
| ---------------------------------------------- | -------------------- |
| Step-by-step intro for new users               | `docs/tutorials/`    |
| Goal-oriented "how do I…" recipe               | `docs/how-to/`       |
| Lookup-style key/flag/state listing            | `docs/reference/`    |
| Background, rationale, design trade-offs       | `docs/explanation/`  |

See `docs/style.md` for tone, headings, and formatting conventions, and `docs/index.md` for the documentation map.

## Reporting Security Issues

Please don't open a public issue for security vulnerabilities. See `SECURITY.md` for the disclosure process.

## Code of Conduct

Be respectful. Disagreement is fine; personal attacks are not.

## License

By contributing you agree your contributions are licensed under the project's [MIT License](LICENSE).
