package openobserve

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mtracer-project/mtracer/domain"
	"github.com/mtracer-project/mtracer/span"
	"github.com/mtracer-project/mtracer/trigger"

	"github.com/lib/pq"
)

const (
	OPEN_OBSERVE_SEARCH_ENDPOINT = "%s/api/%s/_search?type=traces&search_type=ui&use_cache=false"
	START_TIME_BUFFER            = 20 * time.Minute
	HTTP_CLIENT_TIMEOUT          = 30 * time.Second
)

type IOpenObserveTraceRepository interface {
	Get(traceId trigger.TraceId) (*OpenObserveTraceResponse, error)
}

type OpenObserveTraceRepository struct {
	BaseURL      string
	Organization string
	StreamName   string
	Username     string
	Password     string
	Client       *http.Client

	Timeout    time.Duration
	RetryDelay time.Duration
	LastSpan   *span.ExpectedSpan

	ctx context.Context
}

type OpenObserveTraceResponse struct {
	TraceId     trigger.TraceId
	StartTimeNs int64
	EndTimeNs   int64
	DurationNs  int64
	SpanCount   int
	ErrorCount  int
	Spans       []*OpenObserveSpanDTO
}

type OpenObserveSpanDTO struct {
	SpanId        string
	ParentId      string
	ServiceName   string
	OperationName *string
	SpanKind      *string
	SpanStatus    *string
	StartTimeNs   int64
	EndTimeNs     int64
	DurationNs    int64
	Attributes    map[string]any
}

type openObserveSearchRequest struct {
	Query openObserveQuery `json:"query"`
}

type openObserveQuery struct {
	SQL         string `json:"sql"`
	StartTimeNs int64  `json:"start_time,omitempty"`
	EndTimeNs   int64  `json:"end_time,omitempty"`
	From        int    `json:"from,omitempty"`
	Size        int    `json:"size,omitempty"`
}

type openObserveSearchResponse struct {
	Hits []map[string]any `json:"hits"`
}

func NewOpenObserveTraceRepository(config *OpenObserveConfig, ctx context.Context) *OpenObserveTraceRepository {
	httpClient := &http.Client{
		Timeout: HTTP_CLIENT_TIMEOUT,
	}

	return &OpenObserveTraceRepository{
		BaseURL:      strings.TrimSuffix(config.BaseURL, "/"),
		Organization: config.OrgName,
		StreamName:   config.StreamName,
		Username:     config.Username,
		Password:     config.Password,
		Client:       httpClient,
		ctx:          ctx,
	}
}

func (o *OpenObserveTraceRepository) Get(traceId trigger.TraceId) (*OpenObserveTraceResponse, error) {
	req, err := o.prepareHTTPRequest(traceId)
	if err != nil {
		return nil, err
	}
	resp, err := o.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call OpenObserve: %w", err)
	}

	defer resp.Body.Close() // nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		payload, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("OpenObserve returned %s: %s", resp.Status, strings.TrimSpace(string(payload)))
	}

	var searchResponse openObserveSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("decode OpenObserve response: %w", err)
	}

	spans := make([]*OpenObserveSpanDTO, 0, len(searchResponse.Hits))
	for _, hit := range searchResponse.Hits {
		spans = append(spans, &OpenObserveSpanDTO{
			SpanId:        stringValue(hit["span_id"]),
			ParentId:      stringValue(hit["reference_parent_span_id"]),
			ServiceName:   stringValue(hit["service_name"]),
			OperationName: optionalString(hit["operation_name"]),
			SpanKind:      optionalSpanKind(hit["span_kind"]),
			SpanStatus:    optionalString(hit["span_status"]),
			StartTimeNs:   int64Value(hit["start_time"]),
			EndTimeNs:     int64Value(hit["end_time"]),
			DurationNs:    int64Value(hit["duration"]),
			Attributes:    hit,
		})
	}

	return buildOpenObserveTraceResponse(spans, traceId)
}

func (o *OpenObserveTraceRepository) prepareHTTPRequest(traceId trigger.TraceId) (*http.Request, error) {
	if strings.TrimSpace(traceId.String()) == "" {
		return nil, fmt.Errorf("trace id is required")
	}

	query := fmt.Sprintf(
		`SELECT * FROM %s WHERE trace_id = %s ORDER BY start_time ASC`,
		pq.QuoteIdentifier(o.StreamName),
		pq.QuoteLiteral(traceId.String()),
	)

	requestBody, err := json.Marshal(openObserveSearchRequest{
		Query: openObserveQuery{
			SQL:         query,
			StartTimeNs: time.Now().Add(-START_TIME_BUFFER).UnixMicro(),
			EndTimeNs:   time.Now().UnixMicro(),
			From:        0,
			Size:        1000,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("marshal OpenObserve request: %w", err)
	}

	endpoint := fmt.Sprintf(OPEN_OBSERVE_SEARCH_ENDPOINT, o.BaseURL, o.Organization)

	req, err := http.NewRequestWithContext(o.ctx, http.MethodPost, endpoint, bytes.NewReader(requestBody))
	if err != nil {
		return nil, fmt.Errorf("create OpenObserve request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(o.Username, o.Password)

	return req, nil
}

func buildOpenObserveTraceResponse(spans []*OpenObserveSpanDTO, traceId trigger.TraceId) (*OpenObserveTraceResponse, error) {
	if len(spans) == 0 {
		return nil, fmt.Errorf("trace %s not found in OpenObserve", traceId.String())
	}

	first := spans[0]
	traceStart := first.StartTimeNs
	traceEnd := first.EndTimeNs
	errorCount := 0

	for _, currentSpan := range spans {
		if currentSpan.EndTimeNs > traceEnd {
			traceEnd = currentSpan.EndTimeNs
		}
		if currentSpan.SpanStatus != nil && *currentSpan.SpanStatus == "error" {
			errorCount++
		}
	}

	return &OpenObserveTraceResponse{
		TraceId:     traceId,
		StartTimeNs: traceStart,
		EndTimeNs:   traceEnd,
		DurationNs:  traceEnd - traceStart,
		SpanCount:   len(spans),
		ErrorCount:  errorCount,
		Spans:       spans,
	}, nil
}

func stringValue(value any) string {
	str, ok := value.(string)
	if !ok {
		return ""
	}
	return str
}

func optionalString(value any) *string {
	result := stringValue(value)
	if result == "" {
		return nil
	}
	return &result
}

func optionalSpanKind(value any) *string {
	valueStr := optionalString(value)
	if valueStr == nil {
		return nil
	}
	result := domain.SpanKindValue(*valueStr)
	return &result
}

func int64Value(value any) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case float64:
		return int64(v)
	case int:
		return int64(v)
	default:
		return 0
	}
}
