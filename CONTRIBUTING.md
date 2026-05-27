# Contributing to git-explain

Thank you for your interest in contributing! This guide covers everything you need to get started.

---

## Table of contents

- [Dev setup](#dev-setup)
- [Commit format](#commit-format)
- [Branch naming](#branch-naming)
- [Pull request rules](#pull-request-rules)
- [Code style](#code-style)
- [Running tests](#running-tests)

---

## Dev setup

```bash
git clone https://github.com/myothuko98/git-explain
cd git-explain
go mod download

# Build
make build        # outputs bin/git-explain

# Run tests
make test

# Lint (requires golangci-lint)
make lint

# Install locally
make install      # copies to /usr/local/bin
```

Install [`golangci-lint`](https://golangci-lint.run/usage/install/) for linting:

```bash
brew install golangci-lint       # macOS
# or: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

Install [Ollama](https://ollama.com) + `ollama pull llama3.2` to test with a real LLM locally.

---

## Commit format

We follow [Conventional Commits](https://www.conventionalcommits.org/).

```
<type>(<scope>): <short summary>

[optional body — what and why, not how; wrap at 72 chars]

[optional footer — Breaking: ..., Closes #123]
```

### Types

| Type | When to use |
|---|---|
| `feat` | New feature or behaviour |
| `fix` | Bug fix |
| `docs` | Documentation only |
| `test` | Tests only, no production code change |
| `refactor` | Code restructure, no behaviour change |
| `chore` | Build system, dependencies, tooling |
| `ci` | GitHub Actions / CI configuration |
| `perf` | Performance improvement |

### Scopes

`blame` · `log` · `diff` · `pr` · `patterns` · `llm` · `cache` · `render` · `git` · `config` · `cmd`

### Examples

```
feat(blame): add range support for multi-line explanation
fix(llm): fall through to next provider on stream error
docs: update README with v2 features and shell completions
test(git): add mock runner for unit tests
refactor(render): extract StreamHeader/StreamFooter helpers
chore(deps): bump golangci-lint to v1.62
ci: add lint workflow on pull_request
```

### Rules

- Subject line ≤ 72 characters
- Use imperative mood: "add", "fix", "update" — not "added", "fixed"
- No period at end of subject line
- Reference issues in footer: `Closes #42`
- Breaking changes: add `!` after scope and describe in footer:
  ```
  feat(config)!: rename api_key to apiKey in TOML

  Breaking: config files using api_key must be updated.
  ```

---

## Branch naming

```
type/short-description
```

Examples:
```
feat/streaming-output
fix/blame-empty-file
docs/add-shell-completions
chore/upgrade-cobra
```

Keep descriptions lowercase, hyphen-separated, ≤ 5 words.

---

## Pull request rules

1. **One concern per PR** — a PR should do one thing. Mix of unrelated changes → split into separate PRs.
2. **Link to an issue** — every PR should reference an issue (`Closes #42`) or explain why one doesn't exist.
3. **Tests required** — new features and bug fixes must include tests. Coverage should not drop.
4. **Green CI** — all checks (tests, lint) must pass before merge.
5. **Squash before merge** — messy WIP commits should be squashed. Final commit must follow the commit format above.
6. **No force-push to `main`** — `main` is a protected branch.

### PR size guideline

| Lines changed | Expectation |
|---|---|
| < 200 | Quick review, same day |
| 200–500 | Normal review, 1-2 days |
| > 500 | Consider splitting |

---

## Code style

- **Format**: `gofmt` (enforced by CI). Run `gofmt -w .` before committing.
- **Lint**: `golangci-lint run` (see `.golangci.yml` for active rules).
- **Error handling**: always wrap errors with context — `fmt.Errorf("blame %s:%d: %w", file, line, err)`.
- **Comments**: exported symbols must have a doc comment. Inline comments only for non-obvious logic.
- **No `panic`** outside `main` — return errors up the call stack.

---

## Running tests

```bash
make test               # go test ./...
make test-verbose       # go test -v ./...

# Single package
go test ./internal/git/... -v

# With race detector
go test -race ./...
```

Tests in `internal/git` use a mock runner — no real git repo required.

---

## Questions?

Open a [GitHub Discussion](https://github.com/myothuko98/git-explain/discussions) or file an issue.
