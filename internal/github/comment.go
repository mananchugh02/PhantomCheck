package github

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	gh "github.com/google/go-github/v66/github"
)

const phantomCheckSignature = "<!-- phantomcheck-report -->"

func FormatReport(report interface{}, deletedFiles []string) string {
	var b strings.Builder
	reportValue := reflect.ValueOf(report)
	if reportValue.Kind() == reflect.Ptr {
		reportValue = reportValue.Elem()
	}
	b.WriteString(phantomCheckSignature + "\n")
	b.WriteString("## 🔍 PhantomCheck Analysis\n\n")
	b.WriteString("| File | Score | Risk |\n|------|-------|------|\n")
	files := reportValue.FieldByName("Files")
	for i := 0; i < files.Len(); i++ {
		file := files.Index(i)
		fmt.Fprintf(&b, "| `%s` | %.2f | %s %s |\n", fieldString(file, "Filename"), fieldFloat(file, "Score"), riskEmoji(fieldString(file, "Risk")), fieldString(file, "Risk"))
	}
	for i := 0; i < files.Len(); i++ {
		file := files.Index(i)
		b.WriteString("\n### `")
		b.WriteString(fieldString(file, "Filename"))
		b.WriteString("`\n")
		findings := file.FieldByName("Findings")
		for j := 0; j < findings.Len(); j++ {
			finding := findings.Index(j)
			if fieldBool(finding, "IsPhantom") {
				fmt.Fprintf(&b, "- ❌ `%s.%s` (Line %d) — does not exist\n", fieldString(finding, "Package"), fieldString(finding, "Func"), fieldInt(finding, "Line"))
			} else {
				fmt.Fprintf(&b, "- ✅ `%s.%s` (Line %d)\n", fieldString(finding, "Package"), fieldString(finding, "Func"), fieldInt(finding, "Line"))
			}
		}
		semanticFindings := file.FieldByName("SemanticFindings")
		if semanticFindings.Len() > 0 {
			b.WriteString("\n**Semantic findings**\n")
			for j := 0; j < semanticFindings.Len(); j++ {
				finding := semanticFindings.Index(j)
				fmt.Fprintf(&b, "- %s %s (lines %d-%d)\n", riskEmoji(fieldString(finding, "Severity")), fieldString(finding, "Description"), fieldInt(finding, "StartLine"), fieldInt(finding, "EndLine"))
			}
		}
		if fieldBool(file, "SemanticSkipped") {
			fmt.Fprintf(&b, "- ⏭️ Semantic analysis skipped: %s\n", fieldString(file, "SemanticSkipReason"))
		} else if semanticFindings.Len() == 0 {
			b.WriteString("- ✅ No logical issues found\n")
		}
		sandbox := file.FieldByName("SandboxResult")
		switch {
		case fieldString(sandbox, "CompileError") != "":
			fmt.Fprintf(&b, "- ❌ Compile error: %s\n", fieldString(sandbox, "CompileError"))
		case fieldBool(sandbox, "Skipped"):
			fmt.Fprintf(&b, "- ⏭️ Sandbox skipped: %s\n", fieldString(sandbox, "SkipReason"))
		case fieldBool(sandbox, "Panicked"):
			fmt.Fprintf(&b, "- ❌ Runtime panic: %s\n", fieldString(sandbox, "PanicTrace"))
		default:
			b.WriteString("- ✅ No runtime issues\n")
		}
	}
	fmt.Fprintf(&b, "\n---\n### Overall PR Score: %.2f — %s %s\n", fieldFloat(reportValue, "OverallScore"), riskEmoji(fieldString(reportValue, "OverallRisk")), fieldString(reportValue, "OverallRisk"))
	b.WriteString("| Metric | Count |\n|--------|-------|\n")
	fmt.Fprintf(&b, "| Files analyzed | %d |\n", files.Len())
	fmt.Fprintf(&b, "| Phantom APIs   | %d |\n", fieldInt(reportValue, "TotalPhantom"))
	fmt.Fprintf(&b, "| Runtime panics | %d |\n", fieldInt(reportValue, "TotalPanics"))
	fmt.Fprintf(&b, "| Semantic issues | %d |\n", fieldInt(reportValue, "TotalSemanticIssues"))
	b.WriteString("\n> 🤖 Analyzed by [PhantomCheck](https://github.com/mananchugh02/phantomcheck)\n")
	if len(deletedFiles) > 0 {
		b.WriteString("\n### 🗑️ Deleted files (not analyzed)\n")
		for _, path := range deletedFiles {
			fmt.Fprintf(&b, "- `%s`\n", path)
		}
	}
	return b.String()
}

func fieldString(value reflect.Value, name string) string { return value.FieldByName(name).String() }
func fieldFloat(value reflect.Value, name string) float64 { return value.FieldByName(name).Float() }
func fieldInt(value reflect.Value, name string) int       { return int(value.FieldByName(name).Int()) }
func fieldBool(value reflect.Value, name string) bool     { return value.FieldByName(name).Bool() }

func PostComment(ctx context.Context, client *Client, owner, repo string, prNumber int, body string) error {
	_, _, err := client.Issues.CreateComment(ctx, owner, repo, prNumber, &gh.IssueComment{Body: &body})
	return err
}

func FindExistingComment(ctx context.Context, client *Client, owner, repo string, prNumber int) (commentID int64, found bool, err error) {
	opts := &gh.IssueListCommentsOptions{ListOptions: gh.ListOptions{PerPage: 100}}
	for {
		comments, resp, err := client.Issues.ListComments(ctx, owner, repo, prNumber, opts)
		if err != nil {
			return 0, false, err
		}
		for _, comment := range comments {
			if strings.Contains(comment.GetBody(), phantomCheckSignature) {
				return comment.GetID(), true, nil
			}
		}
		if resp == nil || resp.NextPage == 0 {
			return 0, false, nil
		}
		opts.Page = resp.NextPage
	}
}

func UpsertComment(ctx context.Context, client *Client, owner, repo string, prNumber int, body string) error {
	commentID, found, err := FindExistingComment(ctx, client, owner, repo, prNumber)
	if err != nil {
		return err
	}
	if found {
		_, _, err = client.Issues.EditComment(ctx, owner, repo, commentID, &gh.IssueComment{Body: &body})
		return err
	}
	return PostComment(ctx, client, owner, repo, prNumber, body)
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
