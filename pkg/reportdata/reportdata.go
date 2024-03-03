package reportdata

import "github.com/jritter/scapinoculars/pkg/report"

// a ReportData struct holds data of an OpenScap Report
type ReportData struct {
	Reports  map[string]report.Report
	Targets  []string
	Profiles []string
}