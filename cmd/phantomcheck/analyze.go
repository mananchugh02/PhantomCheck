package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/mananchugh02/phantomcheck/internal/analyzer"
	ghrepo "github.com/mananchugh02/phantomcheck/internal/github"
	"github.com/mananchugh02/phantomcheck/internal/llm"
	"github.com/mananchugh02/phantomcheck/internal/sandbox"
	"github.com/mananchugh02/phantomcheck/internal/scorer"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{Use: "analyze", Run: func(_ *cobra.Command, _ []string) {}}
	cmd.Flags().String("pr", "", "GitHub pull request URL")
	cmd.Flags().Bool("test", false, "Run the static analyzer against bundled sample code")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		testMode, _ := cmd.Flags().GetBool("test")
		if testMode {
			report, err := buildTestReport()
			if err != nil {
				return err
			}
			printReport(report, nil)
			printLLMClientTest()
			return nil
		}
		prURL, _ := cmd.Flags().GetString("pr")
		if prURL == "" {
			return fail("missing required --pr flag\nusage: phantomcheck analyze --pr https://github.com/owner/repo/pull/42")
		}
		owner, repo, prNumber, err := parsePR(prURL)
		if err != nil {
			return fail(err.Error())
		}
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			return fail("missing required GITHUB_TOKEN environment variable")
		}
		ctx := context.Background()
		client := ghrepo.NewClient(token)
		toAnalyze, deleted, err := ghrepo.GetChangedFiles(ctx, client, owner, repo, prNumber)
		if err != nil {
			return err
		}
		if prContext, err := ghrepo.GetPRContext(ctx, client, owner, repo, prNumber); err != nil {
			fmt.Fprintln(os.Stderr, "warning: could not load PR context:", err)
		} else {
			printPRContext(prContext)
		}
		if len(toAnalyze) == 0 {
			fmt.Println("No .go files changed in this PR. Nothing to analyze.")
			if len(deleted) > 0 {
				printDeletedFiles(deleted)
			}
			return nil
		}
		headSHA, err := ghrepo.GetPRHeadSHA(ctx, client, owner, repo, prNumber)
		if err != nil {
			return err
		}
		results := make([]scorer.FileResult, 0, len(toAnalyze))
		for _, file := range toAnalyze {
			content, err := ghrepo.GetFileContent(ctx, client, owner, repo, file.Path, headSHA)
			if err != nil {
				return err
			}
			findings, err := analyzer.Analyze(content)
			if err != nil {
				return err
			}
			runResult, err := sandbox.Run(content)
			if err != nil {
				return err
			}
			results = append(results, scorer.ScoreFile(file.Path, findings, runResult))
		}
		report := scorer.BuildReport(results)
		printReport(report, toAnalyze)
		body := ghrepo.FormatReport(report, deletedPaths(deleted))
		if err := ghrepo.PostComment(ctx, client, owner, repo, prNumber, body); err != nil {
			fmt.Fprintln(os.Stderr, "error posting PR comment:", err)
		} else {
			fmt.Println("✅ Report posted as a comment on the PR.")
		}
		return nil
	}
	rootCmd.AddCommand(cmd)
}

func buildTestReport() (scorer.PRReport, error) {
	cleanSrc := `package main

import "fmt"

func demo() {
	fmt.Println("ok")
}`
	cases := []struct {
		filename string
		src      string
		sandbox  string
	}{
		{filename: "pkg/auth/auth.go", src: analyzer.SampleCode, sandbox: sandbox.SamplePanicCode},
		{filename: "pkg/utils/utils.go", src: cleanSrc, sandbox: sandbox.SampleGoodCode},
	}
	results := make([]scorer.FileResult, 0, len(cases))
	for _, fileCase := range cases {
		findings, err := analyzer.Analyze(fileCase.src)
		if err != nil {
			return scorer.PRReport{}, err
		}
		runResult, err := sandbox.Run(fileCase.sandbox)
		if err != nil {
			return scorer.PRReport{}, err
		}
		results = append(results, scorer.ScoreFile(fileCase.filename, findings, runResult))
	}
	return scorer.BuildReport(results), nil
}

func printReport(report scorer.PRReport, files []ghrepo.ChangedFile) {
	fmt.Println("=== PhantomCheck Report ===")
	for _, file := range report.Files {
		fmt.Println()
		fmt.Println("File:", formatFileHeader(file.Filename, files))
		fmt.Printf("Score : %.2f — %s\n", file.Score, file.Risk)
		for _, finding := range file.Findings {
			if finding.IsPhantom {
				fmt.Printf("❌ Line %d: %s.%s — does not exist\n", finding.Line, finding.Package, finding.Func)
			} else {
				fmt.Printf("✅ Line %d: %s.%s\n", finding.Line, finding.Package, finding.Func)
			}
		}
		switch {
		case file.SandboxResult.CompileError != "":
			fmt.Println("❌ Compile error:", file.SandboxResult.CompileError)
		case file.SandboxResult.Skipped:
			fmt.Println("⏭️  Sandbox skipped:", file.SandboxResult.SkipReason)
		case file.SandboxResult.TimedOut:
			fmt.Println("❌ Timed out after 5 seconds")
		case file.SandboxResult.Panicked:
			fmt.Println("❌ Runtime panic:")
			fmt.Println(file.SandboxResult.PanicTrace)
		default:
			fmt.Println("✅ Ran successfully")
			fmt.Println(file.SandboxResult.Output)
		}
	}
	fmt.Println()
	fmt.Println("=== Overall PR Summary ===")
	fmt.Printf("Score  : %.2f — %s\n", report.OverallScore, report.OverallRisk)
	fmt.Printf("Files  : %d\n", len(report.Files))
	fmt.Printf("Phantom APIs : %d\n", report.TotalPhantom)
	fmt.Printf("Panics       : %d\n", report.TotalPanics)
}

func formatFileHeader(path string, files []ghrepo.ChangedFile) string {
	for _, file := range files {
		if file.Path == path && file.Status == "renamed" && file.PreviousPath != "" {
			return fmt.Sprintf("%s (renamed from %s)", path, file.PreviousPath)
		}
	}
	return path
}

func deletedPaths(files []ghrepo.ChangedFile) []string {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	return paths
}

func printDeletedFiles(files []ghrepo.ChangedFile) {
	fmt.Println("🗑️  Deleted files in this PR (not analyzed):")
	for _, file := range files {
		fmt.Println("  -", file.Path)
	}
}

func printPRContext(ctxInfo *ghrepo.PRContext) {
	description := "none"
	if ctxInfo.Description != "" {
		description = ctxInfo.Description
		if len(description) > 100 {
			description = description[:100]
		}
	}
	headSHA := ctxInfo.HeadSHA
	if len(headSHA) > 8 {
		headSHA = headSHA[:8]
	}
	fmt.Printf("PR Title      : %s\n", ctxInfo.Title)
	fmt.Printf("Description   : %s\n", description)
	fmt.Printf("Commits       : %d commit messages fetched\n", len(ctxInfo.CommitMessages))
	fmt.Printf("Head SHA      : %s\n", headSHA)
}

func printLLMClientTest() {
	fmt.Println()
	fmt.Println("=== LLM Client Test ===")
	groqKey := os.Getenv("GROQ_API_KEY")
	if groqKey == "" {
		fmt.Println("⏭️  LLM semantic analysis skipped: GROQ_API_KEY not set")
		return
	}
	client := llm.NewClient(groqKey)
	response, err := client.Complete("You are a concise code review assistant.", "In one sentence, what does this Go code do: fmt.Println(\"hello world\")")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error running LLM semantic analysis:", err)
		return
	}
	fmt.Println(response)
}

func parsePR(raw string) (string, string, int, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" {
		return "", "", 0, fmt.Errorf("invalid PR URL: %s\nexpected: https://github.com/<owner>/<repo>/pull/<number>", raw)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 4 || parts[2] != "pull" || parts[0] == "" || parts[1] == "" || parts[3] == "" {
		return "", "", 0, fmt.Errorf("invalid PR URL: %s\nexpected: https://github.com/<owner>/<repo>/pull/<number>", raw)
	}
	number := 0
	for _, r := range parts[3] {
		if r < '0' || r > '9' {
			return "", "", 0, fmt.Errorf("invalid PR URL: %s\nexpected: https://github.com/<owner>/<repo>/pull/<number>", raw)
		}
		number = number*10 + int(r-'0')
	}
	return parts[0], parts[1], number, nil
}

func fail(message string) error { fmt.Fprintln(os.Stderr, "error:", message); os.Exit(1); return nil }
