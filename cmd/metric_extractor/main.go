package main

import (
	"fmt"
	"log"

	"github.com/jritter/openscap-report-publisher/pkg/reportparser"
)

func main() {
	result, err := reportparser.ParseReport("resources/reports/report1/arf.xml")
	if err != nil {
		log.Println(err)
	}

	for _, result := range result.RuleResults {
		if result.Result != "notselected" {
			fmt.Printf("ID: %s\tResult: %s\n", result.IDRef, result.Result)
		}
	}

}
