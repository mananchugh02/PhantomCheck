package sandbox

import (
	"strings"
	"testing"
)

func TestRun_Success(t *testing.T) {
	result, err := Run(SampleGoodCode)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.CompileError != "" {
		t.Fatalf("expected no compile error, got %q", result.CompileError)
	}
	if result.Panicked {
		t.Fatalf("expected no panic, got %+v", result)
	}
	if !strings.Contains(result.Output, "hello from sandbox") {
		t.Fatalf("expected output to contain hello from sandbox, got %q", result.Output)
	}
}

func TestRun_Panic(t *testing.T) {
	result, err := Run(SamplePanicCode)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.CompileError != "" {
		t.Fatalf("expected no compile error, got %q", result.CompileError)
	}
	if !result.Panicked {
		t.Fatalf("expected panic, got %+v", result)
	}
	if result.PanicTrace == "" {
		t.Fatalf("expected panic trace, got empty string")
	}
}
