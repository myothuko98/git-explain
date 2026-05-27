# git-explain

> **Understand *why* code changed, not just *when*.**

[![CI](https://github.com/myothuko98/git-explain/actions/workflows/lint.yml/badge.svg)](https://github.com/myothuko98/git-explain/actions/workflows/lint.yml)
[![Release](https://img.shields.io/github/v/release/myothuko98/git-explain)](https://github.com/myothuko98/git-explain/releases/latest)
[![Go version](https://img.shields.io/github/go-mod/go-version/myothuko98/git-explain)](go.mod)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/myothuko98/git-explain)](https://goreportcard.com/report/github.com/myothuko98/git-explain)

`git-explain` is a CLI tool that turns raw `git blame` / `git log` output into clear human-readable explanations using AI — privately, with zero mandatory configuration.

```
git-explain blame src/auth.go:42
git-explain blame src/auth.go:40-55
git-explain log a3f9b2c
git-explain log --range HEAD~5..HEAD
git-explain diff
git-explain diff HEAD~3..HEAD
git-explain pr 88
git-explain patterns --author=alice
```

---

## Features

| Command | What it does |
|---|---|
| `blame <file>:<line>` | Explains why a specific line exists |
| `blame <file>:<start>-<end>` | Explains a range of lines as a block |
| `log <sha>` | Explains what a commit changed and why |
| `log --range <range>` | Narrative summary of multiple commits |
| `diff` | Explains current working-tree changes |
| `diff <range>` | Explains a commit range diff |
| `pr <number>` | Summarizes a GitHub pull request (requires `gh`) |
| `patterns` | Detects team coding habits: fix ratios, refactor cycles |
| `setup` | Interactive wizard to configure LLM providers |
| `cache clear` | Clears the local response cache |

All commands accept:
- `--json` — machine-readable JSON output (for editor plugins, scripts)
- `--model <name>` — override the LLM model for that request

---

## Zero-config, privacy-first

No API key required to get started. `git-explain` uses a **provider fallback chain**:

```
Ollama (local) → OpenAI → Anthropic → Gemini → Qwen → Moonshot → rule-based
```

- **Ollama** (preferred) — 100% offline, zero cost, data never leaves your machine
- Cloud providers — opt-in only, configured via `git-explain setup` or env vars
- **Rule-based fallback** — always produces a useful output, even with no LLM configured

Responses are cached locally (`~/.git-explain/cache/`) so repeated queries are instant.

---

## Installation

### Homebrew (macOS / Linux)

```bash
brew install myothuko98/tap/git-explain
```

### Direct download

```bash
curl -fsSL https://github.com/myothuko98/git-explain/releases/latest/download/git-explain-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') -o /usr/local/bin/git-explain
chmod +x /usr/local/bin/git-explain
```

### From source

```bash
git clone https://github.com/myothuko98/git-explain
cd git-explain
make install   # builds and copies to /usr/local/bin
```

---

## Configuration

Run the interactive wizard:

```bash
git-explain setup
```

Or edit `~/.git-explain/config.toml` manually:

```toml
provider = "auto"   # auto | ollama | openai | anthropic | gemini | qwen | moonshot | rule-based

[ollama]
url   = "http://localhost:11434"
model = "llama3.2"

[openai]
api_key = ""        # or set OPENAI_API_KEY env var
model   = "gpt-4o-mini"

[anthropic]
api_key = ""        # or set ANTHROPIC_API_KEY env var
model   = "claude-haiku-4-5"

[gemini]
api_key = ""        # or set GEMINI_API_KEY env var
model   = "gemini-2.0-flash"

[qwen]
api_key = ""        # or set QWEN_API_KEY env var
model   = "qwen-turbo"

[moonshot]
api_key = ""        # or set MOONSHOT_API_KEY env var
model   = "moonshot-v1-8k"
```

### Environment variables

| Variable | Provider |
|---|---|
| `OPENAI_API_KEY` | OpenAI |
| `ANTHROPIC_API_KEY` | Anthropic |
| `GEMINI_API_KEY` | Gemini |
| `QWEN_API_KEY` | Qwen (Alibaba DashScope) |
| `MOONSHOT_API_KEY` | Moonshot AI (Kimi) |

---

## Providers

| Provider | Type | Model default | API key source |
|---|---|---|---|
| **Ollama** | Local (offline) | `llama3.2` | None required |
| **OpenAI** | Cloud | `gpt-4o-mini` | `OPENAI_API_KEY` |
| **Anthropic** | Cloud | `claude-haiku-4-5` | `ANTHROPIC_API_KEY` |
| **Gemini** | Cloud | `gemini-2.0-flash` | `GEMINI_API_KEY` |
| **Qwen** | Cloud | `qwen-turbo` | `QWEN_API_KEY` |
| **Moonshot** | Cloud | `moonshot-v1-8k` | `MOONSHOT_API_KEY` |
| **Rule-based** | Local (offline) | — | None required |

All cloud providers support streaming responses. Qwen uses [Alibaba DashScope](https://dashscope.aliyuncs.com/)'s OpenAI-compatible endpoint. Moonshot uses [Moonshot AI (Kimi)](https://platform.moonshot.cn/)'s OpenAI-compatible endpoint.

---

## Examples

### Explain a line of code

```
$ git-explain blame src/auth/middleware.go:42

🔍 Blame: src/auth/middleware.go:42
commit a3f9b2c  ·  alice

This line was introduced to fix a JWT token expiry race condition.
The <= comparison was changed to < because tokens expiring at exactly
the current Unix timestamp were incorrectly accepted. See PR #88.

via ollama
```

### Explain a block of lines

```
$ git-explain blame src/auth/middleware.go:40-55

🔍 Blame: src/auth/middleware.go:40-55
16 lines  ·  2 unique commits

This block implements rate limiting for the token refresh endpoint.
Lines 40-48 add per-IP counters using a sync.Map, while lines 49-55
apply a sliding window check to reject requests exceeding 10 req/min.
It was introduced after a credential-stuffing incident in October.

via ollama
```

### Rule-based output (no LLM)

```
$ git-explain log HEAD

🔍 fix(auth): prevent JWT expiry bypass
commit a3f9b2c  ·  alice  ·  Mon Nov 4 14:02:11 2024

[rule-based analysis — no LLM configured]
──────────────────────────────────────────────────────────────
  Change type:  Bug Fix
  Risk level:   🟡 Medium
  Scope:        Auth · Security
  Breaking:     No

  What this change likely does:
  Corrects incorrect or unexpected behavior in the codebase.

  Common causes:
  • Off-by-one or boundary condition error
  • Concurrent access without proper synchronization

  Review checklist:
  ✓ Regression test that reproduces the original bug?
  ✓ All error paths handled, not just the happy path?

  Tip: run `git-explain setup` to configure an LLM
       for richer, context-aware explanations.
──────────────────────────────────────────────────────────────
```

### Explain current changes

```
$ git-explain diff

🔍 Diff explanation
working-tree changes

The changes add streaming support to the LLM provider layer.
A new Streamer interface allows providers to write tokens incrementally
to an io.Writer, reducing time-to-first-token from ~4s to ~200ms.

via ollama
```

### Explain a commit range

```
$ git-explain log --range HEAD~5..HEAD

🔍 Log range: HEAD~5..HEAD (5 commits)

Over these five commits the team migrated the auth layer from
session-based to JWT tokens. The first commit added the signing
key config, followed by token generation, then middleware validation,
a refresh endpoint, and finally cleanup of the old session store.

via ollama
```

### JSON output (for scripts / editor plugins)

```bash
git-explain blame src/auth.go:42 --json
```
```json
{
  "title": "Blame: src/auth.go:42",
  "meta": "commit a3f9b2c  ·  alice",
  "provider": "ollama",
  "explanation": "This line was introduced to fix..."
}
```

### Model override

```bash
git-explain log HEAD --model gpt-4o
git-explain blame src/db.go:100 --model llama3.1:70b
git-explain diff --model qwen-plus
```

### Detect team patterns

```
$ git-explain patterns

=== Team Patterns  (500 commits analyzed) ===
alice (210 commits)
  Fix ratio:      42%
  Refactor ratio: 18%
  Top keywords:   auth, token, session, fix, middleware

bob (180 commits)
  Fix ratio:      8%
  Refactor ratio: 61%
  Top keywords:   refactor, db, pool, clean, rename
```

---

## Local LLM with Ollama

1. Install [Ollama](https://ollama.com)
2. Pull a model: `ollama pull llama3.2`
3. Run `git-explain` — it auto-detects Ollama, no config needed

---

## Shell completions

```bash
# Zsh
git-explain completion zsh >> ~/.zshrc

# Bash
git-explain completion bash >> ~/.bashrc

# Fish
git-explain completion fish > ~/.config/fish/completions/git-explain.fish
```

---

## Requirements

- Git ≥ 2.x
- Go ≥ 1.24 (only for building from source)
- `gh` CLI — only needed for `git-explain pr`
- Ollama / API key — optional (rule-based fallback always works)

---

## License

MIT

