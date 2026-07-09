package trigger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	idgenerator "github.com/mtrace-project/mtrace/idGenerator"
	"github.com/mtrace-project/mtrace/parser"
)

const (
	DEFAULT_HTTP_METHOD = "GET"
	DEFAULT_TIMEOUT     = 10 * time.Second
)

type HTTPTrigger struct {
	url     string
	headers map[string][]string
	method  string
	body    *string

	idGenerator idgenerator.IdGenerator
	ctx         context.Context
}

func (t *HTTPTrigger) Init(dto *parser.TriggerDTO, idGenerator idgenerator.IdGenerator, baseDir string, ctx context.Context) error {
	if dto.Args == nil {
		return fmt.Errorf("invalid trigger arguments")
	}

	url, ok := dto.Args["url"].(string)
	if !ok {
		return fmt.Errorf("url argument is required and must be a string")
	}

	method, ok := dto.Args["method"].(string)
	if !ok {
		method = DEFAULT_HTTP_METHOD
	}

	headers := make(map[string][]string)
	if hdrs, ok := dto.Args["headers"].(map[string]any); ok {
		for k, v := range hdrs {
			if valStr, ok := v.(string); ok {
				headers[k] = append(headers[k], valStr)
			}
		}
	}

	var body *string
	if b, ok := dto.Args["body"].(string); ok {
		body = &b
	}

	t.url = url
	t.method = method
	t.headers = headers
	t.body = body
	t.idGenerator = idGenerator
	t.ctx = ctx
	return nil
}

func (t *HTTPTrigger) Trigger() (TraceId, error) {
	traceId, err := t.idGenerator.Generate(idgenerator.TRACE_ID_LENGTH)
	if err != nil {
		return "", fmt.Errorf("error while generating trace ID: %w", err)
	}
	spanId, err := t.idGenerator.Generate(idgenerator.SPAN_ID_LENGTH)
	if err != nil {
		return "", fmt.Errorf("error while generating span ID: %w", err)
	}

	response, err := sendHTTPRequest(traceId, spanId, t.method, t.url, t.body, t.headers, t.ctx)
	if err != nil {
		return "", fmt.Errorf("error while creating the HTTP request: %w", err)
	}

	slog.Info("HTTP Trigger Response", "response", response)

	traceIdObj, err := NewTraceId(traceId)
	if err != nil {
		return "", fmt.Errorf("error while creating TraceId object: %w", err)
	}

	return traceIdObj, nil
}

func sendHTTPRequest(traceId string, spanId string, method string, url string, body *string, headers map[string][]string, ctx context.Context) (string, error) {
	var bodyReader io.Reader
	if body != nil && *body != "" && canHaveBody(method) {
		bodyReader = bytes.NewBufferString(*body)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return "", fmt.Errorf("error while creating the HTTP request: %w", err)
	}

	for key, value := range headers {
		for _, v := range value {
			req.Header.Add(key, v)
		}
	}

	req.Header.Add("traceparent", getTraceparent(traceId, spanId))

	client := &http.Client{
		Timeout: DEFAULT_TIMEOUT,
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error while sending the HTTP request: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error while reading the response: %w", err)
	}

	result := fmt.Sprintf("Status: %s\nBody:\n%s", resp.Status, string(respBody))
	return result, nil
}

func canHaveBody(method string) bool {
	switch strings.ToLower(method) {
	case "post", "put", "patch", "delete":
		return true
	default:
		return false
	}
}

func (t *HTTPTrigger) Example() string {
	return `trigger:
  type: "http"
  args:
    url: "http://example.com/api/endpoint"
    method: "POST"
    headers:
      - Content-Type: "application/json"
      - Authorization: "Bearer <token>"
    body: '{"key": "value"}'`
}
