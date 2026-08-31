package scorer

import (
	"testing"

	"github.com/mananchugh02/phantomcheck/internal/analyzer"
	"github.com/mananchugh02/phantomcheck/internal/sandbox"
)

func TestScoreFile_WithIssues(t *testing.T) {
	findings := []analyzer.Finding{{IsPhantom: true}, {IsPhantom: true}, {IsPhantom: false}}
	result := sandbox.SandboxResult{Panicked: true}
	file := ScoreFile("pkg/auth/auth.go", findings, result)
	if file.Score != 1.0 {
		t.Fatalf("expected score 1.0, got %v", file.Score)
	}
	if file.Risk != "HIGH" {
		t.Fatalf("expected risk HIGH, got %q", file.Risk)
	}
}

func TestScoreFile_NoIssues(t *testing.T) {
	file := ScoreFile("pkg/utils/utils.go", nil, sandbox.SandboxResult{})
	if file.Score != 0.0 {
		t.Fatalf("expected score 0.0, got %v", file.Score)
	}
	if file.Risk != "LOW" {
		t.Fatalf("expected risk LOW, got %q", file.Risk)
	}
}

func TestBuildReport(t *testing.T) {
	files := []FileResult{
		{Filename: "a.go", Score: 0.7, Findings: []analyzer.Finding{{IsPhantom: true}}, SandboxResult: sandbox.SandboxResult{Panicked: true}},
		{Filename: "b.go", Score: 0.0, Findings: []analyzer.Finding{{IsPhantom: false}, {IsPhantom: false}}, SandboxResult: sandbox.SandboxResult{}},
	}
	report := BuildReport(files)
	if report.OverallScore != 0.35 {
		t.Fatalf("expected overall score 0.35, got %v", report.OverallScore)
	}
	if report.OverallRisk != "MEDIUM" {
		t.Fatalf("expected overall risk MEDIUM, got %q", report.OverallRisk)
	}
	if report.TotalPhantom != 1 {
		t.Fatalf("expected total phantom 1, got %d", report.TotalPhantom)
	}
	if report.TotalPanics != 1 {
		t.Fatalf("expected total panics 1, got %d", report.TotalPanics)
	}
}

func TestScoreFile_SkippedSandboxIgnoresRuntimePenalties(t *testing.T) {
	findings := []analyzer.Finding{{IsPhantom: true}}
	result := sandbox.SandboxResult{Skipped: true, SkipReason: "not runnable", Panicked: true, CompileError: "boom"}
	file := ScoreFile("pkg/auth/auth.go", findings, result)
	if file.Score != 0.35 {
		t.Fatalf("expected score 0.35 from phantom findings only, got %v", file.Score)
	}
	if file.Risk != "MEDIUM" {
		t.Fatalf("expected risk MEDIUM, got %q", file.Risk)
	}
}
