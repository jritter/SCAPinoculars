package reportrenderer

import (
	"log"
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
			log.Println("failed executing:", err)
		case *exec.ExitError:
			log.Println("command exit rc =", e.ExitCode())
		default:
			panic(err)
		}
	}
}
