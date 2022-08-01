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
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const reportsDirsKey = "REPORT_DIRS"

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
	var reportDirs = os.Getenv(reportsDirsKey)

	if reportDirs == "" {
		reportDirs = "resources/arf"
	}

	files, err := ioutil.ReadDir(reportDirs)
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
					gauge := prometheus.NewGauge(prometheus.GaugeOpts{
						Name: "openscap_results",
						Help: "OpenSCAP Results",
						ConstLabels: prometheus.Labels{
							"openscap_ref": result.IDRef,
							"severity":     result.Severity,
							"target":       report.Target,
							"profile":      report.Profile.IDRef},
					})

					prometheus.Register(gauge)

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

	ticker := time.NewTicker(5000 * time.Millisecond)
	done := make(chan bool)

	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				fmt.Println("Rerendering...", t)
				handleReports()
			}
		}
	}()

	handleReports()

	reports := http.FileServer(http.Dir("resources/reports/"))
	static := http.FileServer(http.Dir("resources/static/"))
	http.Handle("/reports/", http.StripPrefix("/reports/", reports))
	http.Handle("/static/", http.StripPrefix("/static/", static))
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/render", renderHandler)
	http.ListenAndServe(":2112", nil)
}
