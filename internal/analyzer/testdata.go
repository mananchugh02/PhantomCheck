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

const SampleMethodCode = `package main

import "strings"

type User struct {
	Name string
}

func (u User) GetName() string {
	return u.Name
}

func demo() {
	var sb strings.Builder
	sb.WriteString("ok")
	sb.WriteStrnig("typo")
	user := User{Name: "Ada"}
	user.GetName()
	user.GetEmail()
}`
