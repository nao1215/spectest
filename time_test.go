package spectest

import (
	"testing"
	"time"

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
			interval := NewInterval()

			if !testtime.SetTime(t, tt.fields.start) {
				t.Fatal("start time should not be set")
			}
			interval.Start()

			if !testtime.SetTime(t, tt.fields.end) {
				t.Fatal("end time should not be set")
			}
			interval.End()

			if interval.Duration() != tt.want {
				t.Errorf("duration should be 1 second, but %s", interval.Duration())
			}
		})
	}
}
