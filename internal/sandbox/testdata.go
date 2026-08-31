package sandbox

const SampleGoodCode = `package main

import "fmt"

func main() {
	fmt.Println("hello from sandbox")
}`

const SamplePanicCode = `package main

import "fmt"

func main() {
	fmt.Println([]int{}[1])
}`

const SampleLibraryCode = `package auth

import (
	"fmt"
	"os"
	"strings"
)

func CheckSession(path string) bool {
	_, _ = os.ReadFile(path)
	return strings.Contains("auth-token", "token")
}

func RenderStatus() {
	fmt.Println("auth package ready")
	_, _ = os.ReadFileLines("session.txt")
	_ = strings.ToUpperCase("pending")
	fmt.PrintRed("denied")
}`
