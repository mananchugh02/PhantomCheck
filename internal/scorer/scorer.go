package scorer

import (
	"github.com/mananchugh02/phantomcheck/internal/analyzer"
	"github.com/mananchugh02/phantomcheck/internal/sandbox"
	"github.com/mananchugh02/phantomcheck/internal/semantic"
)

type FileResult struct {
	Filename           string
	Score              float64
	Risk               string
	Findings           []analyzer.Finding
	SemanticFindings   []semantic.SemanticFinding
	SemanticSkipped    bool
	SemanticSkipReason string
	SandboxResult      sandbox.SandboxResult
}

type PRReport struct {
	Files               []FileResult
	OverallScore        float64
	OverallRisk         string
	TotalPhantom        int
	TotalPanics         int
	TotalSemanticIssues int
}

func ScoreFile(filename string, findings []analyzer.Finding, result sandbox.SandboxResult, semResult semantic.SemanticResult) FileResult {
	score := 0.0
	for _, finding := range findings {
		if finding.IsPhantom {
			score += 0.35
		}
	}
	if !result.Skipped {
		if result.Panicked {
			score += 0.40
		}
		if result.CompileError != "" {
			score += 0.50
		}
	}
	for _, finding := range semResult.Findings {
		switch finding.Severity {
		case "HIGH":
			score += 0.30
		case "MEDIUM":
			score += 0.15
		case "LOW":
			score += 0.05
		}
	}
	if score > 1.0 {
		score = 1.0
	}
	return FileResult{Filename: filename, Score: score, Risk: riskFor(score), Findings: findings, SemanticFindings: semResult.Findings, SemanticSkipped: semResult.Skipped, SemanticSkipReason: semResult.SkipReason, SandboxResult: result}
}

func BuildReport(files []FileResult) PRReport {
	report := PRReport{Files: files}
	if len(files) == 0 {
		return report
	}
	for _, file := range files {
		report.OverallScore += file.Score
		for _, finding := range file.Findings {
			if finding.IsPhantom {
				report.TotalPhantom++
			}
		}
		if file.SandboxResult.Panicked {
			report.TotalPanics++
		}
		report.TotalSemanticIssues += len(file.SemanticFindings)
	}
	report.OverallScore /= float64(len(files))
	report.OverallRisk = riskFor(report.OverallScore)
	return report
}

func riskFor(score float64) string {
	switch {
	case score >= 0.6:
		return "HIGH"
	case score >= 0.3:
		return "MEDIUM"
	default:
		return "LOW"
	}
}
