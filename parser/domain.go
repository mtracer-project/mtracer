package parser

import (
	"github.com/mtracer-project/mtracer/domain"
)

type TestDTO struct {
	Name          string             `yaml:"name"`
	Description   string             `yaml:"description"`
	SetupCommands []*SetupCommandDTO `yaml:"setupCommands"`
	Trigger       *TriggerDTO        `yaml:"trigger"`

	WaitBeforeFetch *domain.Duration `yaml:"waitBeforeFetch"`
	Timeout         *domain.Duration `yaml:"timeout"`
	RetryDelay      *domain.Duration `yaml:"retryDelay"`
	LastSpan        *ExpectedSpanDTO `yaml:"lastSpan"`

	ExpectedTraces     []*ExpectedTraceDTO         `yaml:"expectedTraces"`
	ExpectedProperties *ExpectedTracePropertiesDTO `yaml:"expectedProperties"`

	Assertions []*AssertionDTO `yaml:"assertions"`

	PostExecChecks []*PostExecCheckDTO `yaml:"postExecChecks"`
	FilePath       string              `yaml:"-"`
}

type ExpectedTracePropertiesDTO struct {
	MaxDuration *domain.Duration `yaml:"maxDuration"`
	MinDuration *domain.Duration `yaml:"minDuration"`
	SpanCount   *int             `yaml:"spanCount"`
	ErrorCount  *int             `yaml:"errorCount"`
}

type SetupCommandDTO struct {
	Type       string             `yaml:"type"`
	Cmd        string             `yaml:"cmd"`
	Args       map[string]any     `yaml:"args"`
	CleanupCmd *CleanupCommandDTO `yaml:"cleanupCmd"`
}

type CleanupCommandDTO struct {
	Cmd  string         `yaml:"cmd"`
	Args map[string]any `yaml:"args"`
}

type TriggerDTO struct {
	Type string         `yaml:"type"`
	Args map[string]any `yaml:"args"`
}

type ExpectedTraceDTO struct {
	Spans   []*ExpectedSpanDTO `yaml:"spans"`
	Ordered *bool              `yaml:"ordered"`
	Checker *string            `yaml:"checker"`
}

type ExpectedSpanDTO struct {
	SpanDTO     `yaml:",inline"`
	MaxDuration *domain.Duration `yaml:"maxDuration"`
	MinDuration *domain.Duration `yaml:"minDuration"`
}

type AssertionDTO struct {
	Name    string         `yaml:"name"`
	Type    string         `yaml:"type"`
	Queries map[string]any `yaml:"queries"`
}

type TraceDTO struct {
	Spans []*SpanDTO `yaml:"spans"`
}

type SpanDTO struct {
	ServiceName   string  `yaml:"serviceName"`
	OperationName *string `yaml:"operationName"`
	SpanKind      *string `yaml:"spanKind"`
	SpanStatus    *string `yaml:"spanStatus"`
}

type PostExecCheckDTO struct {
	Name string         `yaml:"name"`
	Type string         `yaml:"type"`
	Args map[string]any `yaml:"args"`
}
