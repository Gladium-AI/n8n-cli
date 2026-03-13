package client

import (
	"sort"
	"testing"
)

func TestSanitizeWorkflowBody(t *testing.T) {
	// Simulate a rehydrated workflow body that includes read-only fields
	// from the GET response — the exact scenario that causes the 400 error.
	body := map[string]interface{}{
		// Writable fields — should be kept
		"name":        "Test Workflow",
		"nodes":       []interface{}{},
		"connections":  map[string]interface{}{},
		"settings":    map[string]interface{}{},
		"staticData":  "{}",
		"active":      false,
		"tags":        []interface{}{},
		"versionId":   "abc-123",
		"pinData":     map[string]interface{}{},
		// Read-only fields from GET response — MUST be stripped
		"id":                  "wf-123",
		"createdAt":           "2026-01-01T00:00:00Z",
		"updatedAt":           "2026-03-13T00:00:00Z",
		"triggerCount":        float64(5),
		"sharedWithProjects":  []interface{}{},
		"homeProject":         map[string]interface{}{"id": "proj-1"},
		"usedCredentials":     []interface{}{},
		"meta":                map[string]interface{}{},
	}

	clean := sanitizeWorkflowBody(body)

	// Verify all writable fields are present
	expected := []string{"name", "nodes", "connections", "settings", "staticData", "active", "tags", "versionId", "pinData"}
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

	// Verify read-only fields were stripped
	readOnly := []string{"id", "createdAt", "updatedAt", "triggerCount", "sharedWithProjects", "homeProject", "usedCredentials", "meta"}
	for _, k := range readOnly {
		if _, ok := clean[k]; ok {
			t.Errorf("read-only field %q was NOT stripped", k)
		}
	}

	// Verify values are preserved (not just keys)
	if clean["name"] != "Test Workflow" {
		t.Errorf("expected name=Test Workflow, got %v", clean["name"])
	}
	if clean["active"] != false {
		t.Errorf("expected active=false, got %v", clean["active"])
	}
}

func TestSanitizeWorkflowBodyMinimal(t *testing.T) {
	// Only nodes + connections + name — should all survive
	body := map[string]interface{}{
		"name":        "Minimal",
		"nodes":       []interface{}{},
		"connections":  map[string]interface{}{},
	}

	clean := sanitizeWorkflowBody(body)
	if len(clean) != 3 {
		t.Errorf("expected 3 keys, got %d: %v", len(clean), clean)
	}
}
