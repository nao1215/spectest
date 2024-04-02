package spectest_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/nao1215/spectest"
	"github.com/tenntenn/testtime"
)

func TestIntervalDuration(t *testing.T) {
	type fields struct {
		start time.Time
		end   time.Time
	}
	tests := []struct {
		name   string
		fields fields
		want   time.Duration
	}{
		{
			name: "get 1[s] duration",
			fields: fields{
				start: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				end:   time.Date(2023, 1, 1, 0, 0, 1, 0, time.UTC),
			},
			want: time.Second,
		},
		{
			name: "get 1[ns] duration",
			fields: fields{
				start: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				end:   time.Date(2023, 1, 1, 0, 0, 0, 1, time.UTC),
			},
			want: time.Nanosecond,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			interval := spectest.NewInterval()

			if !testtime.SetTime(t, tt.fields.start) {
				t.Fatal("failed to set start time")
			}
			interval.Start()

			if !testtime.SetTime(t, tt.fields.end) {
				t.Fatal("failed to set end time")
			}
			interval.End()

			if interval.Duration() != tt.want {
				t.Errorf("duration should be 1 second, but %s", interval.Duration())
			}
		})
	}
}

// nolint
func ExampleInterval_Duration() {
	t := &testing.T{}
	interval := spectest.NewInterval()

	// Set started time. Usually, you don't need to set the time. You only call Start() method.
	if !testtime.SetTime(t, time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("failed to set start time")
	}
	interval.Start()

	// Set finished time. Usually, you don't need to set the time. You only call End() method.
	if !testtime.SetTime(t, time.Date(2023, 1, 1, 0, 0, 1, 0, time.UTC)) {
		t.Fatal("failed to set end time")
	}
	interval.End()

	fmt.Printf("duration=%f[s]", interval.Duration().Seconds())
	// Output: duration=1.000000[s]
}
