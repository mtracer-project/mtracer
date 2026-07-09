package jaeger

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mtrace-project/mtrace/trigger"
)

const (
	JAEGER_SEARCH_ENDPOINT = "%s/api/traces/%s"
	JAEGER_DEFAULT_TIMEOUT = 30 * time.Second
)

type IJaegerTraceRepository interface {
	Get(traceId trigger.TraceId) (*JaegerTraceDTO, error)
}

type JaegerTraceRepository struct {
	BaseURL string
	Client  *http.Client
	ctx     context.Context
}

type JaegerTraceResponse struct {
	Data   []JaegerTraceDTO `json:"data"`
	Total  int              `json:"total"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
	Errors any              `json:"errors"`
}

type JaegerTraceDTO struct {
	TraceId   string                   `json:"traceID"`
	Spans     []JaegerSpanDTO          `json:"spans"`
	Processes map[string]JaegerProcess `json:"processes"`
	Warnings  []string                 `json:"warnings"`
}

type JaegerSpanDTO struct {
	TraceId       string            `json:"traceID"`
	SpanId        string            `json:"spanID"`
	OperationName string            `json:"operationName"`
	References    []JaegerReference `json:"references"`
	StartTimeUs   int64             `json:"startTime"`
	DurationUs    int64             `json:"duration"`
	Tags          []JaegerTag       `json:"tags"`
	Logs          []any             `json:"logs"`
	ProcessID     string            `json:"processID"`
	Warnings      []string          `json:"warnings"`
}

type JaegerReference struct {
	RefType string `json:"refType"`
	TraceId string `json:"traceID"`
	SpanId  string `json:"spanID"`
}

type JaegerProcess struct {
	ServiceName string      `json:"serviceName"`
	Tags        []JaegerTag `json:"tags"`
}

type JaegerTag struct {
	Key   string `json:"key"`
	Type  string `json:"type"`
	Value any    `json:"value"`
}

func NewJaegerTraceRepository(config *JaegerConfig, ctx context.Context) *JaegerTraceRepository {
	return &JaegerTraceRepository{
		BaseURL: strings.TrimSuffix(config.BaseURL, "/"),
		Client: &http.Client{
			Timeout: JAEGER_DEFAULT_TIMEOUT,
		},
		ctx: ctx,
	}
}

func (r *JaegerTraceRepository) Get(traceId trigger.TraceId) (*JaegerTraceDTO, error) {
	endpoint := fmt.Sprintf(JAEGER_SEARCH_ENDPOINT, r.BaseURL, traceId.String())
	req, err := http.NewRequestWithContext(r.ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create Jaeger request: %w", err)
	}

	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call Jaeger: %w", err)
	}
	defer resp.Body.Close() // nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("jaeger returned %s: %s", resp.Status, strings.TrimSpace(string(payload)))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read Jaeger response body: %w", err)
	}

	var jaegerResp JaegerTraceResponse
	if err := json.Unmarshal(body, &jaegerResp); err != nil {
		return nil, fmt.Errorf("decode Jaeger response: %w", err)
	}

	if len(jaegerResp.Data) == 0 {
		return nil, fmt.Errorf("trace %s not found in Jaeger", traceId.String())
	}

	return &jaegerResp.Data[0], nil
}
