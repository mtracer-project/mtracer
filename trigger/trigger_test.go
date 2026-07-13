package trigger_test

import (
	"context"
	"testing"

	"github.com/mtracer-project/mtracer/parser"
	testutils "github.com/mtracer-project/mtracer/testUtils"
	"github.com/mtracer-project/mtracer/trigger"
)

func TestUnsupportedTrigger(t *testing.T) {
	dto := &parser.TriggerDTO{
		Type: "unknownType",
	}
	_, err := trigger.NewTrigger(dto, &testutils.MockIdGenerator{}, "", context.Background())
	if err == nil {
		t.Error("Expected error for unsupported trigger type")
	}
}
