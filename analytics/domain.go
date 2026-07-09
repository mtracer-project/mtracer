package analytics

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/mtrace-project/mtrace/test"
	"github.com/mtrace-project/mtrace/trace"
)

type TestAnalytics struct {
	TestName       string
	TraceAnalytics *TraceAnalytics
	Traces         []*trace.Trace
}

// Durations are represented in milliseconds
type TraceAnalytics struct {
	MinDuration               int64 // Minimum duration of the generated traces (in milliseconds)
	MaxDuration               int64
	P50Duration               int64 // 50th percentile duration of the generated traces (in milliseconds)
	P90Duration               int64
	P99Duration               int64
	DurationStandardDeviation float64
	AverageDuration           float64
	AverageSpanCount          float64
	AverageSpanErrorCount     float64
	ErrorRate                 float64 // Percentage of traces with at least one span with an error status
	SpanAnalytics             map[string]*SpanAnalytics
}

type SpanAnalytics struct {
	ServiceName                   string
	OperationName                 string
	MaxDuration                   int64
	MinDuration                   int64
	P50Duration                   int64
	P90Duration                   int64
	P99Duration                   int64
	AveragePosition               float64 // Average position of the span in the trace (0-based index)
	DurationStandardDeviation     float64
	AverageDuration               float64
	ErrorRate                     float64
	AveragePercentageOfTotalTrace float64 // Average percentage of the span duration over the total trace duration
}

func Build(suites []*test.TestSuite) []*TestAnalytics {
	var analyticsList []*TestAnalytics
	for _, suite := range suites {
		if suite == nil {
			continue
		}
		analyticsList = append(analyticsList, analytics(suite))
	}
	return analyticsList
}

type MetricCalculator interface {
	Calculate(data *tracesData, traceAnalytics *TraceAnalytics) error
	String() string
}

func analytics(ts *test.TestSuite) *TestAnalytics {
	traces := make([]*trace.Trace, 0, len(ts.Results))
	for _, result := range ts.Results {
		traces = append(traces, result.Trace)
	}

	traceAnalytics := &TraceAnalytics{
		SpanAnalytics: make(map[string]*SpanAnalytics),
	}
	calculators := []MetricCalculator{
		TraceDurationMetricCalculator{},
		TraceCountsMetricCalculator{},
		SpanMetricCalculator{},
	}

	for _, calculator := range calculators {
		err := calculator.Calculate(getTracesData(traces), traceAnalytics)
		if err != nil {
			slog.Warn("Error calculating metrics", "testName", ts.Name, "metric", calculator.String(), "error", err)
		}
	}

	return &TestAnalytics{
		TestName:       ts.Name,
		TraceAnalytics: traceAnalytics,
		Traces:         traces,
	}
}

type tracesData struct {
	sortedDurations    []int64
	spanCountPerTrace  []int
	errorCountPerTrace []int

	spansData map[string]*spansData // key: serviceName-operationName
}

type spansData struct {
	serviceName              string
	operationName            string
	sortedDurations          []int64
	spanPositionsInTrace     []int // positions of the span in the trace (0-based index)
	occurencies              int   // number of times the span appears in the traces
	errorCount               int   // number of times the span has an error status in the traces
	durationsWithParentTrace []*spanTraceDuration
}

type spanTraceDuration struct {
	duration      int64
	traceDuration int64
}

func getTracesData(traces []*trace.Trace) *tracesData {
	tracesData := &tracesData{
		sortedDurations:    []int64{},
		spanCountPerTrace:  []int{},
		errorCountPerTrace: []int{},
		spansData:          make(map[string]*spansData),
	}

	for _, t := range traces {
		if t == nil {
			continue
		}

		tracesData.sortedDurations = append(tracesData.sortedDurations, t.Duration.Milliseconds())
		tracesData.spanCountPerTrace = append(tracesData.spanCountPerTrace, t.SpanCount)
		tracesData.errorCountPerTrace = append(tracesData.errorCountPerTrace, t.ErrorCount)

		for i, s := range t.Spans {
			if s == nil {
				continue
			}

			key := getSpanKey(s.ServiceName, s.OperationName)
			spanData, ok := tracesData.spansData[key]
			if !ok {
				spanData = &spansData{
					serviceName:              s.ServiceName,
					operationName:            s.OperationName,
					sortedDurations:          []int64{},
					spanPositionsInTrace:     []int{},
					occurencies:              0,
					errorCount:               0,
					durationsWithParentTrace: []*spanTraceDuration{},
				}
				tracesData.spansData[key] = spanData
			}

			spanData.sortedDurations = append(spanData.sortedDurations, s.Duration.Milliseconds())
			spanData.spanPositionsInTrace = append(spanData.spanPositionsInTrace, i)
			spanData.occurencies++

			if strings.EqualFold(s.SpanStatus, "error") {
				spanData.errorCount++
			}

			spanData.durationsWithParentTrace = append(spanData.durationsWithParentTrace, &spanTraceDuration{
				duration:      s.Duration.Milliseconds(),
				traceDuration: t.Duration.Milliseconds(),
			})
		}
	}

	slices.Sort(tracesData.sortedDurations)
	for _, spanData := range tracesData.spansData {
		slices.Sort(spanData.sortedDurations)
	}

	return tracesData
}

func getSpanKey(serviceName, operationName string) string {
	service := strings.ReplaceAll(serviceName, " ", "-")
	operation := strings.ReplaceAll(operationName, " ", "-")
	return fmt.Sprintf("%s-%s", service, operation)
}
