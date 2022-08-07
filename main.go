package main

import (
	"compress/bzip2"
	"errors"
	"io"
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

const reportsDirKey = "REPORT_DIR"

var reportDir = ""

var reports = make(map[string]report.Report)

func renderHandler(w http.ResponseWriter, r *http.Request) {
	handleReports()
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	pwd, _ := os.Getwd()
	reportIndexTemplate, err := template.ParseFiles(pwd + "/templates/index.tmpl")
	if err != nil {
		log.Panicf("Could not open template file: %v", err)
	}
	reportIndexTemplate.Execute(w, reports)
}

func handleReports() {
	filepath.Walk(reportDir, handleCompressedReports)
	filepath.Walk(reportDir, handleReportFile)
}

func handleCompressedReports(path string, info fs.FileInfo, err error) error {
	// If the reports are compressed, we need to uncompress them
	// before we can parse and render them.
	if strings.HasSuffix(path, ".bz2") && !info.IsDir() {
		inputFile, err := os.Open(path)
		if err != nil {
			log.Println(err)
			return err
		}

		defer inputFile.Close()

		outputFile, err := os.Create(strings.TrimSuffix(path, ".bz2"))

		if err != nil {
			log.Println(err)
			return err
		}

		defer outputFile.Close()

		bzip2reader := bzip2.NewReader(inputFile)

		if err != nil {
			log.Println(err)
			return err
		}

		io.Copy(outputFile, bzip2reader)
	}
	return nil
}

func handleReportFile(path string, info fs.FileInfo, err error) error {

	if strings.HasSuffix(path, ".xml") && !info.IsDir() {
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

		// Check if report alrady exists, and render if it doesn't
		_, err := os.Stat(filepath.Dir(path) + "/" + filename)
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("Report %s is not available, rendering... ", filename)
			reportrenderer.RenderReport(path, filepath.Dir(path)+"/"+filename)
			log.Println("Done")
		} else {
			log.Println("Report is already there, not doing anything")
		}

		reportURL := "/reports" + strings.TrimPrefix(filepath.Dir(path)+"/"+filename, reportDir)

		report := report.Report{HTMLReport: reportURL,
			ARFReport:   path,
			Date:        xmlreport.StartTime,
			IDRef:       xmlreport.Profile.IDRef,
			Target:      xmlreport.Target,
			PassedRules: passed,
			FailedRules: failed}

		reports[reportURL] = report
	}
	return nil

}

func main() {

	// periodically retrigger the rendering function
	ticker := time.NewTicker(60 * time.Second)
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

	reportDir = os.Getenv(reportsDirKey)

	if reportDir == "" {
		reportDir = "resources/reports"
	}

	handleReports()

	http.HandleFunc("/", indexHandler)
	reportserver := http.FileServer(http.Dir(reportDir + "/"))
	http.Handle("/reports/", http.StripPrefix("/reports/", reportserver))

	// Endpoint to manually trigger the rendering function
	http.HandleFunc("/render", renderHandler)

	// This endpoint serves the Prometheus metrics
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":2112", nil)
	if err != nil {
		log.Panic(err)
	}
}
