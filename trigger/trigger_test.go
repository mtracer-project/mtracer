package trigger_test

import (
	"context"
	"testing"

	"github.com/mtrace-project/mtrace/parser"
	testutils "github.com/mtrace-project/mtrace/testUtils"
	"github.com/mtrace-project/mtrace/trigger"
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
