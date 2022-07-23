package main

import (
	"fmt"

	"github.com/jritter/openscap-report-publisher/openscap_report_parser"
)

func main() {
	result := openscap_report_parser.ParseReport("resources/arf.xml")


	for _, result := range result.RuleResults {
		if result.Result != "notselected" {
			fmt.Printf("ID: %s\tResult: %s\n", result.IdRef, result.Result)
		}
	}

}
