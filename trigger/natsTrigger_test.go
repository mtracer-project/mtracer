package trigger_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/mtrace-project/mtrace/parser"
	testutils "github.com/mtrace-project/mtrace/testUtils"
	"github.com/mtrace-project/mtrace/trigger"

	nats "github.com/nats-io/nats.go"
)

func TestNATSTrigger_AuthInitialization(t *testing.T) {
	mockIDGen := &testutils.MockIdGenerator{}

	// Valid Token auth init
	tokenDto := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress": "localhost:4222",
			"subject":       "sub",
			"authType":      "token",
			"token":         "my-token",
		},
	}
	_, err := trigger.NewTrigger(tokenDto, mockIDGen, "", context.Background())
	if err != nil {
		t.Errorf("Unexpected error initializing token auth: %v", err)
	}

	// Invalid Token auth (missing token)
	badTokenDto := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress": "localhost:4222",
			"subject":       "sub",
			"authType":      "token",
		},
	}
	_, err = trigger.NewTrigger(badTokenDto, mockIDGen, "", context.Background())
	if err == nil {
		t.Error("Expected error for missing token")
	}

	// Valid JWT auth init
	jwtDto := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress": "localhost:4222",
			"subject":       "sub",
			"authType":      "jwt",
			"jwt":           "my-jwt",
			"seed":          "my-seed",
		},
	}
	_, err = trigger.NewTrigger(jwtDto, mockIDGen, "", context.Background())
	if err != nil {
		t.Errorf("Unexpected error initializing jwt auth: %v", err)
	}

	// Invalid JWT auth (missing seed)
	badJwtDto := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress": "localhost:4222",
			"subject":       "sub",
			"authType":      "jwt",
			"jwt":           "my-jwt",
		},
	}
	_, err = trigger.NewTrigger(badJwtDto, mockIDGen, "", context.Background())
	if err == nil {
		t.Error("Expected error for missing seed in jwt auth")
	}

	// Valid NKey auth init
	nkeyDto := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress": "localhost:4222",
			"subject":       "sub",
			"authType":      "nkey",
			"seed":          "my-seed",
		},
	}
	_, err = trigger.NewTrigger(nkeyDto, mockIDGen, "", context.Background())
	if err != nil {
		t.Errorf("Unexpected error initializing nkey auth: %v", err)
	}

	// Invalid NKey auth (missing seed)
	badNkeyDto := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress": "localhost:4222",
			"subject":       "sub",
			"authType":      "nkey",
		},
	}
	_, err = trigger.NewTrigger(badNkeyDto, mockIDGen, "", context.Background())
	if err == nil {
		t.Error("Expected error for missing seed in nkey auth")
	}

	// Valid mTLS auth init
	mtlsDto := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress":  "localhost:4222",
			"subject":        "sub",
			"authType":       "mtls",
			"clientCertPath": "cert.pem",
			"clientKeyPath":  "key.pem",
		},
	}
	_, err = trigger.NewTrigger(mtlsDto, mockIDGen, "", context.Background())
	if err != nil {
		t.Errorf("Unexpected error initializing mtls auth: %v", err)
	}

	// Invalid mTLS auth (missing clientKeyPath)
	badMtlsDto := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress":  "localhost:4222",
			"subject":        "sub",
			"authType":       "mtls",
			"clientCertPath": "cert.pem",
		},
	}
	_, err = trigger.NewTrigger(badMtlsDto, mockIDGen, "", context.Background())
	if err == nil {
		t.Error("Expected error for missing clientKeyPath in mtls auth")
	}
}

func TestNATSTrigger(t *testing.T) {
	// Start mock NATS server
	natsAddr := testutils.StartMockNATSServer(t, false)

	// Connect NATS client to subscribe and verify message delivery
	nc, err := nats.Connect("nats://" + natsAddr)
	if err != nil {
		t.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	subject := "test.subject"
	sub, err := nc.SubscribeSync(subject)
	if err != nil {
		t.Fatalf("Failed to subscribe to %s: %v", subject, err)
	}

	// Stub ID generator
	expectedTraceID := "9876543210fedcba9876543210fedcba"
	expectedSpanID := "9876543210fedcba"
	mockIDGen := &testutils.MockIdGenerator{
		TraceID: expectedTraceID,
		SpanID:  expectedSpanID,
	}

	dto := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress": natsAddr,
			"subject":       subject,
			"authType":      "noauth",
			"data":          `{"nats_key": "nats_value"}`,
			"headers": map[string]any{
				"Nats-Header": "Nats-Value",
			},
		},
	}

	trig, err := trigger.NewTrigger(dto, mockIDGen, "", context.Background())
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

	// Retrieve message and assert
	msg, err := sub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("Failed to receive NATS message: %v", err)
	}

	if string(msg.Data) != `{"nats_key": "nats_value"}` {
		t.Errorf("Expected message data %q, got %q", `{"nats_key": "nats_value"}`, string(msg.Data))
	}

	customHeader := msg.Header.Get("Nats-Header")
	if customHeader != "Nats-Value" {
		t.Errorf("Expected Nats-Header %q, got %q", "Nats-Value", customHeader)
	}

	traceparent := msg.Header.Get("traceparent")
	expectedTraceparent := fmt.Sprintf("00-%s-%s-01", expectedTraceID, expectedSpanID)
	if traceparent != expectedTraceparent {
		t.Errorf("Expected traceparent %q, got %q", expectedTraceparent, traceparent)
	}
}

func TestNATSTrigger_InvalidAuth(t *testing.T) {
	mockIDGen := &testutils.MockIdGenerator{}

	// 1. Missing authType
	dto1 := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress": "localhost:4222",
			"subject":       "sub",
		},
	}
	if _, err := trigger.NewTrigger(dto1, mockIDGen, "", context.Background()); err == nil {
		t.Error("Expected error for missing authType")
	}

	// 2. Unsupported authType
	dtoUnsupported := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress": "localhost:4222",
			"subject":       "sub",
			"authType":      "badauth",
		},
	}
	if _, err := trigger.NewTrigger(dtoUnsupported, mockIDGen, "", context.Background()); err == nil {
		t.Error("Expected error for unsupported authType")
	}

	// 3. Userpass: Missing username
	dtoUserpassNoUser := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress": "localhost:4222",
			"subject":       "sub",
			"authType":      "userpass",
			"password":      "pass",
		},
	}
	if _, err := trigger.NewTrigger(dtoUserpassNoUser, mockIDGen, "", context.Background()); err == nil {
		t.Error("Expected error for missing username in userpass auth")
	}

	// 4. Userpass: Missing password
	dtoUserpassNoPass := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress": "localhost:4222",
			"subject":       "sub",
			"authType":      "userpass",
			"username":      "user",
		},
	}
	if _, err := trigger.NewTrigger(dtoUserpassNoPass, mockIDGen, "", context.Background()); err == nil {
		t.Error("Expected error for missing password in userpass auth")
	}

	// 5. Token: Missing token
	dtoTokenNoToken := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress": "localhost:4222",
			"subject":       "sub",
			"authType":      "token",
		},
	}
	if _, err := trigger.NewTrigger(dtoTokenNoToken, mockIDGen, "", context.Background()); err == nil {
		t.Error("Expected error for missing token in token auth")
	}

	// 6. JWT: Missing jwt
	dtoJWTNoJWT := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress": "localhost:4222",
			"subject":       "sub",
			"authType":      "jwt",
			"seed":          "my-seed",
		},
	}
	if _, err := trigger.NewTrigger(dtoJWTNoJWT, mockIDGen, "", context.Background()); err == nil {
		t.Error("Expected error for missing jwt in jwt auth")
	}

	// 7. JWT: Missing seed
	dtoJWTNoSeed := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress": "localhost:4222",
			"subject":       "sub",
			"authType":      "jwt",
			"jwt":           "my-jwt",
		},
	}
	if _, err := trigger.NewTrigger(dtoJWTNoSeed, mockIDGen, "", context.Background()); err == nil {
		t.Error("Expected error for missing seed in jwt auth")
	}

	// 8. NKey: Missing seed
	dtoNKeyNoSeed := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress": "localhost:4222",
			"subject":       "sub",
			"authType":      "nkey",
		},
	}
	if _, err := trigger.NewTrigger(dtoNKeyNoSeed, mockIDGen, "", context.Background()); err == nil {
		t.Error("Expected error for missing seed in nkey auth")
	}

	// 9. mTLS: Missing clientCertPath
	dtoMTLSNoCert := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress": "localhost:4222",
			"subject":       "sub",
			"authType":      "mtls",
			"clientKeyPath": "key.pem",
		},
	}
	if _, err := trigger.NewTrigger(dtoMTLSNoCert, mockIDGen, "", context.Background()); err == nil {
		t.Error("Expected error for missing clientCertPath in mtls auth")
	}

	// 10. mTLS: Missing clientKeyPath
	dtoMTLSNoKey := &parser.TriggerDTO{
		Type: "nats",
		Args: map[string]any{
			"serverAddress":  "localhost:4222",
			"subject":        "sub",
			"authType":       "mtls",
			"clientCertPath": "cert.pem",
		},
	}
	if _, err := trigger.NewTrigger(dtoMTLSNoKey, mockIDGen, "", context.Background()); err == nil {
		t.Error("Expected error for missing clientKeyPath in mtls auth")
	}
}
