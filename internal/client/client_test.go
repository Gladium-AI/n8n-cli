package client

import (
	"sort"
	"testing"
)

func TestSanitizeWorkflowBody(t *testing.T) {
	body := map[string]interface{}{
		// Allowed fields — should be kept
		"name":        "Test Workflow",
		"nodes":       []interface{}{},
		"connections": map[string]interface{}{},
		"settings":    map[string]interface{}{},
		"staticData":  "{}",
		"pinData":     map[string]interface{}{},
		"versionId":   "abc-123",
		// Common GET/read-only fields — MUST be stripped
		"id":             "wf-123",
		"createdAt":      "2026-01-01T00:00:00Z",
		"updatedAt":      "2026-03-13T00:00:00Z",
		"triggerCount":   float64(5),
		"meta":           map[string]interface{}{},
		"active":         false,
		"tags":           []interface{}{},
		"description":    "desc",
		"isArchived":     false,
		"shared":         []interface{}{},
		"versionCounter": float64(7),
	}

	clean := sanitizeWorkflowBody(body)

	expected := []string{"connections", "name", "nodes", "pinData", "settings", "staticData", "versionId"}
	sort.Strings(expected)

	var got []string
	for k := range clean {
		got = append(got, k)
	}
	sort.Strings(got)

	if len(got) != len(expected) {
		t.Fatalf("expected %d keys, got %d: %v", len(expected), len(got), got)
	}
	for i := range expected {
		if got[i] != expected[i] {
			t.Errorf("key %d: expected %q, got %q", i, expected[i], got[i])
		}
	}

	readOnly := []string{"id", "createdAt", "updatedAt", "triggerCount", "meta", "active", "tags", "description", "isArchived", "shared", "versionCounter"}
	for _, k := range readOnly {
		if _, ok := clean[k]; ok {
			t.Errorf("field %q was NOT stripped", k)
		}
	}
}

func TestSanitizeWorkflowBodyMinimal(t *testing.T) {
	body := map[string]interface{}{
		"name":        "Minimal",
		"nodes":       []interface{}{},
		"connections": map[string]interface{}{},
	}

	clean := sanitizeWorkflowBody(body)
	if len(clean) != 3 {
		t.Errorf("expected 3 keys, got %d: %v", len(clean), clean)
	}
}
