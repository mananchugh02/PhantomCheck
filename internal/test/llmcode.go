package main

import (
	"fmt"
	"os"
	"strings"
)

func processFile(path string) {
	// Real calls
	content, err := os.ReadFile(path)
	if err != nil {
		fmt.Println(err)
	}

	// Phantom API calls — these do NOT exist
	lines := os.ReadFileLines(path)
	upper := strings.ToUpperCase(string(content))
	fmt.PrintRed("error found")

	// Runtime panic — index out of range
	var results []string
	fmt.Println(results[0])

	fmt.Println(lines, upper)
}