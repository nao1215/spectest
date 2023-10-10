package spectest

import (
	"time"

	"github.com/tenntenn/testtime"
)

// Interval represents a time interval.
type Interval struct {
	// Started is the Started time of the interval.
	Started time.Time
	// Finished is the Finished time of the interval.
	Finished time.Time
}

// NewInterval creates a new interval.
// This method is not set the start and end time of the interval.
func NewInterval() *Interval {
	return &Interval{}
}

// Start sets the start time of the interval.
func (i *Interval) Start() *Interval {
	i.Started = testtime.Now()
	return i
}

// End sets the end time of the interval.
func (i *Interval) End() *Interval {
	i.Finished = testtime.Now()
	return i
}

// Duration returns the duration of the interval.
func (i Interval) Duration() time.Duration {
	return i.Finished.Sub(i.Started)
}
