package report

import (
	"github.com/mananchugh02/phantomcheck/internal/scorer"
)

type JSONFinding struct {
	Line    int    `json:"line"`
	Package string `json:"package"`
	Func    string `json:"func"`
	Phantom bool   `json:"is_phantom"`
}

type JSONSemanticFinding struct {
	Description string `json:"description"`
	Severity    string `json:"severity"`
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
}

type JSONSandbox struct {
	Skipped      bool   `json:"skipped"`
	SkipReason   string `json:"skip_reason,omitempty"`
	CompileError string `json:"compile_error,omitempty"`
	Panicked     bool   `json:"panicked"`
	PanicTrace   string `json:"panic_trace,omitempty"`
	TimedOut     bool   `json:"timed_out"`
}

type JSONFile struct {
	Filename         string                `json:"filename"`
	Score            float64               `json:"score"`
	Risk             string                `json:"risk"`
	PhantomFindings  []JSONFinding         `json:"phantom_findings"`
	Sandbox          JSONSandbox           `json:"sandbox"`
	SemanticFindings []JSONSemanticFinding `json:"semantic_findings"`
}

type JSONReport struct {
	OverallScore   float64    `json:"overall_score"`
	OverallRisk    string     `json:"overall_risk"`
	ExitCode       int        `json:"exit_code"`
	FilesAnalyzed  int        `json:"files_analyzed"`
	PhantomAPIs    int        `json:"phantom_apis"`
	Panics         int        `json:"panics"`
	SemanticIssues int        `json:"semantic_issues"`
	Files          []JSONFile `json:"files"`
}

func BuildJSONReport(report scorer.PRReport) JSONReport {
	result := JSONReport{
		OverallScore:   report.OverallScore,
		OverallRisk:    report.OverallRisk,
		FilesAnalyzed:  len(report.Files),
		PhantomAPIs:    report.TotalPhantom,
		Panics:         report.TotalPanics,
		SemanticIssues: report.TotalSemanticIssues,
		Files:          make([]JSONFile, 0, len(report.Files)),
	}
	for _, file := range report.Files {
		jsonFile := JSONFile{
			Filename:         file.Filename,
			Score:            file.Score,
			Risk:             file.Risk,
			PhantomFindings:  make([]JSONFinding, 0, len(file.Findings)),
			SemanticFindings: make([]JSONSemanticFinding, 0, len(file.SemanticFindings)),
			Sandbox: JSONSandbox{
				Skipped:      file.SandboxResult.Skipped,
				SkipReason:   file.SandboxResult.SkipReason,
				CompileError: file.SandboxResult.CompileError,
				Panicked:     file.SandboxResult.Panicked,
				PanicTrace:   file.SandboxResult.PanicTrace,
				TimedOut:     file.SandboxResult.TimedOut,
			},
		}
		for _, finding := range file.Findings {
			jsonFile.PhantomFindings = append(jsonFile.PhantomFindings, JSONFinding{Line: finding.Line, Package: finding.Package, Func: finding.Func, Phantom: finding.IsPhantom})
		}
		for _, finding := range file.SemanticFindings {
			jsonFile.SemanticFindings = append(jsonFile.SemanticFindings, JSONSemanticFinding{Description: finding.Description, Severity: finding.Severity, StartLine: finding.StartLine, EndLine: finding.EndLine})
		}
		result.Files = append(result.Files, jsonFile)
	}
	return result
}
