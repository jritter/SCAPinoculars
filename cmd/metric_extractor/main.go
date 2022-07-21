package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

type AssetReportCollection struct {
	XMLName xml.Name `xml:"asset-report-collection"`
	Reports Reports  `xml:"reports"`
}

type Reports struct {
	XMLName xml.Name `xml:"reports"`
	Reports []Report `xml:"report"`
}

type Report struct {
	XMLName xml.Name `xml:"report"`
	Id      string   `xml:"id,attr"`
	Content Content  `xml:"content"`
}

type Content struct {
	XMLName    xml.Name   `xml:"content"`
	TestResult TestResult `xml:"TestResult"`
}

type TestResult struct {
	XMLName     xml.Name     `xml:"TestResult"`
	Id          string       `xml:"id,attr"`
	StartTime   time.Time    `xml:"start-time,attr"`
	EndTime     time.Time    `xml:"end-time,attr"`
	RuleResults []RuleResult `xml:"rule-result"`
}

type RuleResult struct {
	XMLName  xml.Name `xml:"rule-result"`
	IdRef    string   `xml:"idref,attr"`
	Severity string   `xml:"severity,attr"`
	Result   string   `xml:"result"`
}

func main() {

	// Open our xmlFile
	xmlFile, err := os.Open("resources/arf.xml")
	// if we os.Open returns an error then handle it
	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Successfully opened resources/arf.xml")
	// defer the closing of our xmlFile so that we can parse it later on
	byteValue, _ := ioutil.ReadAll(xmlFile)
	defer xmlFile.Close()

	var assetReportCollection AssetReportCollection
	xml.Unmarshal(byteValue, &assetReportCollection)

    fmt.Println(assetReportCollection.Reports.Reports[0].Content.TestResult.StartTime)

	for _, result := range assetReportCollection.Reports.Reports[0].Content.TestResult.RuleResults {
		if result.Result != "notselected" {
			fmt.Printf("ID: %s\tResult: %s\n", result.IdRef, result.Result)
		}
	}

}
