package main

import (
	"fmt"
	"os/exec"
)

func main() {
	_, err := exec.Command(
		"oscap",
		"xccdf",
		"generate",
		"report",
		"--output",
		"resources/report.html",
		"resources/arf.xml").Output()
	if err != nil {
		switch e := err.(type) {
		case *exec.Error:
			fmt.Println("failed executing:", err)
		case *exec.ExitError:
			fmt.Println("command exit rc =", e.ExitCode())
		default:
			panic(err)
		}
	}
}
