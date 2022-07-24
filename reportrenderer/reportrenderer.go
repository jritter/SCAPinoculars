package reportrenderer

import (
	"fmt"
	"os/exec"
)

func RenderReport(inputFile string, outputFile string) {
	_, err := exec.Command(
		"oscap",
		"xccdf",
		"generate",
		"report",
		"--output",
		outputFile,
		inputFile).Output()
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
