package spectest

import (
	"net/http"
	"testing"
)

func TestMetaReportFileName(t *testing.T) {
	type fields struct {
		ConsumerName      string
		Duration          int64
		Host              string
		Method            string
		Name              string
		Path              string
		ReportFileName    string
		StatusCode        int
		TestingTargetName string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "get custom report file name",
			fields: fields{
				ReportFileName: "custom-report-file-name",
				Method:         http.MethodGet,
				Path:           "/api/v1/users",
			},
			want: "custom-report-file-name",
		},
		{
			name: "get default report file name (it's a hash)",
			fields: fields{
				ReportFileName: "",
				Method:         http.MethodGet,
				Path:           "/api/v1/users",
			},
			want: "405834851_2166136261",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Meta{
				ConsumerName:      tt.fields.ConsumerName,
				Duration:          tt.fields.Duration,
				Host:              tt.fields.Host,
				Method:            tt.fields.Method,
				Name:              tt.fields.Name,
				Path:              tt.fields.Path,
				ReportFileName:    tt.fields.ReportFileName,
				StatusCode:        tt.fields.StatusCode,
				TestingTargetName: tt.fields.TestingTargetName,
			}
			if got := m.reportFileName(); got != tt.want {
				t.Errorf("Meta.reportFileName() = %v, want %v", got, tt.want)
			}
		})
	}
}
