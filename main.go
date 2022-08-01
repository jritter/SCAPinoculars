package main

import (
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/jritter/openscap-report-publisher/report"
	"github.com/jritter/openscap-report-publisher/reportparser"
	"github.com/jritter/openscap-report-publisher/reportrenderer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const reportsDirsKey = "REPORT_DIRS"

var reports = []report.Report{}

func renderHandler(w http.ResponseWriter, r *http.Request) {
	handleReports()
}

func handleReports() {

	var reportDirs = strings.Split(os.Getenv(reportsDirsKey), " ")

	if reportDirs[0] == "" {
		reportDirs[0] = "resources/arf"
	}

	for _, dir := range reportDirs {
		filepath.Walk(dir, handleReportFile)
	}

	reportIndexTemplate, _ := template.ParseFiles("templates/index.tmpl")
	reportIndexFile, err := os.Create("resources/reports/index.html")

	reportIndexTemplate.Execute(reportIndexFile, reports)
	if err != nil {
		log.Println("create file: ", reports)
		return
	}

}

func handleReportFile(path string, info fs.FileInfo, err error) error {

	if strings.HasSuffix(path, ".xml") {
		log.Printf("Processing file %s\n", path)
		xmlreport := reportparser.ParseReport(path)

		passed, failed := 0, 0

		// Prometheus Metrics
		// Each report has multiple RuleResults
		for _, result := range xmlreport.RuleResults {

			// We only care about RuleResults with state pass or fail
			if result.Result == "fail" || result.Result == "pass" {

				// Create Prometheus gauge for each RuleResult
				// and we add report and result specific labels
				gauge := prometheus.NewGauge(prometheus.GaugeOpts{
					Name: "openscap_results",
					Help: "OpenSCAP Results",
					ConstLabels: prometheus.Labels{
						"openscap_ref": result.IDRef,
						"severity":     result.Severity,
						"target":       xmlreport.Target,
						"profile":      xmlreport.Profile.IDRef},
				})

				prometheus.Register(gauge)

				// gauge value 0 means fail, gauge vaule 1 means pass
				if result.Result == "fail" {
					gauge.Set(0)
				} else {
					gauge.Set(1)
				}
			}

			switch result.Result {
			case "pass":
				passed++
			case "fail":
				failed++
			}
		}

		// HTML Report
		filename := xmlreport.Profile.IDRef + "_" + xmlreport.Target + "_" + xmlreport.StartTime.Format("200601021504") + ".html"
		log.Println(filename)

		// Check if report alrady exists, and render if it doesn't
		_, err := os.Stat("resources/reports/" + filename)
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("Report %s is not available, rendering... ", filename)
			reportrenderer.RenderReport("resources/arf/"+path, "resources/reports/"+filename)
			log.Println("Done")
		} else {
			log.Println("Report is already there, not doing anything")
		}

		report := report.Report{HTMLReport: filename,
			ARFReport:   path,
			Date:        xmlreport.StartTime,
			IDRef:       xmlreport.Profile.IDRef,
			Target:      xmlreport.Target,
			PassedRules: passed,
			FailedRules: failed}

		reports = append(reports, report)
	}
	return nil

}

func main() {

	// periodically retrigger the rendering function
	ticker := time.NewTicker(5000 * time.Millisecond)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				log.Println("Rerendering...", t)
				handleReports()
			}
		}
	}()

	handleReports()

	// This endpoint serves the rendered reports
	reports := http.FileServer(http.Dir("resources/reports/"))
	http.Handle("/reports/", http.StripPrefix("/reports/", reports))

	// Endpoint to manually trigger the rendering function
	http.HandleFunc("/render", renderHandler)

	// This endpoint serves the Prometheus metrics
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
