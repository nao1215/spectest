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

	md "github.com/nao1215/markdown"
	"github.com/nao1215/mermaid/sequence"
)

type (
	// htmlTemplateModel is the model used to render the HTML template.
	// NOTE: The fields must be exported to be used by the template.
	htmlTemplateModel struct {
		// Title is the title of the report
		Title string
		// SubTitle is the subtitle of the report
		SubTitle string
		// StatusCode is the HTTP status code of the response
		StatusCode int
		// BadgeClass is the CSS class of the status code badge
		BadgeClass string
		// LogEntries is the list of log entries
		LogEntries []LogEntry
		// WebSequenceDSL is the DSL used to render the sequence diagram
		WebSequenceDSL string
		// MetaJSON is the JSON representation of the meta data
		MetaJSON htmlTemplate.JS
	}

	// SequenceDiagramFormatter implementation of a ReportFormatter
	SequenceDiagramFormatter struct {
		// storagePath is the path where the report will be saved
		storagePath string
		// fs is the file system used to save the report
		fs fileSystem
	}
)

// ReportFormatterConfig is the configuration for a ReportFormatter
type ReportFormatterConfig struct {
	// Path is the path where the report will be saved.
	// By default, the report will be saved in the ".sequence"
	Path string
	// Kind is the kind of report to generate
	Kind ReportKind
}

// ReportKind is the kind of the report.
type ReportKind uint

const (
	// ReportKindHTML is the HTML report kind. This is the default.
	ReportKindHTML ReportKind = 0
	// ReportKindMarkdown is the Markdown report kind.
	ReportKindMarkdown ReportKind = 1
)

// SequenceDiagram produce a sequence diagram at the given path or .sequence by default.
// SequenceDiagramFormatter generate html report with sequence diagram.
// Deprecated: Use SequenceReport instead.
func SequenceDiagram(path ...string) ReportFormatter {
	var storagePath string
	if len(path) == 0 {
		storagePath = ".sequence"
	} else {
		storagePath = path[0]
	}
	return &SequenceDiagramFormatter{storagePath: storagePath, fs: &defaultFileSystem{}}
}

// SequenceReport produce a sequence diagram at the given path or .sequence by default.
// SequenceDiagramFormatter generate html report or markdown report with sequence diagram.
func SequenceReport(config ReportFormatterConfig) ReportFormatter {
	if config.Path == "" {
		config.Path = ".sequence"
	}
	if config.Kind == ReportKindMarkdown {
		return &MarkdownFormatter{storagePath: config.Path, fs: &defaultFileSystem{}}
	}
	return &SequenceDiagramFormatter{storagePath: config.Path, fs: &defaultFileSystem{}}
}

// Format formats the events received by the recorder
func (sdf *SequenceDiagramFormatter) Format(recorder *Recorder) {
	output, err := sdf.newHTMLTemplateModel(recorder)
	if err != nil {
		panic(err)
	}

	template, err := htmlTemplate.New("sequenceDiagram").
		Funcs(htmlTemplate.FuncMap{
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
		}).
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

// formatDiagramRequest formats the HTTP request into a string for logging purposes.
// It takes in a pointer to an http.Request and returns a string.
// If the URL contains a query, it appends it to the string.
// If the resulting string is longer than 65 characters, it truncates it and appends "...".
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

// badgeCSSClass returns a CSS class for a badge based on the HTTP status code.
// If the status code is between 400 and 499, it returns a warning class.
// If the status code is 500 or greater, it returns a danger class.
// Otherwise, it returns a success class.
func badgeCSSClass(status int) string {
	class := "badge badge-success"
	if status >= http.StatusBadRequest && status < http.StatusInternalServerError {
		class = "badge badge-warning"
	} else if status >= http.StatusInternalServerError {
		class = "badge badge-danger"
	}
	return class
}

// newHTMLTemplateModel returns a new htmlTemplateModel and an error.
// It iterates through the Recorder's events and creates a webSequenceDiagramDSL and logs.
// If the Content Type is an image, it generates an image and replaces the response body with the image path.
// It returns an htmlTemplateModel containing the webSequenceDiagramDSL, logs, title, subtitle, status code, badge class, and Meta data in JSON format.
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
				entry.Body = filepath.Clean(filepath.Base(imagePath(sdf.storagePath, recorder.Meta.reportFileName(), contentType, i)))
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
		WebSequenceDSL: webSequenceDiagram.String(),
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
	defer file.Close() //nolint

	_, err = file.Write([]byte(body))
	if err != nil {
		panic(err) //FIXME: error handling
	}
}

// imagePath returns the image path.
func imagePath(dir, name, contentType string, index int) string {
	return fmt.Sprintf("%s/%s_%d.%s", dir, name, index, toImageExt(contentType))
}

// imageName returns the image name.
func imageName(name, contentType string, index int) string {
	return fmt.Sprintf("%s_%d.%s", name, index, toImageExt(contentType))
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

// formatBodyContent reads the bodyReadCloser and replaces it with the replacementBody.
// It returns a string representation of the body with indentation if it is a valid JSON, otherwise it returns the original body.
// If bodyReadCloser is nil, it returns an empty string and no error.
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

// quoted returns a quoted string representation of the input string.
func quoted(in string) string {
	return fmt.Sprintf("%q", in)
}

// webSequenceDiagramDSL is the DSL used to render the sequence diagram.
type webSequenceDiagramDSL struct {
	// data is the buffer used to store the sequence diagram
	data bytes.Buffer
	// count is the number of rows in the sequence diagram
	count int
	// meta is the meta data of the sequence diagram
	meta *Meta
}

// addRequestRow adds a request row to the sequence diagram
func (r *webSequenceDiagramDSL) addRequestRow(source string, target string, description string) {
	r.addRow("->", source, target, description)
}

// addResponseRow adds a response row to the sequence diagram
func (r *webSequenceDiagramDSL) addResponseRow(source string, target string, description string) {
	r.addRow("->>", source, target, description)
}

// addRow adds a row to the sequence diagram
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

// String returns the string representation of the sequence diagram
func (r *webSequenceDiagramDSL) String() string {
	return r.data.String()
}

// MarkdownFormatter implementation of a ReportFormatter
type MarkdownFormatter struct {
	// storagePath is the path where the report will be saved
	storagePath string
	// fs is the file system used to save the report
	fs fileSystem
}

// Format formats the events received by the recorder.
// TODO: refactor this method
func (m *MarkdownFormatter) Format(recorder *Recorder) {
	if len(recorder.Events) == 0 {
		panic("no events are defined") // TODO: error handling
	}

	if err := m.fs.mkdirAll(m.storagePath, os.ModePerm); err != nil {
		panic(err) // TODO: error handling
	}

	fileName := fmt.Sprintf("%s.md", recorder.Meta.reportFileName())
	f, err := m.fs.create(filepath.Clean(filepath.Join(m.storagePath, fileName)))
	if err != nil {
		panic(err) // TODO: error handling
	}

	status, err := recorder.ResponseStatus()
	if err != nil {
		panic(err) // TODO: error handling
	}

	logs, err := m.logEntry(recorder.Events)
	if err != nil {
		panic(err) // TODO: error handling
	}
	m.generateMarkdown(f, recorder, status, logs)
}

// generateMarkdown generates a markdown report.
func (m *MarkdownFormatter) generateMarkdown(w io.Writer, recorder *Recorder, status int, logs []LogEntry) {
	markdown := m.statusBadge(md.NewMarkdown(w).H2(recorder.Title), status).LF()
	if recorder.SubTitle != "" {
		markdown = markdown.H3(recorder.SubTitle).LF()
	}
	markdown = markdown.CodeBlocks(md.SyntaxHighlightMermaid, m.mermaidSequenceDiagram(recorder)).LF()

	markdown = markdown.H2("Event log")
	for i, log := range logs {
		markdown = markdown.H4(fmt.Sprintf("Event %d", i+1)).LF()
		if log.Header != "" {
			markdown = markdown.PlainText(strings.ReplaceAll(log.Header, "\r\n", "  \r\n")).LF()
		}
		if log.Body != "" {
			contentType := extractContentType(log.Header)
			if isImage(contentType) {
				generateImage(log.Body, m.storagePath, recorder.Meta.reportFileName(), contentType, i)
				body := filepath.Clean(imageName(recorder.Meta.reportFileName(), contentType, i))
				markdown = markdown.PlainText(md.Image(body, body)).LF()
			} else if strings.Contains(contentType, "application/json") {
				markdown = markdown.CodeBlocks(md.SyntaxHighlightJSON, log.Body).LF()
			} else {
				markdown = markdown.CodeBlocks(md.SyntaxHighlightText, log.Body).LF()
			}
		}
		markdown = markdown.HorizontalRule().LF()
	}

	if err := markdown.Build(); err != nil {
		panic(err) // TODO: error handling
	}
}

func (m *MarkdownFormatter) mermaidSequenceDiagram(recorder *Recorder) string {
	seq := sequence.NewDiagram(io.Discard).AutoNumber()

	for _, event := range recorder.Events {
		switch v := event.(type) {
		case HTTPRequest:
			seq.SyncRequest(v.Source, v.Target, formatDiagramRequest(v.Value))
		case HTTPResponse:
			seq.SyncResponse(v.Source, v.Target, strconv.Itoa(v.Value.StatusCode))
		case MessageRequest:
			seq.SyncRequest(v.Source, v.Target, v.Header)
		case MessageResponse:
			seq.SyncResponse(v.Source, v.Target, v.Header)
		default:
			panic("received unknown event type") // TODO: error handling
		}
	}
	return seq.String()
}

// statusBadge returns a markdown with a status badge based on the HTTP status code.
func (m *MarkdownFormatter) statusBadge(md *md.Markdown, status int) *md.Markdown {
	switch {
	case status >= http.StatusBadRequest && status < http.StatusInternalServerError:
		return md.YellowBadgef("%d", status)
	case status >= http.StatusInternalServerError:
		return md.RedBadgef("%d", status)
	default:
		return md.GreenBadgef("%d", status)
	}
}

// logEntry returns a list of log entries based on the events.
func (m *MarkdownFormatter) logEntry(events []Event) ([]LogEntry, error) {
	var logs []LogEntry

	for _, event := range events {
		switch v := event.(type) {
		case HTTPRequest:
			httpReq := v.Value
			entry, err := NewHTTPRequestLogEntry(httpReq)
			if err != nil {
				return nil, err
			}
			entry.Timestamp = v.Timestamp
			logs = append(logs, entry)
		case HTTPResponse:
			entry, err := NewHTTPResponseLogEntry(v.Value)
			if err != nil {
				return nil, err
			}
			entry.Timestamp = v.Timestamp
			logs = append(logs, entry)
		case MessageRequest:
			logs = append(logs, LogEntry{Header: v.Header, Body: v.Body, Timestamp: v.Timestamp})
		case MessageResponse:
			logs = append(logs, LogEntry{Header: v.Header, Body: v.Body, Timestamp: v.Timestamp})
		default:
			panic("received unknown event type")
		}
	}
	return logs, nil
}
