package trigger

import (
	"context"
	"fmt"
	"log/slog"

	idgenerator "github.com/mtracer-project/mtracer/idGenerator"
	"github.com/mtracer-project/mtracer/parser"

	nats "github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type JetstreamTrigger struct {
	NATSTrigger
	ctx context.Context
}

func (j *JetstreamTrigger) Init(dto *parser.TriggerDTO, idGenerator idgenerator.IdGenerator, baseDir string, ctx context.Context) error {
	err := j.NATSTrigger.Init(dto, idGenerator, baseDir, ctx)
	if err != nil {
		return err
	}

	j.ctx = ctx
	return nil
}

func (j *JetstreamTrigger) Trigger() (TraceId, error) {
	traceId, err := j.idGenerator.Generate(idgenerator.TRACE_ID_LENGTH)
	if err != nil {
		return "", fmt.Errorf("error while generating trace ID: %w", err)
	}

	spanId, err := j.idGenerator.Generate(idgenerator.SPAN_ID_LENGTH)
	if err != nil {
		return "", fmt.Errorf("error while generating span ID: %w", err)
	}

	nc, err := j.natsConn.Connect(j.serverAddress, j.caPemPath)
	if err != nil {
		return "", fmt.Errorf("error while connecting to NATS: %w", err)
	}
	defer nc.Close() //nolint:errcheck

	js, err := jetstream.New(nc)
	if err != nil {
		return "", fmt.Errorf("error while initializing JetStream: %w", err)
	}

	msg := nats.NewMsg(j.subject)
	msg.Data = j.data
	if msg.Header == nil {
		msg.Header = nats.Header{}
	}

	for key, values := range j.headers {
		for _, value := range values {
			msg.Header.Add(key, value)
		}
	}

	traceparent := getTraceparent(traceId, spanId)
	msg.Header.Set("traceparent", traceparent)

	slog.Info("Jetstream Trigger", "subject", j.subject, "headers", msg.Header, "traceparent", traceparent, "data", string(j.data))

	ack, err := js.PublishMsg(j.ctx, msg)
	if err != nil {
		return "", fmt.Errorf("error while publishing Jetstream message: %w", err)
	}

	ackMsg := fmt.Sprintf("Jetstream message published with sequence: %d", ack.Sequence)

	slog.Info("Jetstream Trigger", "ackMsg", ackMsg)

	traceIdObj, err := NewTraceId(traceId)
	if err != nil {
		return "", fmt.Errorf("error while creating TraceId object: %w", err)
	}

	return traceIdObj, nil
}

func (t *JetstreamTrigger) Example() string {
	return `trigger:
  type: "jetstream"
  args:
    serverAddress: "localhost:4222"
    subject: "test.subject"
    headers:
      - Content-Type: "application/json"
    data: '{"key": "value"}'
    caPemPath: "path/to/ca.pem"
    authType: "userpass" # noauth | userpass | token | jwt | nkey | mtls
    username: "nats-user" # required if authType is userpass
    password: "nats-password" # required if authType is userpass`
}
