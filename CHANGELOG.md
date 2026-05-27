# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.1.1] — 2026-05-27

### Fixed
- **CI lint pipeline** — upgraded to `golangci-lint-action@v7` (required for golangci-lint v2)
  and pinned lint version to `v2.12.2` to match local tooling.
- **golangci-lint v2 config** — migrated `issues.exclude-rules` to `linters.exclusions.rules`
  per golangci-lint v2 schema; `golangci-lint config verify` now passes cleanly in CI.

---

## [1.1.0] — 2026-05-27

### Added
- **Model picker in setup wizard** — `git-explain setup` now shows a numbered list of models
  for every provider; Ollama lists installed models live from `/api/tags`, cloud providers
  show curated lists of popular models. Enter a number, press Enter to keep current, or type
  any custom model name.
- **`llm.ListModels(ctx, url)`** — exported helper to enumerate installed Ollama models.

### Fixed
- **Ollama auto-detection** — `Available()` now queries `/api/tags` and auto-selects the first
  non-embedding model when the configured model (`llama3.2`) is not installed. Ollama with
  zero models is treated as unavailable.
- **Explicit `--provider ollama`** — lazy `ensureModel()` call in `Explain()` and `Stream()` so
  model auto-detection applies even when `--provider ollama` bypasses `Available()`.
- **stdin buffering** — setup wizard now uses a single shared `bufio.Reader` over `os.Stdin`,
  preventing input corruption when multiple prompts are answered in sequence.

---

## [1.0.0] — 2026-05-27

### Added

#### Commands
- `blame <file>:<line>` — explain why a specific line exists
- `blame <file>:<start>-<end>` — explain a range of lines as a single block
- `log <sha>` — explain what a commit changed and why
- `log --range <range>` — narrative summary of multiple commits
- `diff` — explain current working-tree changes
- `diff <range>` — explain a commit range diff
- `pr <number>` — summarize a GitHub pull request (requires `gh` CLI)
- `patterns [--author=<name>] [--limit=<n>]` — detect team coding habits
- `setup` — interactive wizard to configure LLM providers
- `cache clear` — clear the local response cache
- `--json` flag on all commands for machine-readable output
- `--model <name>` flag on all commands to override the LLM model

#### Providers
- **Ollama** — local, offline, zero-config auto-detection (`llama3.2`)
- **OpenAI** — `gpt-4o-mini` default, `OPENAI_API_KEY` env var
- **Anthropic** — `claude-haiku-4-5` default, `ANTHROPIC_API_KEY` env var
- **Gemini** — `gemini-2.0-flash` default, `GEMINI_API_KEY` env var
- **Qwen** — `qwen-turbo` default via Alibaba DashScope, `QWEN_API_KEY` env var
- **Moonshot** — `moonshot-v1-8k` default via Kimi, `MOONSHOT_API_KEY` env var
- **Rule-based** — offline fallback, always works without any LLM

Provider fallback chain: Ollama → OpenAI → Anthropic → Gemini → Qwen → Moonshot → rule-based

#### Rule-based engine
- 10 change-type patterns: bugfix, feat, refactor, test, docs, deps, perf, ci, chore, security
- Keyword scoring with domain-scope bonus (auth, db, api, frontend, etc.)
- Risk level classification (low / medium / high)
- Breaking-change detection from commit message signals
- Per-change-type checklists and side-effect warnings

#### Configuration
- `~/.git-explain/config.toml` with per-provider sections
- Environment variable overrides for all API keys
- Interactive `setup` wizard writes config file

#### Infrastructure
- Shared HTTP client with 30 s timeout across all cloud providers
- Local response cache with atomic writes (`~/.git-explain/cache/`)
- 1 MB `bufio.Scanner` buffer for large SSE response lines
- `golangci-lint` CI with `gofmt`, `govet`, `misspell`, `fieldalignment` checks
- Cross-platform release binaries: `darwin-amd64`, `darwin-arm64`, `linux-amd64`, `linux-arm64`, `windows-amd64`
- Shell completions for Bash, Zsh, and Fish

#### Security
- Gemini API key passed via `x-goog-api-key` header (not query string)
- `readLine()` uses `bufio.Reader` to correctly handle spaces in API keys

### Tests
- 57 tests, all passing with `-race`
- Git package: blame, log, diff, show integration tests
- LLM package: rule-based provider with 12 commit-type test cases
- Cache package: 8 unit tests including concurrent-write safety
- Config package: 6 unit tests including env-var override

[1.0.0]: https://github.com/myothuko98/git-explain/releases/tag/v1.0.0
