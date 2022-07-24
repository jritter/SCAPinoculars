package main

import (
	"fmt"

	"github.com/jritter/openscap-report-publisher/reportparser"
)

func main() {
	result := reportparser.ParseReport("resources/arf.xml")

	for _, result := range result.RuleResults {
		if result.Result != "notselected" {
			fmt.Printf("ID: %s\tResult: %s\n", result.IDRef, result.Result)
		}
	}

}
