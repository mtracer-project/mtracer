package trigger_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mtracer-project/mtracer/parser"
	testutils "github.com/mtracer-project/mtracer/testUtils"
	"github.com/mtracer-project/mtracer/trigger"

	nats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func TestJetstreamTrigger(t *testing.T) {
	// Start mock NATS server with JetStream
	natsAddr := testutils.StartMockNATSServer(t, true)

	// Connect NATS client to create stream and consumer
	nc, err := nats.Connect("nats://" + natsAddr)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	js, err := jetstream.New(nc)
	if err != nil {
		t.Fatalf("Failed to create JetStream context: %v", err)
	}

	subject := "test.jetstream"
	streamName := "TEST_STREAM"

	// Create stream matching the subject
	_, err = js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     streamName,
		Subjects: []string{subject},
	})
	if err != nil {
		t.Fatalf("Failed to create JetStream stream: %v", err)
	}

	// Create consumer to fetch published messages
	cons, err := js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Durable: "test-consumer",
	})
	if err != nil {
		t.Fatalf("Failed to create JetStream consumer: %v", err)
	}

	// Stub ID generator
	expectedTraceID := "11112222333344445555666677778888"
	expectedSpanID := "1111222233334444"
	mockIDGen := &testutils.MockIdGenerator{
		TraceID: expectedTraceID,
		SpanID:  expectedSpanID,
	}

	dto := &parser.TriggerDTO{
		Type: "jetstream",
		Args: map[string]any{
			"serverAddress": natsAddr,
			"subject":       subject,
			"authType":      "noauth",
			"data":          `{"js_key": "js_value"}`,
		},
	}

	trig, err := trigger.NewTrigger(dto, mockIDGen, "", ctx)
	if err != nil {
		t.Fatalf("Failed to create trigger: %v", err)
	}

	traceID, err := trig.Trigger()
	if err != nil {
		t.Fatalf("Trigger failed: %v", err)
	}

	if traceID.String() != expectedTraceID {
		t.Errorf("Expected trace ID %q, got %q", expectedTraceID, traceID.String())
	}

	// Fetch message and assert
	msgs, err := cons.Fetch(1)
	if err != nil {
		t.Fatalf("Failed to fetch JetStream messages: %v", err)
	}

	select {
	case msg := <-msgs.Messages():
		if string(msg.Data()) != `{"js_key": "js_value"}` {
			t.Errorf("Expected message data %q, got %q", `{"js_key": "js_value"}`, string(msg.Data()))
		}

		traceparent := msg.Headers().Get("traceparent")
		expectedTraceparent := fmt.Sprintf("00-%s-%s-01", expectedTraceID, expectedSpanID)
		if traceparent != expectedTraceparent {
			t.Errorf("Expected traceparent %q, got %q", expectedTraceparent, traceparent)
		}

		_ = msg.Ack()
	case <-ctx.Done():
		t.Fatal("Timeout waiting for JetStream message")
	}
}
