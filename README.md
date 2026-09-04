# PhantomCheck

**Detect hallucinated Go code in GitHub Pull Requests.**

[![Go 1.21+](https://img.shields.io/badge/Go-1.21%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-2ea44f.svg)](LICENSE)
[![Build](https://github.com/mananchugh02/phantomcheck/actions/workflows/tests.yml/badge.svg)](https://github.com/mananchugh02/phantomcheck/actions/workflows/tests.yml)
[![PRs welcome](https://img.shields.io/badge/PRs-welcome-ff69b4)](https://github.com/mananchugh02/phantomcheck/pulls)

## What is PhantomCheck?

PhantomCheck is a Go CLI that reviews GitHub Pull Requests for hallucinated or logically incorrect Go code. It checks every changed `.go` file with static analysis, an isolated runtime sandbox, and optional Groq-powered semantic review. The result is scored per file, summarized for the PR, and posted as a deduplicated GitHub comment.

## How it works

```text
PR URL
  |
  v
GitHub Fetcher: changed .go files, patches, PR context
  |
  v
Static Analysis: go/parser + go/ast + go/types
  |                 phantom APIs and phantom methods
  v
Sandbox Runner: os/exec + context
  |              compile errors, panics, 5s timeouts
  v
Semantic LLM: Groq inference against diff and PR intent
  |            logical bugs and missing edge cases
  v
Scorer: per-file scores and overall PR average
  |
  v
Terminal report + GitHub PR comment
```

Analysis runs concurrently for changed files, with a maximum of five active file analyses. Non-standalone package files are skipped by the sandbox instead of generating false compile failures.

## Detection examples

### Static analysis

The type checker catches package functions and methods that do not exist:

```go
os.ReadFileLines("config.json")       // phantom package function
strings.ToUpperCase("pending")        // phantom package function
sb.WriteStrnig("hello")                // phantom method on strings.Builder
user.GetEmail()                        // phantom method on a known User type
```

Real calls are reported as valid, including `os.ReadFile`, `strings.Contains`, `fmt.Println`, `strings.Builder.WriteString`, and methods defined on local structs.

### Sandbox runner

Standalone programs are built and run in a temporary directory:

```go
func main() {
	values := []int{}
	fmt.Println(values[1]) // compile succeeds; runtime panic is captured
}
```

Files such as `package auth` without `func main()` are marked as skipped because they depend on sibling package files and are not standalone programs.

### Semantic analysis

The optional Groq layer compares the patch with the PR title, description, and commit messages:

```text
PR intent: validate the user input before indexing
Actual diff: indexes the slice before checking its length
Result: HIGH semantic issue, with the affected line range
```

It reports missing error handling, inverted conditions, off-by-one errors, and other logical mismatches that compile and may even run successfully.

## Sample PR comment

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
- 🔴 Accessing results[0] on an empty slice (lines 22-23)
- 🟡 Error from `os.ReadFile` is not handled (lines 11-14)
- ❌ Compile error: undefined: os.ReadFileLines

---
### Overall PR Score: 1.00 — 🔴 HIGH
| Metric | Count |
|--------|-------|
| Files analyzed | 1 |
| Phantom APIs | 3 |
| Runtime panics | 0 |
| Semantic issues | 2 |

> 🤖 Analyzed by [PhantomCheck](https://github.com/mananchugh02/phantomcheck)
```

## Getting started

### Install

```bash
git clone https://github.com/mananchugh02/phantomcheck.git
cd phantomcheck
go build -o phantomcheck ./cmd/phantomcheck/
```

Or install directly:

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

### Built-in test data

```bash
phantomcheck analyze --test
```

### Machine-readable output and CI gating

```bash
phantomcheck analyze --pr https://github.com/owner/repo/pull/123 --output json
phantomcheck analyze --pr https://github.com/owner/repo/pull/123 --fail-on HIGH
phantomcheck analyze --test --output json --fail-on MEDIUM
```

`--output` accepts `text` by default or `json`. JSON is printed to stdout; operational messages go to stderr. JSON reports include `exit_code` so consumers can inspect the intended CI result. `--fail-on` accepts `LOW`, `MEDIUM`, or `HIGH`, case-insensitively, and exits with code `1` when the overall risk meets or exceeds the threshold.

## Environment variables

| Variable | Required | Purpose |
|----------|----------|---------|
| `GITHUB_TOKEN` | PR mode | GitHub API access: read repository contents and write PR comments |
| `GROQ_API_KEY` | Optional | Enables semantic analysis through Groq's OpenAI-compatible API |

## GitHub Actions

### This repository

The repository uses two workflows:

- `tests.yml` runs `go test ./...` on pushes and pull requests.
- `phantomcheck.yml` runs PhantomCheck on pull requests with the automatic `GITHUB_TOKEN` and the `GROQ_API_KEY` repository secret.

### Add PhantomCheck to your repository

Create `.github/workflows/phantomcheck.yml`:

```yaml
name: PhantomCheck

on:
  pull_request:

permissions:
  contents: read
  pull-requests: write
  models: read

jobs:
  analyze:
    runs-on: ubuntu-latest
    steps:
      - name: Run PhantomCheck
        uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Install PhantomCheck
        run: go install github.com/mananchugh02/phantomcheck/cmd/phantomcheck@latest

      - name: Analyze pull request
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          GROQ_API_KEY: ${{ secrets.GROQ_API_KEY }}
        run: phantomcheck analyze --pr "https://github.com/${{ github.repository }}/pull/${{ github.event.pull_request.number }}" --fail-on HIGH
```

The workflow needs `contents: read` to fetch changed files and `pull-requests: write` to update the PR comment. Add `GROQ_API_KEY` under repository **Settings → Secrets and variables → Actions** to enable semantic analysis; the other two layers work without it.

## Project structure

```text
cmd/phantomcheck/   CLI entry point, flags, and pipeline orchestration
internal/analyzer/  Layer 1: Go AST/type analysis for phantom APIs and methods
internal/sandbox/   Layer 2: isolated compile/run checks with panic and timeout capture
internal/semantic/  Layer 3: Groq semantic review of diffs against PR intent
internal/llm/       OpenAI-compatible Groq chat-completions client
internal/scorer/    Per-file scoring and overall PR aggregation
internal/report/    Machine-readable JSON report mapping
internal/github/    PR files, patches, context, and deduplicated comments
```

## Running tests

```bash
go test ./...
```

Run the local CLI sample without GitHub credentials:

```bash
go run ./cmd/phantomcheck analyze --test
```

## Roadmap

- [x] GitHub Actions support
- [x] JSON output flag (`--output json`)
- [x] Parallel file analysis (goroutines)
- [x] Comment deduplication
- [x] Semantic LLM analysis layer
- [x] `--fail-on` CI gating flag
- [ ] Python file support via Tree-sitter
- [ ] Web dashboard
- [ ] Config file (`.phantomcheck.yml`)
- [ ] Fine-tuned model for Go hallucination detection

## Contributing

PRs are welcome. Keep changes focused, add tests for behavior changes, and run `go test ./...` before submitting.

## License

MIT
