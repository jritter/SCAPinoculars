package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/jritter/openscap-report-publisher/reportparser"
	"github.com/jritter/openscap-report-publisher/reportrenderer"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	reg = prometheus.NewRegistry()
)

type Report struct {
	HTMLReport string
	ARFReport  string
	Date       time.Time
	IDRef      string
	Target     string
}

func renderHandler(w http.ResponseWriter, r *http.Request) {
	handleReports()
}

func handleReports() {
	var reports = []Report{}

	// read our input directory
	files, err := ioutil.ReadDir("resources/arf")
	if err != nil {
		log.Fatal(err)
	}

	// loop over all xml files
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".xml") {
			report := reportparser.ParseReport("resources/arf/" + file.Name())

			// Prometheus Metrics
			// Each report has multiple RuleResults
			for _, result := range report.RuleResults {

				// We only care about RuleResults with state pass or fail
				if result.Result == "fail" || result.Result == "pass" {

					// Create Prometheus gauge for each RuleResult
					// and we add report and result specific labels
					gauge := promauto.With(reg).NewGauge(prometheus.GaugeOpts{
						Name: "openscap_results",
						Help: "OpenSCAP Results",
						ConstLabels: prometheus.Labels{
							"openscap_ref": result.IDRef,
							"severity":     result.Severity,
							"target":       report.Target,
							"profile":      report.Profile.IDRef},
					})

					// gauge value 0 means fail, gauge vaule 1 means pass
					if result.Result == "fail" {
						gauge.Set(0)
					} else {
						gauge.Set(1)
					}
				}
			}

			// HTML Report
			filename := report.Profile.IDRef + "_" + report.Target + "_" + report.StartTime.Format("200601021504") + ".html"
			fmt.Println(filename)

			// Check if report alrady exists, and render if it doesn't
			_, err := os.Stat("resources/reports/" + filename)
			if errors.Is(err, os.ErrNotExist) {
				fmt.Printf("Report %s is not available, rendering... ", filename)
				reportrenderer.RenderReport("resources/arf/"+file.Name(), "resources/reports/"+filename)
				fmt.Println("Done")
			} else {
				fmt.Println("Report is already there, not doing anything")
			}

			reports = append(reports, Report{HTMLReport: filename,
				ARFReport: file.Name(),
				Date:      report.StartTime,
				IDRef:     report.Profile.IDRef,
				Target:    report.Target})
		}
	}

	reportIndexTemplate, _ := template.ParseFiles("templates/index.tmpl")
	reportIndexFile, err := os.Create("resources/reports/index.html")

	reportIndexTemplate.Execute(reportIndexFile, reports)
	if err != nil {
		log.Println("create file: ", reports)
		return
	}
}

func main() {

	handleReports()

	fs := http.FileServer(http.Dir("resources/reports/"))
	http.Handle("/reports/", http.StripPrefix("/reports/", fs))
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/render", renderHandler)
	http.ListenAndServe(":2112", nil)
}
