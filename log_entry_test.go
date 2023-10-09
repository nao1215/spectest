package spectest

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewHTTPRequestLogEntry(t *testing.T) {
	type args struct {
		req *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    LogEntry
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				req: &http.Request{
					Method:     http.MethodGet,
					URL:        &url.URL{Path: "/path"},
					Proto:      "HTTP/1.1",
					ProtoMajor: 1,
					ProtoMinor: 1,
					Host:       "example.com",
					Body:       io.NopCloser(bytes.NewBufferString("request body")),
				},
			},
			want: LogEntry{
				Header: "GET /path HTTP/1.1\r\nHost: example.com\r\n\r\n",
				Body:   "request body",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewHTTPRequestLogEntry(tt.args.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHTTPRequestLogEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("value is mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNewHTTPResponseLogEntry(t *testing.T) {
	type args struct {
		res *http.Response
	}
	tests := []struct {
		name    string
		args    args
		want    LogEntry
		wantErr bool
	}{
		{
			name: "test",
			args: args{
				res: &http.Response{
					ProtoMajor:    1,
					ProtoMinor:    1,
					StatusCode:    http.StatusOK,
					ContentLength: 21,
					Body:          io.NopCloser(bytes.NewBufferString("response body")),
				},
			},
			want: LogEntry{
				Header: "HTTP/1.1 200 OK\r\nContent-Length: 21\r\n\r\n",
				Body:   "response body",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewHTTPResponseLogEntry(tt.args.res)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHTTPResponseLogEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("value is mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
