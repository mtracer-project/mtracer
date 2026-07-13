package span_test

import (
	"strings"
	"testing"
	"time"

	"github.com/mtracer-project/mtracer/domain"
	"github.com/mtracer-project/mtracer/parser"
	"github.com/mtracer-project/mtracer/span"
)

func TestNewSpan(t *testing.T) {
	spanId := "span-123"
	parentId := "parent-456"
	serviceName := "my-service"
	operationName := "my-operation"
	spanKind := "server"
	spanStatus := "ok"
	startTime := time.Now()
	endTime := startTime.Add(100 * time.Millisecond)
	duration := 100 * time.Millisecond

	s := span.NewSpan(
		&spanId,
		&parentId,
		&serviceName,
		&operationName,
		&spanKind,
		&spanStatus,
		&startTime,
		&endTime,
		&duration,
		nil,
	)

	if s.SpanId != spanId {
		t.Errorf("Expected SpanId %q, got %q", spanId, s.SpanId)
	}
	if s.ParentId != parentId {
		t.Errorf("Expected ParentId %q, got %q", parentId, s.ParentId)
	}
	if s.ServiceName != serviceName {
		t.Errorf("Expected ServiceName %q, got %q", serviceName, s.ServiceName)
	}
	if s.OperationName != operationName {
		t.Errorf("Expected OperationName %q, got %q", operationName, s.OperationName)
	}
	if s.SpanKind != spanKind {
		t.Errorf("Expected SpanKind %q, got %q", spanKind, s.SpanKind)
	}
	if s.SpanStatus != spanStatus {
		t.Errorf("Expected SpanStatus %q, got %q", spanStatus, s.SpanStatus)
	}
	if !s.StartTime.Equal(startTime) {
		t.Errorf("Expected StartTime %v, got %v", startTime, s.StartTime)
	}
	if !s.EndTime.Equal(endTime) {
		t.Errorf("Expected EndTime %v, got %v", endTime, s.EndTime)
	}
	if s.Duration != duration {
		t.Errorf("Expected Duration %v, got %v", duration, s.Duration)
	}
	if s.Attributes != nil {
		t.Errorf("Expected Attributes nil, got %v", s.Attributes)
	}
}

func TestNewSpan_WithAttributes(t *testing.T) {
	spanId := "span-attr"
	parentId := "parent-attr"
	serviceName := "attr-service"
	operationName := "attr-op"
	spanKind := "client"
	spanStatus := "ok"
	startTime := time.Now()
	endTime := startTime.Add(50 * time.Millisecond)
	duration := 50 * time.Millisecond

	attributes := map[string]any{
		"http.method":      "GET",
		"http.status_code": 200,
		"consumer":         true,
	}

	s := span.NewSpan(
		&spanId,
		&parentId,
		&serviceName,
		&operationName,
		&spanKind,
		&spanStatus,
		&startTime,
		&endTime,
		&duration,
		attributes,
	)

	if len(s.Attributes) != 3 {
		t.Fatalf("Expected 3 attributes, got %d", len(s.Attributes))
	}
	if s.Attributes["http.method"] != "GET" {
		t.Errorf("Expected http.method 'GET', got %v", s.Attributes["http.method"])
	}
	if s.Attributes["http.status_code"] != 200 {
		t.Errorf("Expected http.status_code 200, got %v", s.Attributes["http.status_code"])
	}
	if s.Attributes["consumer"] != true {
		t.Errorf("Expected consumer true, got %v", s.Attributes["consumer"])
	}
}

func TestSpan_String_IncludesAttributes(t *testing.T) {
	s := &span.Span{
		SpanId:      "s1",
		ServiceName: "svc",
		Attributes: map[string]any{
			"key1": "val1",
		},
	}

	str := s.String()
	if !strings.Contains(str, "Attributes:") {
		t.Errorf("Expected String() to contain 'Attributes:', got %q", str)
	}
	if !strings.Contains(str, "key1") {
		t.Errorf("Expected String() to contain attribute key 'key1', got %q", str)
	}
}

func TestNewExpectedSpan_Nil(t *testing.T) {
	s := span.NewExpectedSpan(nil)
	if s != nil {
		t.Errorf("Expected NewExpectedSpan(nil) to return nil, got %v", s)
	}
}

func TestNewExpectedSpan_NonNil(t *testing.T) {
	op := "operation"
	kind := "client"
	status := "error"
	maxDur := domain.Duration(200 * time.Millisecond)
	minDur := domain.Duration(50 * time.Millisecond)

	dto := &parser.ExpectedSpanDTO{
		SpanDTO: parser.SpanDTO{
			ServiceName:   "test-service",
			OperationName: &op,
			SpanKind:      &kind,
			SpanStatus:    &status,
		},
		MaxDuration: &maxDur,
		MinDuration: &minDur,
	}

	s := span.NewExpectedSpan(dto)
	if s == nil {
		t.Fatal("Expected NewExpectedSpan to return a valid struct, got nil")
	}

	// Wait, maxDuration and minDuration are unexported, but we can verify it by using s.Equal or reflection,
	// or we can test it directly since we are in span_test. Wait, span_test cannot read unexported fields!
	// But TestSpanEqual_DurationMatching in comparator_test.go already tests duration matching,
	// which implicitly verifies that they are set correctly.
	// Let's just check the exported fields:
	if s.ServiceName != "test-service" {
		t.Errorf("Expected ServiceName %q, got %q", "test-service", s.ServiceName)
	}
	if s.OperationName == nil || *s.OperationName != op {
		t.Errorf("Expected OperationName %q, got %v", op, s.OperationName)
	}
	if s.SpanKind == nil || *s.SpanKind != kind {
		t.Errorf("Expected SpanKind %q, got %v", kind, s.SpanKind)
	}
	if s.SpanStatus == nil || *s.SpanStatus != status {
		t.Errorf("Expected SpanStatus %q, got %v", status, s.SpanStatus)
	}
}

func TestSpanToProto(t *testing.T) {
	// Case 1: nil span
	var nilSpan *span.Span
	if nilSpan.ToProto() != nil {
		t.Error("Expected nilSpan.ToProto() to be nil")
	}

	// Case 2: valid span
	startTime := time.Unix(100, 200).UTC()
	endTime := time.Unix(101, 300).UTC()
	duration := 1100 * time.Millisecond

	s := &span.Span{
		SpanId:        "s123",
		ParentId:      "p456",
		ServiceName:   "service",
		OperationName: "operation",
		SpanKind:      "client",
		SpanStatus:    "error",
		StartTime:     startTime,
		EndTime:       endTime,
		Duration:      duration,
	}

	p := s.ToProto()
	if p == nil {
		t.Fatal("Expected non-nil proto span")
	}

	if p.SpanId != "s123" {
		t.Errorf("Expected SpanId %q, got %q", "s123", p.SpanId)
	}
	if p.ParentId != "p456" {
		t.Errorf("Expected ParentId %q, got %q", "p456", p.ParentId)
	}
	if p.ServiceName != "service" {
		t.Errorf("Expected ServiceName %q, got %q", "service", p.ServiceName)
	}
	if p.OperationName != "operation" {
		t.Errorf("Expected OperationName %q, got %q", "operation", p.OperationName)
	}
	if p.SpanKind != "client" {
		t.Errorf("Expected SpanKind %q, got %q", "client", p.SpanKind)
	}
	if p.SpanStatus != "error" {
		t.Errorf("Expected SpanStatus %q, got %q", "error", p.SpanStatus)
	}
	if p.StartTime.Seconds != 100 || p.StartTime.Nanos != 200 {
		t.Errorf("Expected StartTime seconds=100, nanos=200, got %d, %d", p.StartTime.Seconds, p.StartTime.Nanos)
	}
	if p.EndTime.Seconds != 101 || p.EndTime.Nanos != 300 {
		t.Errorf("Expected EndTime seconds=101, nanos=300, got %d, %d", p.EndTime.Seconds, p.EndTime.Nanos)
	}
	if p.Duration.Seconds != 1 || p.Duration.Nanos != 100000000 {
		t.Errorf("Expected Duration seconds=1, nanos=100000000, got %d, %d", p.Duration.Seconds, p.Duration.Nanos)
	}
	if p.Attributes != nil {
		t.Errorf("Expected nil Attributes for span without attributes, got %v", p.Attributes)
	}
}

func TestSpanToProto_WithAttributes(t *testing.T) {
	s := &span.Span{
		SpanId:        "s-attr",
		ParentId:      "p-attr",
		ServiceName:   "service",
		OperationName: "operation",
		SpanKind:      "server",
		SpanStatus:    "ok",
		StartTime:     time.Unix(100, 0).UTC(),
		EndTime:       time.Unix(101, 0).UTC(),
		Duration:      time.Second,
		Attributes: map[string]any{
			"http.method":      "POST",
			"http.status_code": float64(201),
			"is.error":         false,
		},
	}

	p := s.ToProto()
	if p == nil {
		t.Fatal("Expected non-nil proto span")
	}

	if len(p.Attributes) != 3 {
		t.Fatalf("Expected 3 proto attributes, got %d", len(p.Attributes))
	}

	method := p.Attributes["http.method"]
	if method == nil || method.GetStringValue() != "POST" {
		t.Errorf("Expected http.method 'POST', got %v", method)
	}

	statusCode := p.Attributes["http.status_code"]
	if statusCode == nil || statusCode.GetNumberValue() != 201 {
		t.Errorf("Expected http.status_code 201, got %v", statusCode)
	}

	isError := p.Attributes["is.error"]
	if isError == nil || isError.GetBoolValue() != false {
		t.Errorf("Expected is.error false, got %v", isError)
	}
}

func TestSpanToProto_EmptyAttributes(t *testing.T) {
	s := &span.Span{
		SpanId:     "s-empty",
		StartTime:  time.Unix(100, 0).UTC(),
		EndTime:    time.Unix(101, 0).UTC(),
		Duration:   time.Second,
		Attributes: map[string]any{},
	}

	p := s.ToProto()
	if p == nil {
		t.Fatal("Expected non-nil proto span")
	}

	// Empty map should not produce proto attributes (len(s.Attributes) > 0 check in ToProto)
	if len(p.Attributes) != 0 {
		t.Errorf("Expected 0 proto attributes for empty map, got %d", len(p.Attributes))
	}
}
