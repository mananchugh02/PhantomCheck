package semantic

import (
	"testing"

	"github.com/mananchugh02/phantomcheck/internal/github"
)

func TestAnalyze_SkipsEmptyPatch(t *testing.T) {
	result, err := Analyze(nil, github.ChangedFile{}, github.PRContext{})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if !result.Skipped {
		t.Fatalf("expected semantic analysis to be skipped")
	}
	if result.SkipReason == "" {
		t.Fatalf("expected a skip reason")
	}
}

func TestParseFindings(t *testing.T) {
	findings, err := parseFindings(`{"findings":[{"description":"unchecked error","severity":"HIGH","start_line":4,"end_line":4},{"description":"missing edge case","severity":"LOW","start_line":8,"end_line":10}]}`)
	if err != nil {
		t.Fatalf("parseFindings returned error: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
	if findings[0].Description != "unchecked error" || findings[0].Severity != "HIGH" {
		t.Fatalf("unexpected first finding: %+v", findings[0])
	}
	if findings[1].Description != "missing edge case" || findings[1].Severity != "LOW" {
		t.Fatalf("unexpected second finding: %+v", findings[1])
	}
}
