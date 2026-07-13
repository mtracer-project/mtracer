package domain_test

import (
	"testing"
	"time"

	"github.com/mtracer-project/mtracer/domain"

	"go.yaml.in/yaml/v3"
)

func TestDuration_UnmarshalYAML(t *testing.T) {
	// Case 1: valid duration string
	var d domain.Duration
	node := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "5s",
	}
	err := d.UnmarshalYAML(node)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if time.Duration(d) != 5*time.Second {
		t.Errorf("Expected 5s, got %v", time.Duration(d))
	}

	// Case 2: invalid duration string
	nodeInvalid := &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: "invalid-duration",
	}
	err = d.UnmarshalYAML(nodeInvalid)
	if err == nil {
		t.Error("Expected error for invalid duration string, got nil")
	}

	// Case 3: Decode error (e.g. wrong yaml kind)
	nodeWrongKind := &yaml.Node{
		Kind:  yaml.SequenceNode,
		Value: "",
	}
	err = d.UnmarshalYAML(nodeWrongKind)
	if err == nil {
		t.Error("Expected error for non-scalar node type, got nil")
	}
}

func TestDuration_ToTimeDuration(t *testing.T) {
	// Case 1: nil duration pointer
	var nilDur *domain.Duration
	if nilDur.ToTimeDuration() != nil {
		t.Errorf("Expected ToTimeDuration on nil pointer to return nil, got %v", nilDur.ToTimeDuration())
	}

	// Case 2: non-nil duration pointer
	dur := domain.Duration(10 * time.Minute)
	res := dur.ToTimeDuration()
	if res == nil || *res != 10*time.Minute {
		t.Errorf("Expected 10m, got %v", res)
	}
}

func TestFromTimeDuration(t *testing.T) {
	res := domain.FromTimeDuration(12 * time.Hour)
	if time.Duration(res) != 12*time.Hour {
		t.Errorf("Expected 12h, got %v", res)
	}
}

func TestNanosecondsToTime(t *testing.T) {
	ns := int64(1717590000000000000) // some timestamp
	res := domain.NanosecondsToTime(ns)
	expected := time.Unix(0, ns).UTC()
	if !res.Equal(expected) {
		t.Errorf("Expected time %v, got %v", expected, res)
	}
}

func TestDerefString(t *testing.T) {
	// Case 1: nil pointer
	if domain.DerefString(nil) != "" {
		t.Errorf("Expected empty string for nil pointer, got %q", domain.DerefString(nil))
	}

	// Case 2: non-nil pointer
	s := "hello"
	if domain.DerefString(&s) != "hello" {
		t.Errorf("Expected 'hello', got %q", domain.DerefString(&s))
	}
}
