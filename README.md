# git-explain

> **Understand *why* code changed, not just *when*.**

`git-explain` is a CLI tool that turns raw `git blame` / `git log` output into clear human-readable explanations using AI — privately, with zero mandatory configuration.

```
git-explain blame src/auth.go:42
git-explain log a3f9b2c
git-explain pr 88
git-explain patterns --author=alice
```

---

## Features

| Command | What it does |
|---|---|
| `blame <file>:<line>` | Explains why a specific line of code exists and what commit introduced it |
| `log <sha>` | Explains what a commit changed and why it was needed |
| `pr <number>` | Summarises a GitHub pull request (requires `gh` CLI) |
| `patterns` | Detects team coding habits: fix ratios, refactor cycles, top keywords |
| `setup` | Interactive wizard to configure LLM providers |

---

## Zero-config, privacy-first

No API key required to get started. `git-explain` uses a **provider fallback chain**:

```
Ollama (local, free) → OpenAI → Anthropic → Gemini → rule-based (always works)
```

- **Ollama** (preferred) — 100% offline, zero cost, data never leaves your machine
- Cloud providers — opt-in only, configured via `git-explain setup` or env vars
- **Rule-based fallback** — always produces a useful output, even with no LLM

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
provider = "auto"   # auto | ollama | openai | anthropic | gemini | rule-based

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
```

---

## Examples

### Explain a line of code

```
$ git-explain blame src/auth/middleware.go:42

🔍 Blame Explanation
src/auth/middleware.go:42  ·  commit a3f9b2c  ·  alice@example.com
╭──────────────────────────────────────────────────────────────────╮
│ This line was introduced in commit a3f9b2c to fix a JWT token    │
│ expiry race condition discovered in production. The `<=`          │
│ comparison was changed to `<` because tokens expiring at exactly  │
│ the current Unix timestamp were incorrectly accepted. PR #88      │
│ contains the full discussion.                                     │
╰──────────────────────────────────────────────────────────────────╯
via ollama
```

### Explain a commit

```
$ git-explain log HEAD~3

🔍 refactor: extract DB connection pool
commit b12cd34  ·  bob  ·  2024-11-01
╭──────────────────────────────────────────────────────────────────╮
│ This commit extracted the inline database connection logic into   │
│ a dedicated pool manager. It was motivated by timeouts seen       │
│ under load — the old approach created a new connection per        │
│ request. The pool now reuses up to 20 connections and closes      │
│ idle ones after 30 seconds.                                       │
╰──────────────────────────────────────────────────────────────────╯
via ollama
```

### Detect team patterns

```
$ git-explain patterns

▸ Team Patterns  (500 commits analysed)

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

## Requirements

- Git ≥ 2.x
- Go ≥ 1.21 (only for building from source)
- `gh` CLI — only needed for `git-explain pr`
- Ollama / API key — optional (rule-based fallback always works)

---

## License

MIT
