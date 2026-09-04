# Contributing to PhantomCheck

Thanks for your interest in contributing to PhantomCheck. Contributions that improve detection quality, reliability, and developer experience are welcome.

## Development setup

### Prerequisites

- Go 1.21 or later
- A GitHub personal access token with `repo` scope
- A Groq API key, available for free at [console.groq.com](https://console.groq.com/)

Clone and build the project:

```bash
git clone https://github.com/mananchugh02/PhantomCheck.git
cd PhantomCheck
go build -o phantomcheck ./cmd/phantomcheck/
```

Set the environment variables required for live PR analysis:

```bash
export GITHUB_TOKEN=your_github_token
export GROQ_API_KEY=your_groq_api_key
```

In Windows PowerShell:

```powershell
$env:GITHUB_TOKEN = "your_github_token"
$env:GROQ_API_KEY = "your_groq_api_key"
```

## Running tests

Run the full test suite without cached results:

```bash
go test ./... -count=1
```

Run the built-in end-to-end sample:

```bash
go run ./cmd/phantomcheck/ analyze --test
```

Tests do not require `GITHUB_TOKEN` or `GROQ_API_KEY`. Those credentials are only needed for live `--pr` mode.

## Project structure

- `cmd/phantomcheck/` — pipeline orchestration and CLI flags
- `internal/analyzer/` — static analysis layer using `go/ast` and `go/types`
- `internal/sandbox/` — sandbox runner using `os/exec`
- `internal/semantic/` — LLM semantic analysis through Groq
- `internal/llm/` — Groq API client
- `internal/scorer/` — scoring engine
- `internal/report/` — JSON report builder
- `internal/github/` — GitHub API integration

## Making changes

- Keep each PR focused on one thing.
- Add or update tests for every behavior change.
- Run `go test ./...` before submitting.
- Run `go vet ./...` and ensure there are no new warnings.
- Keep the `--test` flag working so the tool can be tested without GitHub credentials.

## Submitting a PR

Fork the repository, create a branch, and open a PR against `main`.

PhantomCheck automatically analyzes its own PRs, so yes, it reviews itself. In the PR description, explain what the change does and why; the semantic analysis layer uses that context when reviewing the code.

## Reporting issues

Open a [GitHub issue](https://github.com/mananchugh02/PhantomCheck/issues) with clear steps to reproduce the problem. For analysis bugs, include the output of:

```bash
phantomcheck analyze --verbose --pr <url>
```
