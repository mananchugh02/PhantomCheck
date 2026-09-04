package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"

	"github.com/mananchugh02/phantomcheck/internal/analyzer"
	ghrepo "github.com/mananchugh02/phantomcheck/internal/github"
	"github.com/mananchugh02/phantomcheck/internal/llm"
	reportpkg "github.com/mananchugh02/phantomcheck/internal/report"
	"github.com/mananchugh02/phantomcheck/internal/sandbox"
	"github.com/mananchugh02/phantomcheck/internal/scorer"
	"github.com/mananchugh02/phantomcheck/internal/semantic"
	"github.com/spf13/cobra"
)

type fileAnalysisResult struct {
	file          ghrepo.ChangedFile
	findings      []analyzer.Finding
	sandboxResult sandbox.SandboxResult
	semResult     semantic.SemanticResult
	fileResult    scorer.FileResult
	err           error
}

func init() {
	cmd := &cobra.Command{Use: "analyze", Run: func(_ *cobra.Command, _ []string) {}}
	cmd.Flags().String("pr", "", "GitHub pull request URL")
	cmd.Flags().Bool("test", false, "Run the static analyzer against bundled sample code")
	cmd.Flags().String("output", "text", "Output format: text or json")
	cmd.Flags().String("fail-on", "", "Exit with code 1 if overall risk meets or exceeds this level. Valid values: LOW, MEDIUM, HIGH")
	cmd.RunE = func(cmd *cobra.Command, _ []string) error {
		output, _ := cmd.Flags().GetString("output")
		if output != "text" && output != "json" {
			return fail("invalid --output value: expected text or json")
		}
		failOn, _ := cmd.Flags().GetString("fail-on")
		failOn = strings.ToUpper(strings.TrimSpace(failOn))
		if failOn != "" && failOn != "LOW" && failOn != "MEDIUM" && failOn != "HIGH" {
			return fail("invalid --fail-on value: expected LOW, MEDIUM, or HIGH")
		}
		testMode, _ := cmd.Flags().GetBool("test")
		if testMode {
			report, err := buildTestReport()
			if err != nil {
				return err
			}
			if output == "json" {
				exitCode := exitCodeFor(report.OverallRisk, failOn)
				if err := printJSONReport(report, exitCode); err != nil {
					return err
				}
				maybeFail(report.OverallRisk, failOn, exitCode)
				return nil
			}
			printReport(report, nil)
			printLLMClientTest()
			maybeFail(report.OverallRisk, failOn, exitCodeFor(report.OverallRisk, failOn))
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
		prContext := ghrepo.PRContext{}
		if fetchedContext, err := ghrepo.GetPRContext(ctx, client, owner, repo, prNumber); err != nil {
			fmt.Fprintln(os.Stderr, "warning: could not load PR context:", err)
		} else {
			prContext = *fetchedContext
			if output == "text" {
				printPRContext(fetchedContext)
			}
		}
		if len(toAnalyze) == 0 {
			if output == "text" {
				fmt.Println("No .go files changed in this PR. Nothing to analyze.")
			}
			if len(deleted) > 0 && output == "text" {
				printDeletedFiles(deleted)
			}
			if output == "json" {
				return printJSONReport(scorer.BuildReport(nil), 0)
			}
			return nil
		}
		headSHA, err := ghrepo.GetPRHeadSHA(ctx, client, owner, repo, prNumber)
		if err != nil {
			return err
		}
		groqClient := (*llm.Client)(nil)
		if groqKey := os.Getenv("GROQ_API_KEY"); groqKey != "" {
			groqClient = llm.NewClient(groqKey)
		}
		sem := make(chan struct{}, 5)
		resultsCh := make(chan fileAnalysisResult, len(toAnalyze))
		for _, file := range toAnalyze {
			file := file
			sem <- struct{}{}
			go func() {
				defer func() { <-sem }()
				result := fileAnalysisResult{file: file, semResult: semantic.SemanticResult{Skipped: true, SkipReason: "GROQ_API_KEY not set"}}
				content, err := ghrepo.GetFileContent(ctx, client, owner, repo, file.Path, headSHA)
				if err != nil {
					result.err = err
					resultsCh <- result
					return
				}
				result.findings, err = analyzer.Analyze(content)
				if err != nil {
					result.err = err
					resultsCh <- result
					return
				}
				result.sandboxResult, err = sandbox.Run(content)
				if err != nil {
					result.err = err
					resultsCh <- result
					return
				}
				if groqClient != nil {
					result.semResult, err = semantic.Analyze(groqClient, file, prContext)
					if err != nil {
						fmt.Fprintln(os.Stderr, "warning: semantic analysis failed for", file.Path, ":", err)
						result.semResult = semantic.SemanticResult{Skipped: true, SkipReason: "semantic analysis failed"}
					}
				}
				result.fileResult = scorer.ScoreFile(file.Path, result.findings, result.sandboxResult, result.semResult)
				resultsCh <- result
			}()
		}
		results := make([]fileAnalysisResult, 0, len(toAnalyze))
		for range toAnalyze {
			results = append(results, <-resultsCh)
		}
		sort.Slice(results, func(i, j int) bool { return results[i].file.Path < results[j].file.Path })
		fileResults := make([]scorer.FileResult, 0, len(results))
		for _, result := range results {
			if result.err != nil {
				return result.err
			}
			fileResults = append(fileResults, result.fileResult)
		}
		report := scorer.BuildReport(fileResults)
		if output == "text" {
			printReport(report, toAnalyze)
		}
		body := ghrepo.FormatReport(report, deletedPaths(deleted))
		if err := ghrepo.UpsertComment(ctx, client, owner, repo, prNumber, body); err != nil {
			fmt.Fprintln(os.Stderr, "error posting PR comment:", err)
		} else {
			if output == "text" {
				fmt.Println("✅ Report posted to PR as a review comment.")
			} else {
				fmt.Fprintln(os.Stderr, "✅ Report posted to PR as a review comment.")
			}
		}
		if output == "json" {
			exitCode := exitCodeFor(report.OverallRisk, failOn)
			if err := printJSONReport(report, exitCode); err != nil {
				return err
			}
			maybeFail(report.OverallRisk, failOn, exitCode)
		}
		maybeFail(report.OverallRisk, failOn, exitCodeFor(report.OverallRisk, failOn))
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
		results = append(results, scorer.ScoreFile(fileCase.filename, findings, runResult, semantic.SemanticResult{}))
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
		fmt.Println("=== Semantic Analysis ===")
		if file.SemanticSkipped {
			fmt.Println("⏭️  Semantic analysis skipped:", file.SemanticSkipReason)
		} else if len(file.SemanticFindings) == 0 {
			fmt.Println("✅ No logical issues found")
		} else {
			for _, finding := range file.SemanticFindings {
				fmt.Printf("%s %s (lines %d-%d)\n", semanticEmoji(finding.Severity), finding.Description, finding.StartLine, finding.EndLine)
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
	fmt.Printf("Semantic issues : %d\n", report.TotalSemanticIssues)
}

func printJSONReport(report scorer.PRReport, exitCode int) error {
	jsonReport := reportpkg.BuildJSONReport(report)
	jsonReport.ExitCode = exitCode
	data, err := json.MarshalIndent(jsonReport, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func exitCodeFor(risk, threshold string) int {
	if threshold != "" && riskRank(risk) >= riskRank(threshold) {
		return 1
	}
	return 0
}

func riskRank(risk string) int {
	switch strings.ToUpper(risk) {
	case "HIGH":
		return 3
	case "MEDIUM":
		return 2
	case "LOW":
		return 1
	default:
		return 0
	}
}

func maybeFail(risk, threshold string, exitCode int) {
	if exitCode != 1 {
		return
	}
	fmt.Fprintf(os.Stderr, "PhantomCheck: overall risk %s meets or exceeds --fail-on threshold %s. Exiting with code 1.\n", risk, threshold)
	os.Exit(1)
}

func semanticEmoji(severity string) string {
	switch severity {
	case "HIGH":
		return "🔴"
	case "MEDIUM":
		return "🟡"
	default:
		return "🟢"
	}
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
