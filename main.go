package main

import (
	"fmt"
	"net/http"

	"github.com/jritter/openscap-report-publisher/openscap_report_parser"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)


func main() {

	result := openscap_report_parser.ParseReport("resources/arf.xml")

	for _, result := range result.RuleResults {
		if result.Result != "notselected" {
			fmt.Printf("ID: %s\tResult: %s\n", result.IdRef, result.Result)
			gauge := promauto.NewGauge(prometheus.GaugeOpts{
				Name: "openscap_results",
				Help: "OpenSCAP Results",
				ConstLabels: prometheus.Labels{"openscap_ref": result.IdRef, "severity": result.Severity},
			})
			if (result.Result == "fail"){
				gauge.Set(0)
			} else {
				gauge.Set(1)
			}
		}
	}

	

	fs := http.FileServer(http.Dir("resources/"))
	http.Handle("/resources/", http.StripPrefix("/resources/", fs))
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
