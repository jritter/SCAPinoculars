package reportparser

import (
	"encoding/xml"
	"io"
	"os"
	"time"
)

// Represents an AssetReportCollection Object on a OpenSCAP XCCDF Report
type AssetReportCollection struct {
	XMLName xml.Name `xml:"asset-report-collection"`
	Reports Reports  `xml:"reports"`
}

// Represents a Report List Object on a OpenSCAP XCCDF Report
type Reports struct {
	XMLName xml.Name `xml:"reports"`
	Reports []Report `xml:"report"`
}

// Represents a Report Object on a OpenSCAP XCCDF Report
type Report struct {
	XMLName xml.Name `xml:"report"`
	ID      string   `xml:"id,attr"`
	Content Content  `xml:"content"`
}

// Represents a Content Object on a OpenSCAP XCCDF Report
type Content struct {
	XMLName    xml.Name   `xml:"content"`
	TestResult TestResult `xml:"TestResult"`
}

type TestResult struct {
	XMLName     xml.Name     `xml:"TestResult"`
	ID          string       `xml:"id,attr"`
	StartTime   time.Time    `xml:"start-time,attr"`
	EndTime     time.Time    `xml:"end-time,attr"`
	RuleResults []RuleResult `xml:"rule-result"`
	Title       string       `xml:"title"`
	Profile     Profile      `xml:"profile"`
	Target      string       `xml:"target"`
}

type Profile struct {
	XMLName xml.Name `xml:"profile"`
	IDRef   string   `xml:"idref,attr"`
}

type RuleResult struct {
	XMLName  xml.Name `xml:"rule-result"`
	IDRef    string   `xml:"idref,attr"`
	Severity string   `xml:"severity,attr"`
	Result   string   `xml:"result"`
}

func ParseReport(file string) (TestResult, error) {

	var testResult TestResult

	// Open our xmlFile
	xmlFile, err := os.Open(file)
	// if we os.Open returns an error then handle it
	if err != nil {
		return testResult, err
	}

	// defer the closing of our xmlFile so that we can parse it later on
	byteValue, _ := io.ReadAll(xmlFile)
	defer xmlFile.Close()

	var assetReportCollection AssetReportCollection

	if err = xml.Unmarshal(byteValue, &assetReportCollection); err != nil {
		return testResult, err
	}
	testResult = assetReportCollection.Reports.Reports[0].Content.TestResult
	return testResult, nil
}
