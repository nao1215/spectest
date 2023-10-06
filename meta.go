package spectest

import (
	"fmt"
	"hash/fnv"
	"strings"
)

// ConsumerDefaultName default consumer name
const ConsumerDefaultName = "cli"

// SystemUnderTestDefaultName default name for system under test
const SystemUnderTestDefaultName = "sut"

// Meta represents the meta data for the report.
type Meta struct {
	// ConsumerName represents the name of the consumer.
	ConsumerName string `json:"consumer_name,omitempty"`
	// Duration represents the duration of the report.
	// This is the time between the first request and the last response.
	Duration int64 `json:"duration,omitempty"`
	// Host represents the host of the report. e.g. example.com
	Host string `json:"host,omitempty"`
	// Method represents http request method of the report.
	Method string `json:"method,omitempty"`
	// Name represents the title of the report.
	Name string `json:"name,omitempty"`
	// Path represents http request url of the report. e.g. /api/v1/users
	Path string `json:"path,omitempty"`
	// ReportFileName represents the name of the report file.
	ReportFileName string `json:"report_file_name,omitempty"`
	// StatusCode represents the final http status code of the report.
	StatusCode int `json:"status_code,omitempty"`
	// TestingTargetName represents the name of the system under test.
	TestingTargetName string `json:"testing_target_name,omitempty"`
}

// newMeta creates a new meta data object.
func newMeta() *Meta {
	return &Meta{
		ConsumerName:      ConsumerDefaultName,
		TestingTargetName: SystemUnderTestDefaultName,
	}
}

// hash generates a hash for the report name.
func (m *Meta) hash() string {
	prefix := fnv.New32a()
	_, err := prefix.Write([]byte(fmt.Sprintf("%s%s", strings.ToUpper(m.Method), m.Path)))
	if err != nil {
		panic(err)
	}

	suffix := fnv.New32a()
	_, err = suffix.Write([]byte(m.Name))
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%d_%d", prefix.Sum32(), suffix.Sum32())
}

// reportFileName returns the report file name.
// If the report file name is not set, it will generate a hash.
// Return value does not include the file extension.
func (m *Meta) reportFileName() string {
	if m.ReportFileName != "" {
		return m.ReportFileName
	}
	return m.hash()
}
