package domain

type Metric struct {
	TotalJobsCreated int
	JobsCompleted    int
	JobsFailed       int
	JobsRetried      int
	JobsInProgress   int
}

func NewMetric() *Metric {
	return &Metric{
		TotalJobsCreated: 0,
		JobsCompleted:    0,
		JobsFailed:       0,
		JobsRetried:      0,
		JobsInProgress:   0,
	}
}
