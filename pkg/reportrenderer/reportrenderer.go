package reportrenderer

import (
	"log"
	"os/exec"
)

func RenderReport(inputFile string, outputFile string) {
	cmd := exec.Command(
		"oscap",
		"xccdf",
		"generate",
		"report",
		"--output",
		outputFile,
		inputFile)

	output, err := cmd.CombinedOutput()
	if len(output) > 0 {
		log.Printf("oscap output:\n%s", output)
	}
	if err != nil {
		log.Println("error rendering report:", err)
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
