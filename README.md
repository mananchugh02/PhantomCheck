# PhantomCheck

**Detect hallucinated Go code in GitHub PRs before it lands.**

[![Go 1.21+](https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-2ea44f.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen)](https://github.com/mananchugh02/phantomcheck/actions)
[![PRs welcome](https://img.shields.io/badge/PRs-welcome-ff69b4)](https://github.com/mananchugh02/phantomcheck/pulls)

## ✨ What is PhantomCheck?

PhantomCheck is a Go CLI that reviews GitHub Pull Requests for LLM-generated hallucinations in Go code. It catches phantom API calls, runtime panics, and compile errors, then turns those signals into a scored report. The result is posted both in your terminal and back onto the PR as a comment.

## 🧭 How it works

```text
PR URL
  ↓
Fetch changed .go files from GitHub
  ↓
Static analysis (go/parser + go/ast + go/types)
  └─ detects phantom API calls
  ↓
Sandbox runner (os/exec + context)
  └─ compiles, runs, catches panics/timeouts
  ↓
Scorer
  └─ converts signals into 0.0-1.0 risk per file
  ↓
Terminal report + PR comment
```

## 📝 Sample PR comment

```md
## 🔍 PhantomCheck Analysis

| File | Score | Risk |
|------|-------|------|
| `internal/test/llmcode.go` | 1.00 | 🔴 HIGH |

### `internal/test/llmcode.go`
- ✅ `os.ReadFile` (Line 11)
- ✅ `fmt.Println` (Line 13)
- ❌ `os.ReadFileLines` (Line 17) — does not exist
- ❌ `strings.ToUpperCase` (Line 18) — does not exist
- ❌ `fmt.PrintRed` (Line 19) — does not exist
- ❌ Compile error: undefined: os.ReadFileLines

---
### Overall PR Score: 1.00 — 🔴 HIGH
| Metric | Count |
|--------|-------|
| Files analyzed | 1 |
| Phantom APIs   | 3 |
| Runtime panics | 0 |

> 🤖 Analyzed by [PhantomCheck](https://github.com/mananchugh02/phantomcheck)
```

## 🚀 Getting started

### Install

```bash
git clone https://github.com/mananchugh02/phantomcheck
cd phantomcheck
go build -o phantomcheck ./cmd/phantomcheck/
```

### Or install directly

```bash
go install github.com/mananchugh02/phantomcheck/cmd/phantomcheck@latest
```

### Analyze a PR

```bash
export GITHUB_TOKEN=your_token_here
phantomcheck analyze --pr https://github.com/owner/repo/pull/123
```

### Windows PowerShell

```powershell
$env:GITHUB_TOKEN="your_token_here"
phantomcheck analyze --pr https://github.com/owner/repo/pull/123
```

### Run built-in test mode

```bash
phantomcheck analyze --test
```

## 🔐 GitHub Token setup

PhantomCheck uses the GitHub API to fetch PR files and post comments. Your token needs these permissions:

- `repo` read access to fetch changed PR files and file contents
- `repo` write access to post PR comments

Create a token in GitHub under **Settings → Developer settings → Personal access tokens**. Use a fine-grained or classic token that can read the target repository and write issues/comments. Store it in `GITHUB_TOKEN` before running PhantomCheck.

## 🧱 Project structure

```text
cmd/phantomcheck/   CLI entry point and PR command wiring
internal/analyzer/   Static analysis using Go AST and type information
internal/sandbox/    Temporary build-and-run sandbox for runtime checks
internal/scorer/     File scoring and PR-level aggregation logic
internal/github/     GitHub API client, PR file fetchers, and comment posting
internal/report/     Report formatting helpers and presentation utilities
```

## ✅ Running tests

```bash
go test ./internal/analyzer
go test ./internal/sandbox
go test ./internal/scorer
go test ./internal/github
```

Run the full suite when the tree is healthy:

```bash
go test ./...
```

## 🛣 Roadmap

- [ ] GitHub Actions support
- [ ] JSON output flag
- [ ] Python file support via Tree-sitter
- [ ] PR comment deduplication
- [ ] Web dashboard

## 🤝 Contributing

Issues and PRs are welcome. Keep changes focused, include tests where practical, and preserve the CLI output format unless the change is intentional.

## 📄 License

MIT
