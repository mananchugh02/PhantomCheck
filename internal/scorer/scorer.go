package scorer

import (
	"github.com/mananchugh02/phantomcheck/internal/analyzer"
	"github.com/mananchugh02/phantomcheck/internal/sandbox"
)

type FileResult struct {
	Filename      string
	Score         float64
	Risk          string
	Findings      []analyzer.Finding
	SandboxResult sandbox.SandboxResult
}

type PRReport struct {
	Files        []FileResult
	OverallScore float64
	OverallRisk  string
	TotalPhantom int
	TotalPanics  int
}

func ScoreFile(filename string, findings []analyzer.Finding, result sandbox.SandboxResult) FileResult {
	score := 0.0
	for _, finding := range findings {
		if finding.IsPhantom {
			score += 0.35
		}
	}
	if result.Panicked {
		score += 0.40
	}
	if result.CompileError != "" {
		score += 0.50
	}
	if score > 1.0 {
		score = 1.0
	}
	return FileResult{Filename: filename, Score: score, Risk: riskFor(score), Findings: findings, SandboxResult: result}
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
