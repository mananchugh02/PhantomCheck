package analyzer

import "testing"

func TestAnalyze_PhantomCalls(t *testing.T) {
	src := `package main

import (
	"fmt"
	"os"
	"strings"
)

func demo() {
	_, _ = os.ReadFile("input.txt")
	_ = strings.Contains("phantom", "ant")
	fmt.Println("ok")
	_, _ = os.ReadFileLines("input.txt")
	_ = strings.ContainsIgnoreCase("phantom", "ANT")
	fmt.PrintRed("bad")
}`
	findings, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(findings) != 6 {
		t.Fatalf("expected 6 findings, got %d", len(findings))
	}
	want := map[string]bool{"os.ReadFile": false, "strings.Contains": false, "fmt.Println": false, "os.ReadFileLines": true, "strings.ContainsIgnoreCase": true, "fmt.PrintRed": true}
	for _, finding := range findings {
		key := finding.Package + "." + finding.Func
		wantPhantom, ok := want[key]
		if !ok {
			t.Fatalf("unexpected finding %s on line %d", key, finding.Line)
		}
		if finding.IsPhantom != wantPhantom {
			t.Fatalf("finding %s on line %d: expected IsPhantom=%v, got %v", key, finding.Line, wantPhantom, finding.IsPhantom)
		}
		delete(want, key)
	}
	if len(want) != 0 {
		t.Fatalf("missing findings for: %v", want)
	}
}

func TestAnalyze_AllReal(t *testing.T) {
	src := `package main

import (
	"fmt"
	"os"
	"strings"
)

func demo() {
	_, _ = os.ReadFile("input.txt")
	_ = strings.Contains("phantom", "ant")
	fmt.Println("ok")
}`
	findings, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	for _, finding := range findings {
		if finding.IsPhantom {
			t.Fatalf("expected no phantom findings, got %+v", finding)
		}
	}
}

func TestAnalyze_AllPhantom(t *testing.T) {
	src := `package main

import (
	"fmt"
	"os"
	"strings"
)

func demo() {
	_, _ = os.ReadFileLines("input.txt")
	_ = strings.ContainsIgnoreCase("phantom", "ANT")
	fmt.PrintRed("bad")
}`
	findings, err := Analyze(src)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if len(findings) != 3 {
		t.Fatalf("expected 3 findings, got %d", len(findings))
	}
	for _, finding := range findings {
		if !finding.IsPhantom {
			t.Fatalf("expected all findings to be phantom, got %+v", finding)
		}
	}
}

func TestAnalyze_PhantomMethodCalls(t *testing.T) {
	findings, err := Analyze(SampleMethodCode)
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	want := map[string]bool{
		"strings.Builder.WriteString": false,
		"strings.Builder.WriteStrnig": true,
		"User.GetName":                false,
		"User.GetEmail":               true,
	}
	for _, finding := range findings {
		key := finding.Package + "." + finding.Func
		if wantPhantom, ok := want[key]; ok {
			if finding.IsPhantom != wantPhantom {
				t.Fatalf("finding %s on line %d: expected IsPhantom=%v, got %v", key, finding.Line, wantPhantom, finding.IsPhantom)
			}
			delete(want, key)
		}
	}
	if len(want) != 0 {
		t.Fatalf("missing method findings for: %v", want)
	}
}
