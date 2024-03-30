// Package plantuml write plantuml markup to a writer
package plantuml

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/nao1215/spectest"
)

const requestOperation = "->"
const responseOperation = "-->"

type Formatter struct {
	writer io.Writer
}

type DSL struct {
	count int
	data  bytes.Buffer
}

func (r *DSL) AddRequestRow(source, target, description, body string) {
	r.addRow(requestOperation, source, target, description, body)
}

func (r *DSL) AddResponseRow(source, target, description, body string) {
	r.addRow(responseOperation, source, target, description, body)
}

func (r *DSL) addRow(operation, source, target, description, body string) {
	var notePosition = "left"
	if operation == requestOperation {
		notePosition = "right"
	}

	var note string
	if body != "" {
		note = fmt.Sprintf("\nnote %s\n%s\nend note", notePosition, escape(body))
	}

	r.count += 1
	r.data.WriteString(fmt.Sprintf("%s%s%s: (%d) %s %s\n",
		source,
		operation,
		target,
		r.count,
		description,
		note))
}

func (r *DSL) ToString() string {
	return fmt.Sprintf("\n@startuml\nskinparam noteFontSize 11\nskinparam monochrome true\n%s\n@enduml", r.data.String())
}

func (r *Formatter) Format(recorder *spectest.Recorder) {
	var sb strings.Builder

	meta, err := json.Marshal(recorder.Meta)
	if err != nil {
		panic(err)
	}
	markup, err := buildMarkup(recorder)
	if err != nil {
		panic(err)
	}

	sb.Write(meta)
	sb.Write([]byte(markup))

	_, err = r.writer.Write([]byte(sb.String()))
	if err != nil {
		panic(err)
	}
}

func NewFormatter(writer io.Writer) spectest.ReportFormatter {
	return &Formatter{writer: writer}
}

func buildMarkup(r *spectest.Recorder) (string, error) {
	if len(r.Events) == 0 {
		return "", errors.New("no events are defined")
	}

	dsl := &DSL{}
	for _, event := range r.Events {
		switch v := event.(type) {
		case spectest.HTTPRequest:
			httpReq := v.Value
			entry, err := spectest.NewHTTPRequestLogEntry(httpReq)
			if err != nil {
				return "", err
			}
			entry.Timestamp = v.Timestamp
			dsl.AddRequestRow(v.Source, v.Target, fmt.Sprintf("%s %s", httpReq.Method, httpReq.URL), formatNote(entry))
		case spectest.HTTPResponse:
			entry, err := spectest.NewHTTPResponseLogEntry(v.Value)
			if err != nil {
				return "", err
			}
			entry.Timestamp = v.Timestamp
			dsl.AddResponseRow(v.Source, v.Target, strconv.Itoa(v.Value.StatusCode), formatNote(entry))
		case spectest.MessageRequest:
			dsl.AddRequestRow(v.Source, v.Target, v.Header, v.Body)
		case spectest.MessageResponse:
			dsl.AddResponseRow(v.Source, v.Target, v.Header, v.Body)
		default:
			panic("received unknown event type")
		}
	}

	return dsl.ToString(), nil
}

func escape(in string) string {
	return in
}

func formatNote(entry spectest.LogEntry) string {
	return fmt.Sprintf("%s%s", entry.Header, entry.Body)
}
