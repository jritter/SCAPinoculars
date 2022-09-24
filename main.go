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
	"strconv"
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
const renderIntervalKey = "RENDER_INTERVAL"

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
	if err = reportIndexTemplate.Execute(w, reports); err != nil {
		log.Panicf("Could not render index template: %v", err)
	}
}

func handleReports() {
	if err := filepath.Walk(reportDir, handleCompressedReports); err != nil {
		log.Panicf("Could not decompress Reports: %v", err)
	}
	if err := filepath.Walk(reportDir, handleReportFile); err != nil {
		log.Panicf("Could not parse Reports: %v", err)
	}
	housekeep()
}

func handleCompressedReports(path string, info fs.FileInfo, err error) error {
	// If the reports are compressed, we need to uncompress them
	// before we can parse and render them.
	if strings.HasSuffix(path, ".bzip2") && !info.IsDir() {

		// Derrive output file

		uncompressedFile := strings.TrimSuffix(path, ".bzip2")

		_, err := os.Stat(uncompressedFile)
		if errors.Is(err, os.ErrNotExist) {


			log.Printf("Uncompressing file %s\n", path)
			inputFile, err := os.Open(path)
			if err != nil {
				log.Println(err)
				return err
			}

			defer inputFile.Close()

			outputFile, err := os.Create(uncompressedFile)

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

			_, err = io.Copy(outputFile, bzip2reader)
			if err != nil {
				log.Println(err)
				return err
			}

		} else {
			log.Printf("File is %s already uncompressed, skipping...", uncompressedFile)
		}
	}
	return nil
}

func handleReportFile(path string, info fs.FileInfo, err error) error {

	if strings.HasSuffix(path, ".xml") && !info.IsDir() {

		// Let's see if we already parsed the report
		_, exists := reports[path]

		// Only parse the file if it hasn't been parsed yet
		if ! exists {

			log.Printf("Processing file %s\n", path)
			xmlreport, err := reportparser.ParseReport(path)
			if err != nil {
				log.Println(err)
				return err
			}

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

					err = prometheus.Register(gauge)
					if err != nil {
						log.Println(err)
						return err
					}

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
			_, err = os.Stat(filepath.Dir(path) + "/" + filename)
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

			reports[path] = report
		} else {
			log.Printf("No need to process file %s as it has already been parsed\n", path)
		}
	}
	return nil
}

func housekeep() {
	for path := range reports {
		_, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("Report %s no longer exists, deleting from cache\n", path)
			delete(reports, path)
		}
	}
}

func main() {
	reportDir = os.Getenv(reportsDirKey)

	if reportDir == "" {
		reportDir = "resources/reports"
	}

	// 3600 seconds is the default value
	var renderIntervalDuration time.Duration = 3600
	renderInterval := os.Getenv(renderIntervalKey)
	if renderInterval != "" {
		renderIntervalDurationInt, err := strconv.ParseInt(renderInterval, 10, 0)
		if (err != nil){
			log.Printf("Could ot parse environment variable %s, using the default of %s\n", renderIntervalKey, renderIntervalDuration * time.Second)
		}	else {
			renderIntervalDuration = time.Duration(renderIntervalDurationInt)
		}
	}

	// periodically retrigger the rendering function
	log.Printf("Rendering reports every %s\n", renderIntervalDuration * time.Second)
	ticker := time.NewTicker(renderIntervalDuration * time.Second)
	done := make(chan bool)

	go func() {
		// initial load
		handleReports()
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

	http.HandleFunc("/", indexHandler)
	reportserver := http.FileServer(http.Dir(reportDir + "/"))
	http.Handle("/reports/", http.StripPrefix("/reports/", reportserver))

	// Endpoint to manually trigger the rendering function
	http.HandleFunc("/render", renderHandler)

	log.Println("OpenSCAP Report Publisher started")
	log.Printf("Publishser looks for reports in %s\n", reportDir)
	log.Println("Listening on port 2112")
	// This endpoint serves the Prometheus metrics
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":2112", nil)
	if err != nil {
		log.Panic(err)
	}
}
