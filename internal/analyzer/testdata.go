package analyzer

const SampleCode = `package main

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
