package idgenerator_test

import (
	"regexp"
	"testing"

	idgenerator "github.com/mtrace-project/mtrace/idGenerator"
)

func TestIdGeneratorV1Generate(t *testing.T) {
	generator := &idgenerator.IdGeneratorV1{}
	traceID, err := generator.Generate(idgenerator.TRACE_ID_LENGTH)
	if err != nil {
		t.Fatalf("failed to generate trace ID: %v", err)
	}

	if len(traceID) != idgenerator.TRACE_ID_LENGTH {
		t.Fatalf("expected ID length %d, got %d: %q", idgenerator.TRACE_ID_LENGTH, len(traceID), traceID)
	}

	if matched := regexp.MustCompile(`^[0-9a-f]{32}$`).MatchString(traceID); !matched {
		t.Fatalf("expected lowercase hex trace ID, got %q", traceID)
	}

	if traceID == "00000000000000000000000000000000" {
		t.Fatalf("expected trace ID to be non-zero")
	}
}

func TestIdGeneratorV1GenerateInvalidLength(t *testing.T) {
	generator := &idgenerator.IdGeneratorV1{}

	_, err := generator.Generate(0)
	if err == nil {
		t.Fatal("expected error for length 0, got nil")
	}

	_, err = generator.Generate(-1)
	if err == nil {
		t.Fatal("expected error for negative length, got nil")
	}

	_, err = generator.Generate(31) // odd number
	if err == nil {
		t.Fatal("expected error for odd length, got nil")
	}
}
