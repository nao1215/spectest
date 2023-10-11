package spectest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	htmlTemplate "html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type (
	htmlTemplateModel struct {
		Title          string
		SubTitle       string
		StatusCode     int
		BadgeClass     string
		LogEntries     []LogEntry
		WebSequenceDSL string
		MetaJSON       htmlTemplate.JS
	}

	// SequenceDiagramFormatter implementation of a ReportFormatter
	SequenceDiagramFormatter struct {
		storagePath string
		fs          fileSystem
	}

	fileSystem interface {
		create(name string) (*os.File, error)
		mkdirAll(path string, perm os.FileMode) error
	}

	osFileSystem struct{}

	webSequenceDiagramDSL struct {
		data  bytes.Buffer
		count int
		meta  *Meta
	}
)

func (r *osFileSystem) create(name string) (*os.File, error) {
	return os.Create(filepath.Clean(name))
}

func (r *osFileSystem) mkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (r *webSequenceDiagramDSL) addRequestRow(source string, target string, description string) {
	r.addRow("->", source, target, description)
}

func (r *webSequenceDiagramDSL) addResponseRow(source string, target string, description string) {
	r.addRow("->>", source, target, description)
}

func (r *webSequenceDiagramDSL) addRow(operation, source string, target string, description string) {
	if r.meta.ConsumerName != "" {
		source = strings.ReplaceAll(source, ConsumerDefaultName, r.meta.ConsumerName)
		target = strings.ReplaceAll(target, ConsumerDefaultName, r.meta.ConsumerName)
	}
	if r.meta.TestingTargetName != "" {
		source = strings.ReplaceAll(source, SystemUnderTestDefaultName, r.meta.TestingTargetName)
		target = strings.ReplaceAll(target, SystemUnderTestDefaultName, r.meta.TestingTargetName)
	}

	r.count++
	r.data.WriteString(fmt.Sprintf("%s%s%s: (%d) %s\n",
		quoted(source),
		operation,
		quoted(target),
		r.count,
		description),
	)
}

func (r *webSequenceDiagramDSL) toString() string {
	return r.data.String()
}

// Format formats the events received by the recorder
func (r *SequenceDiagramFormatter) Format(recorder *Recorder) {
	output, err := newHTMLTemplateModel(recorder)
	if err != nil {
		panic(err)
	}

	template, err := htmlTemplate.New("sequenceDiagram").
		Funcs(*templateFunc).
		Parse(reportTemplate)
	if err != nil {
		panic(err)
	}

	var out bytes.Buffer
	err = template.Execute(&out, output)
	if err != nil {
		panic(err)
	}

	fileName := fmt.Sprintf("%s.html", recorder.Meta.reportFileName())
	err = r.fs.mkdirAll(r.storagePath, os.ModePerm)
	if err != nil {
		panic(err)
	}
	saveFilesTo := fmt.Sprintf("%s/%s", r.storagePath, fileName)

	f, err := r.fs.create(saveFilesTo)
	if err != nil {
		panic(err)
	}

	s, _ := filepath.Abs(saveFilesTo)
	_, err = f.WriteString(out.String())
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created sequence diagram (%s): %s\n", fileName, filepath.FromSlash(s))
}

// SequenceDiagram produce a sequence diagram at the given path or .sequence by default
func SequenceDiagram(path ...string) *SequenceDiagramFormatter {
	var storagePath string
	if len(path) == 0 {
		storagePath = ".sequence"
	} else {
		storagePath = path[0]
	}
	return &SequenceDiagramFormatter{storagePath: storagePath, fs: &osFileSystem{}}
}

var templateFunc = &htmlTemplate.FuncMap{
	"inc": func(i int) int {
		return i + 1
	},
}

func formatDiagramRequest(req *http.Request) string {
	out := fmt.Sprintf("%s %s", req.Method, req.URL.Path)
	if req.URL.RawQuery != "" {
		out = fmt.Sprintf("%s?%s", out, req.URL.RawQuery)
	}
	if len(out) > 65 {
		return fmt.Sprintf("%s...", out[:65])
	}
	return out
}

func badgeCSSClass(status int) string {
	class := "badge badge-success"
	if status >= 400 && status < 500 {
		class = "badge badge-warning"
	} else if status >= 500 {
		class = "badge badge-danger"
	}
	return class
}

func newHTMLTemplateModel(r *Recorder) (htmlTemplateModel, error) {
	if len(r.Events) == 0 {
		return htmlTemplateModel{}, errors.New("no events are defined")
	}
	var logs []LogEntry
	webSequenceDiagram := &webSequenceDiagramDSL{meta: r.Meta}

	for _, event := range r.Events {
		switch v := event.(type) {
		case HTTPRequest:
			httpReq := v.Value
			webSequenceDiagram.addRequestRow(v.Source, v.Target, formatDiagramRequest(httpReq))
			entry, err := NewHTTPRequestLogEntry(httpReq)
			if err != nil {
				return htmlTemplateModel{}, err
			}
			entry.Timestamp = v.Timestamp
			logs = append(logs, entry)
		case HTTPResponse:
			webSequenceDiagram.addResponseRow(v.Source, v.Target, strconv.Itoa(v.Value.StatusCode))
			entry, err := NewHTTPResponseLogEntry(v.Value)
			if err != nil {
				return htmlTemplateModel{}, err
			}
			entry.Timestamp = v.Timestamp
			logs = append(logs, entry)
		case MessageRequest:
			webSequenceDiagram.addRequestRow(v.Source, v.Target, v.Header)
			logs = append(logs, LogEntry{Header: v.Header, Body: v.Body, Timestamp: v.Timestamp})
		case MessageResponse:
			webSequenceDiagram.addResponseRow(v.Source, v.Target, v.Header)
			logs = append(logs, LogEntry{Header: v.Header, Body: v.Body, Timestamp: v.Timestamp})
		default:
			panic("received unknown event type")
		}
	}

	status, err := r.ResponseStatus()
	if err != nil {
		return htmlTemplateModel{}, err
	}

	jsonMeta, err := json.Marshal(r.Meta)
	if err != nil {
		return htmlTemplateModel{}, err
	}

	return htmlTemplateModel{
		WebSequenceDSL: webSequenceDiagram.toString(),
		LogEntries:     logs,
		Title:          r.Title,
		SubTitle:       r.SubTitle,
		StatusCode:     status,
		BadgeClass:     badgeCSSClass(status),
		//#nosec
		// FIXME: G203 (CWE-79): The used method does not auto-escape HTML.
		// This can potentially lead to 'Cross-site Scripting' vulnerabilities,
		// in case the attacker controls the input. (Confidence: LOW, Severity: MEDIUM)
		MetaJSON: htmlTemplate.JS(jsonMeta),
	}, nil
}

func formatBodyContent(bodyReadCloser io.ReadCloser, replaceBody func(replacementBody io.ReadCloser)) (string, error) {
	if bodyReadCloser == nil {
		return "", nil
	}

	body, err := io.ReadAll(bodyReadCloser)
	if err != nil {
		return "", err
	}

	replaceBody(io.NopCloser(bytes.NewReader(body)))

	buf := new(bytes.Buffer)
	if json.Valid(body) {
		jsonEncodeErr := json.Indent(buf, body, "", "    ")
		if jsonEncodeErr != nil {
			return "", jsonEncodeErr
		}
		s := buf.String()
		return s, nil
	}

	_, err = buf.Write(body)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func quoted(in string) string {
	return fmt.Sprintf("%q", in)
}
