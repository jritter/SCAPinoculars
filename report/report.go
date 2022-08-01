package report

import "time"

type Report struct {
	HTMLReport  string
	ARFReport   string
	Date        time.Time
	IDRef       string
	Target      string
	PassedRules int
	FailedRules int
}

func (r Report) PercentPassed() float64 {
	return float64(r.PassedRules) / float64(r.PassedRules + r.FailedRules) * float64(100)
}

func (r Report) PercentFailed() float64 {
	return float64(r.FailedRules) / float64(r.PassedRules + r.FailedRules) * float64(100)
}
