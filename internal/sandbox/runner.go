package sandbox

import (
	"bytes"
	"context"
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
}

func Run(src string) (SandboxResult, error) {
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
