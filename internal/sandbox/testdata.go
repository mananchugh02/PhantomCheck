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
