package semantic

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mananchugh02/phantomcheck/internal/github"
	"github.com/mananchugh02/phantomcheck/internal/llm"
)

type SemanticFinding struct {
	Description string `json:"description"`
	Severity    string `json:"severity"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
}

type SemanticResult struct {
	Findings   []SemanticFinding
	Skipped    bool
	SkipReason string
}

func Analyze(client *llm.Client, file github.ChangedFile, pr github.PRContext) (SemanticResult, error) {
	if file.Patch == "" {
		return SemanticResult{Skipped: true, SkipReason: "no diff available for this file"}, nil
	}
	description := truncate(pr.Description, 500, "")
	diff := truncate(file.Patch, 3000, "[diff truncated to 3000 chars]")
	systemPrompt := "You are a senior Go code reviewer specializing in detecting logical bugs and mismatches between stated intent and actual implementation. You respond only in valid JSON. Never include markdown, backticks, or explanation outside the JSON."
	userPrompt := fmt.Sprintf("PR TITLE: %s\nPR DESCRIPTION: %s\nCOMMIT MESSAGES:\n%s\nFILE: %s\nDIFF:\n%s\n\nAnalyze this diff against the stated intent. Find logical bugs, missing edge cases, incorrect conditions, or mismatches between what the commit claims and what the code actually does.\n\nRespond ONLY with a JSON object in this exact format:\n{\n  \"findings\": [\n    {\n      \"description\": \"<clear explanation of the issue>\",\n      \"severity\": \"HIGH|MEDIUM|LOW\",\n      \"start_line\": <integer or 0 if unknown>,\n      \"end_line\": <integer or 0 if unknown>\n    }\n  ]\n}\nIf no issues are found, return: {\"findings\": []}", pr.Title, description, strings.Join(pr.CommitMessages, "\n"), file.Path, diff)
	response, err := client.Complete(systemPrompt, userPrompt)
	if err != nil {
		return SemanticResult{}, err
	}
	findings, err := parseFindings(response)
	if err != nil {
		return SemanticResult{}, err
	}
	return SemanticResult{Findings: findings}, nil
}

func truncate(value string, limit int, suffix string) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	if len([]rune(suffix)) >= limit {
		return string([]rune(suffix)[:limit])
	}
	return string(runes[:limit-len([]rune(suffix))]) + suffix
}

func parseFindings(raw string) ([]SemanticFinding, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(raw, "```") {
		lines := strings.Split(raw, "\n")
		if len(lines) >= 2 {
			lines = lines[1:]
			if strings.TrimSpace(lines[len(lines)-1]) == "```" {
				lines = lines[:len(lines)-1]
			}
			raw = strings.TrimSpace(strings.Join(lines, "\n"))
		}
	}
	var parsed struct {
		Findings []SemanticFinding `json:"findings"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, fmt.Errorf("could not parse semantic findings: %w", err)
	}
	return parsed.Findings, nil
}
