package report

import "time"

type Report struct {
	HTMLReport string
	ARFReport  string
	Date       time.Time
	IDRef      string
	Target     string
}
