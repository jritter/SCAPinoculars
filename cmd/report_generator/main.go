package main

import "github.com/jritter/openscap-report-publisher/reportrenderer"

func main() {
	reportrenderer.RenderReport("resources/arf/arf.xml", "resources/reports/report.html")
}
