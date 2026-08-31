package sandbox

import (
	"bytes"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type SandboxResult struct {
	CompileError string
	Panicked     bool
	PanicTrace   string
	Output       string
	TimedOut     bool
	Skipped      bool
	SkipReason   string
}

func Run(src string) (SandboxResult, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "source.go", src, parser.AllErrors)
	if err != nil {
		return SandboxResult{Skipped: true, SkipReason: "could not parse source: " + strings.TrimSpace(err.Error())}, nil
	}
	if !isStandaloneMain(file) {
		return SandboxResult{Skipped: true, SkipReason: "not a standalone `package main` file with `func main()`"}, nil
	}

	tempDir, err := os.MkdirTemp("", "phantomcheck-sandbox-*")
	if err != nil {
		return SandboxResult{}, err
	}
	defer os.RemoveAll(tempDir)

	sourcePath := filepath.Join(tempDir, "main.go")
	if err := os.WriteFile(sourcePath, []byte(src), 0o600); err != nil {
		return SandboxResult{}, err
	}
	binaryName := "sandbox-bin"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	binaryPath := filepath.Join(tempDir, binaryName)
	compileCtx, cancelCompile := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelCompile()
	compileCmd := exec.CommandContext(compileCtx, "go", "build", "-o", binaryPath, sourcePath)
	var compileErr bytes.Buffer
	compileCmd.Stderr = &compileErr
	if err := compileCmd.Run(); err != nil {
		return SandboxResult{CompileError: strings.TrimSpace(compileErr.String())}, nil
	}
	runCtx, cancelRun := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelRun()
	runCmd := exec.CommandContext(runCtx, binaryPath)
	var stdout, stderr bytes.Buffer
	runCmd.Stdout = &stdout
	runCmd.Stderr = &stderr
	if err := runCmd.Run(); err != nil {
		if runCtx.Err() == context.DeadlineExceeded {
			return SandboxResult{TimedOut: true}, nil
		}
	}
	result := SandboxResult{Output: stdout.String()}
	trace := stderr.String()
	result.Output = strings.TrimRight(result.Output, "\n")
	if strings.Contains(trace, "goroutine") && strings.Contains(trace, "panic") {
		result.Panicked = true
		result.PanicTrace = trace
	}
	if result.Output == "" && trace != "" && !result.Panicked {
		result.Output = trace
	}
	return result, nil
}

func isStandaloneMain(file *ast.File) bool {
	if file == nil || file.Name == nil || file.Name.Name != "main" {
		return false
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Recv != nil || fn.Name == nil || fn.Name.Name != "main" {
			continue
		}
		if fn.Type != nil && (fn.Type.Params == nil || len(fn.Type.Params.List) == 0) {
			return true
		}
	}
	return false
}
