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

	"github.com/jritter/scapinoculars/pkg/report"
	"github.com/jritter/scapinoculars/pkg/reportdata"
	"github.com/jritter/scapinoculars/pkg/reportparser"
	"github.com/jritter/scapinoculars/pkg/reportrenderer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const reportsDirKey = "REPORT_DIR"
const renderIntervalKey = "RENDER_INTERVAL"
const reportsOutputKey = "REPORT_OUTPUT_DIR"

var reportDir = ""
var reportOutputDir = ""
var pwd, _ = os.Getwd()

var reportData *reportdata.ReportData = &reportdata.ReportData{Reports: make(map[string]report.Report), Targets: []string{}, Profiles: []string{}}

func renderHandler(w http.ResponseWriter, r *http.Request) {
	handleReports()
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	reportIndexTemplate, err := template.ParseFiles(pwd + "/templates/index.tmpl")
	if err != nil {
		log.Panicf("Could not open template file: %v", err)
	}
	if err = reportIndexTemplate.Execute(w, reportData); err != nil {
		log.Panicf("Could not render index template: %v", err)
	}
}

func handleReports() {
	reportFolderInfo, err := os.Stat(reportDir)
	if err != nil {
		log.Printf("directory %s does not exist, terminating.\n", reportDir)
		os.Exit(1)
	}
	if !reportFolderInfo.IsDir() {
		log.Printf("%s is not a directory, terminating.\n", reportDir)
		os.Exit(1)
	}

	outputFolderInfo, err := os.Stat(reportOutputDir)
	if err != nil {
		log.Printf("directory %s does not exist, terminating.\n", reportOutputDir)
		os.Exit(1)
	}
	if !outputFolderInfo.IsDir() {
		log.Printf("%s is not a directory, terminating.\n", reportOutputDir)
		os.Exit(1)
	}

	if err := filepath.Walk(reportDir, handleCompressedReports); err != nil {
		log.Panicf("Could not decompress Reports: %v", err)
	}
	if err := filepath.Walk(reportOutputDir, handleReportFile); err != nil {
		log.Panicf("Could not parse Reports: %v", err)
	}
	housekeep()
}

func handleCompressedReports(path string, info fs.FileInfo, err error) error {
	if !info.IsDir() {
		log.Printf("Handling file %s\n", path)
		// Derrive the name of the uncompressed report in the target directory
		uncompressedFile := strings.TrimSuffix(strings.Replace(path, reportDir, reportOutputDir, 1), ".bzip2")

		// Create the Target directory if necessary
		if err := os.MkdirAll(filepath.Dir(uncompressedFile), os.ModePerm); err != nil {
			log.Fatal(err)
		}

		if strings.HasSuffix(path, ".bzip2") {

			// If the reports are compressed, we need to uncompress them
			// before we can parse and render them.

			_, err := os.Stat(uncompressedFile)
			if errors.Is(err, os.ErrNotExist) {

				log.Printf("Uncompressing file %s\n", path)
				inputFile, err := os.Open(path)
				if err != nil {
					log.Println(err)
					return err
				}

				outputFile, err := os.Create(uncompressedFile)

				if err != nil {
					log.Println(err)
					return err
				}

				bzip2reader := bzip2.NewReader(inputFile)

				_, err = io.Copy(outputFile, bzip2reader)
				if err != nil {
					log.Println(err)
					return err
				}

				err = inputFile.Close()
				if err != nil {
					log.Printf("failed to close input file: %v", err)
				}

				err = outputFile.Close()
				if err != nil {
					log.Printf("failed to close output file: %v", err)
				}

			}
		} else {
			log.Printf("File is %s already uncompressed, skipping...", path)

			if _, err := os.Stat(uncompressedFile); errors.Is(err, os.ErrNotExist) {
				log.Printf("File is not available in output Directory, copying...")
				bytesRead, err := os.ReadFile(path)
				if err != nil {
					log.Fatal(err)
				}
				err = os.WriteFile(uncompressedFile, bytesRead, 0644)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}
	return nil
}

func handleReportFile(path string, info fs.FileInfo, err error) error {

	if strings.HasSuffix(path, ".xml") && !info.IsDir() {

		// Let's see if we already parsed the report
		_, exists := reportData.Reports[path]

		// Only parse the file if it hasn't been parsed yet
		if !exists {

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

					if err = prometheus.Register(gauge); err != nil {
						are := &prometheus.AlreadyRegisteredError{}
						if errors.As(err, are) {
							// A gauge for that metric has been registered before.
							// Use the old counter from now on.
							gauge = are.ExistingCollector.(prometheus.Gauge)
						} else {
							log.Panic(err)
						}
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

			reportURL := "/reports" + strings.TrimPrefix(filepath.Dir(path)+"/"+filename, reportOutputDir)

			report := report.Report{HTMLReport: reportURL,
				ARFReport:   path,
				Date:        xmlreport.StartTime,
				IDRef:       xmlreport.Profile.IDRef,
				Target:      xmlreport.Target,
				PassedRules: passed,
				FailedRules: failed}

			reportData.Reports[path] = report

			reportData.Targets = appendIfNotExist(reportData.Targets, report.Target)
			reportData.Profiles = appendIfNotExist(reportData.Profiles, report.IDRef)

		} else {
			log.Printf("No need to process file %s as it has already been parsed\n", path)
		}
	}
	return nil
}

func housekeep() {
	for path := range reportData.Reports {
		_, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			log.Printf("Report %s no longer exists, deleting from cache\n", path)
			delete(reportData.Reports, path)
		}
	}
}

func appendIfNotExist(slice []string, item string) []string {
	for _, i := range slice {
		if i == item {
			return slice
		}
	}
	return append(slice, item)
}

func main() {
	reportDir = os.Getenv(reportsDirKey)
	reportOutputDir = os.Getenv(reportsOutputKey)

	if reportDir == "" {
		reportDir = "resources/reports"
	}

	// We are using the same directory to render the reports into
	// if nothing else is configured
	if reportOutputDir == "" {
		reportOutputDir = reportDir
	}

	// 3600 seconds is the default value
	var renderIntervalDuration time.Duration = 3600
	renderInterval := os.Getenv(renderIntervalKey)
	if renderInterval != "" {
		renderIntervalDurationInt, err := strconv.ParseInt(renderInterval, 10, 0)
		if err != nil {
			log.Printf("Could ot parse environment variable %s, using the default of %s\n", renderIntervalKey, renderIntervalDuration*time.Second)
		} else {
			renderIntervalDuration = time.Duration(renderIntervalDurationInt)
		}
	}

	// periodically retrigger the rendering function
	log.Printf("Rendering reports every %s\n", renderIntervalDuration*time.Second)
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

	styles := http.FileServer(http.Dir(pwd + "/styles"))
	http.Handle("/styles/", http.StripPrefix("/styles/", styles))

	reportserver := http.FileServer(http.Dir(reportOutputDir + "/"))
	http.Handle("/reports/", http.StripPrefix("/reports/", reportserver))

	// Endpoint to manually trigger the rendering function
	http.HandleFunc("/render", renderHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "2112"
	}

	log.Println("SCAPinoculars started")
	log.Printf("Publisher looks for reports in %s\n", reportDir)
	log.Printf("Publisher renders reports in %s\n", reportOutputDir)
	log.Printf("Listening on port %s\n", port)
	// This endpoint serves the Prometheus metrics
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Panic(err)
	}
}
