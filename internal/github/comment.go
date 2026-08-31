package github

import (
	"context"
	"fmt"
	"strings"

	gh "github.com/google/go-github/v66/github"
	"github.com/mananchugh02/phantomcheck/internal/scorer"
)

func FormatReport(report scorer.PRReport, deletedFiles []string) string {
	var b strings.Builder
	b.WriteString("## 🔍 PhantomCheck Analysis\n\n")
	b.WriteString("| File | Score | Risk |\n|------|-------|------|\n")
	for _, file := range report.Files {
		fmt.Fprintf(&b, "| `%s` | %.2f | %s %s |\n", file.Filename, file.Score, riskEmoji(file.Risk), file.Risk)
	}
	for _, file := range report.Files {
		b.WriteString("\n### `")
		b.WriteString(file.Filename)
		b.WriteString("`\n")
		for _, finding := range file.Findings {
			if finding.IsPhantom {
				fmt.Fprintf(&b, "- ❌ `%s.%s` (Line %d) — does not exist\n", finding.Package, finding.Func, finding.Line)
			} else {
				fmt.Fprintf(&b, "- ✅ `%s.%s` (Line %d)\n", finding.Package, finding.Func, finding.Line)
			}
		}
		switch {
		case file.SandboxResult.CompileError != "":
			fmt.Fprintf(&b, "- ❌ Compile error: %s\n", file.SandboxResult.CompileError)
		case file.SandboxResult.Skipped:
			fmt.Fprintf(&b, "- ⏭️ Sandbox skipped: %s\n", file.SandboxResult.SkipReason)
		case file.SandboxResult.Panicked:
			fmt.Fprintf(&b, "- ❌ Runtime panic: %s\n", file.SandboxResult.PanicTrace)
		default:
			b.WriteString("- ✅ No runtime issues\n")
		}
	}
	fmt.Fprintf(&b, "\n---\n### Overall PR Score: %.2f — %s %s\n", report.OverallScore, riskEmoji(report.OverallRisk), report.OverallRisk)
	b.WriteString("| Metric | Count |\n|--------|-------|\n")
	fmt.Fprintf(&b, "| Files analyzed | %d |\n", len(report.Files))
	fmt.Fprintf(&b, "| Phantom APIs   | %d |\n", report.TotalPhantom)
	fmt.Fprintf(&b, "| Runtime panics | %d |\n", report.TotalPanics)
	b.WriteString("\n> 🤖 Analyzed by [PhantomCheck](https://github.com/mananchugh02/phantomcheck)\n")
	if len(deletedFiles) > 0 {
		b.WriteString("\n### 🗑️ Deleted files (not analyzed)\n")
		for _, path := range deletedFiles {
			fmt.Fprintf(&b, "- `%s`\n", path)
		}
	}
	return b.String()
}

func PostComment(ctx context.Context, client *Client, owner, repo string, prNumber int, body string) error {
	_, _, err := client.Issues.CreateComment(ctx, owner, repo, prNumber, &gh.IssueComment{Body: &body})
	return err
}

func riskEmoji(risk string) string {
	switch risk {
	case "HIGH":
		return "🔴"
	case "MEDIUM":
		return "🟡"
	default:
		return "🟢"
	}
}
