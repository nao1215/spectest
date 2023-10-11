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
func (sdf *SequenceDiagramFormatter) Format(recorder *Recorder) {
	output, err := sdf.newHTMLTemplateModel(recorder)
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
	err = sdf.fs.mkdirAll(sdf.storagePath, os.ModePerm)
	if err != nil {
		panic(err)
	}
	saveFilesTo := filepath.Join(sdf.storagePath, fileName)

	f, err := sdf.fs.create(saveFilesTo)
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
	"contains": func(str string, subs ...string) bool {
		for _, sub := range subs {
			if strings.Contains(str, sub) {
				return true
			}
		}
		return false
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

func (sdf *SequenceDiagramFormatter) newHTMLTemplateModel(recorder *Recorder) (htmlTemplateModel, error) {
	if len(recorder.Events) == 0 {
		return htmlTemplateModel{}, errors.New("no events are defined")
	}
	var logs []LogEntry
	webSequenceDiagram := &webSequenceDiagramDSL{meta: recorder.Meta}

	for i, event := range recorder.Events {
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
			// If the Content Type is an image, display the image in the report instead of the response body (binary).
			contentType := extractContentType(entry.Header)
			if isImage(contentType) {
				generateImage(entry.Body, sdf.storagePath, recorder.Meta.reportFileName(), contentType, i)
				entry.Body = filepath.Clean(imagePath(sdf.storagePath, recorder.Meta.reportFileName(), contentType, i))
			}

			entry.Timestamp = v.Timestamp
			logs = append(logs, entry)
		case MessageRequest:
			webSequenceDiagram.addRequestRow(v.Source, v.Target, v.Header)
			logs = append(logs, LogEntry{Header: v.Header, Body: v.Body, Timestamp: v.Timestamp})
		case MessageResponse:
			webSequenceDiagram.addResponseRow(v.Source, v.Target, v.Header)

			// If the Content Type is an image, display the image in the report instead of the response body (binary).
			contentType := extractContentType(v.Header)
			body := v.Body
			if isImage(contentType) {
				generateImage(v.Body, sdf.storagePath, recorder.Meta.reportFileName(), contentType, i)
				body = filepath.Clean(imagePath(sdf.storagePath, recorder.Meta.reportFileName(), contentType, i))
			}
			logs = append(logs, LogEntry{Header: v.Header, Body: body, Timestamp: v.Timestamp})
		default:
			panic("received unknown event type")
		}
	}

	status, err := recorder.ResponseStatus()
	if err != nil {
		return htmlTemplateModel{}, err
	}

	jsonMeta, err := json.Marshal(recorder.Meta)
	if err != nil {
		return htmlTemplateModel{}, err
	}

	return htmlTemplateModel{
		WebSequenceDSL: webSequenceDiagram.toString(),
		LogEntries:     logs,
		Title:          recorder.Title,
		SubTitle:       recorder.SubTitle,
		StatusCode:     status,
		BadgeClass:     badgeCSSClass(status),
		//#nosec
		// FIXME: G203 (CWE-79): The used method does not auto-escape HTML.
		// This can potentially lead to 'Cross-site Scripting' vulnerabilities,
		// in case the attacker controls the input. (Confidence: LOW, Severity: MEDIUM)
		MetaJSON: htmlTemplate.JS(jsonMeta),
	}, nil
}

// extractContentType extracts the content type from the header.
// If the content type is not found, it returns an empty string.
// Original source:  "GET /path HTTP/1.1\r\nHost: example.com\r\nContent-Type: application/json\r\n\r\n"
// Extract: "application/json"
func extractContentType(headers string) string {
	for _, header := range strings.Split(headers, "\r\n") {
		if strings.HasPrefix(header, "Content-Type") {
			return strings.TrimSpace(strings.TrimPrefix(header, "Content-Type:"))
		}
	}
	return ""
}

// isImage returns true if the content type is an image.
func isImage(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/gif", "image/svg+xml", "image/bmp", "image/webp", "image/tiff", "image/x-icon":
		return true
	}
	return false
}

// generateImage generates an image from the body.
func generateImage(body, dir, name, contentType string, index int) {
	file, err := os.Create(filepath.Clean(imagePath(dir, name, contentType, index)))
	if err != nil {
		panic(err) //FIXME: error handling
	}
	defer file.Close()

	_, err = file.Write([]byte(body))
	if err != nil {
		panic(err) //FIXME: error handling
	}
}

// imagePath returns the image name.
func imagePath(dir, name, contentType string, index int) string {
	return fmt.Sprintf("%s/%s_%d.%s", dir, name, index, toImageExt(contentType))
}

// toImageExt returns the image extension based on the content type.
func toImageExt(contentType string) string {
	switch contentType {
	case "image/jpeg":
		return "jpeg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/svg+xml":
		return "svg"
	case "image/bmp":
		return "bmp"
	case "image/webp":
		return "webp"
	case "image/tiff":
		return "tiff"
	case "image/x-icon":
		return "ico"
	}
	return ""
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
