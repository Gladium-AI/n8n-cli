package parser

import (
	"encoding/json"
	"testing"
)

var sampleWorkflowJSON = `{
  "id": "1",
  "name": "Test Workflow",
  "active": false,
  "nodes": [
    {
      "id": "uuid-start",
      "name": "Start",
      "type": "n8n-nodes-base.start",
      "typeVersion": 1,
      "position": [250, 300],
      "parameters": {}
    },
    {
      "id": "uuid-http",
      "name": "HTTP Request",
      "type": "n8n-nodes-base.httpRequest",
      "typeVersion": 1,
      "position": [450, 300],
      "parameters": {
        "url": "https://example.com",
        "method": "GET"
      }
    },
    {
      "id": "uuid-set",
      "name": "Set Data",
      "type": "n8n-nodes-base.set",
      "typeVersion": 1,
      "position": [650, 300],
      "parameters": {
        "values": {
          "string": [{"name": "key", "value": "val"}]
        }
      },
      "credentials": {
        "httpBasicAuth": {"id": "cred-1", "name": "My Creds"}
      }
    }
  ],
  "connections": {
    "Start": {
      "main": [
        [
          {"node": "HTTP Request", "type": "main", "index": 0}
        ]
      ]
    },
    "HTTP Request": {
      "main": [
        [
          {"node": "Set Data", "type": "main", "index": 0}
        ]
      ]
    }
  },
  "settings": {},
  "tags": [{"id": "tag-1", "name": "test"}]
}`

func loadSampleWorkflow(t *testing.T) (*ParsedWorkflow, map[string]interface{}) {
	t.Helper()
	var raw map[string]interface{}
	if err := json.Unmarshal([]byte(sampleWorkflowJSON), &raw); err != nil {
		t.Fatal(err)
	}
	pw, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return pw, raw
}

func TestParseMeta(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	if pw.Meta.ID != "1" {
		t.Errorf("expected ID=1, got %s", pw.Meta.ID)
	}
	if pw.Meta.Name != "Test Workflow" {
		t.Errorf("expected name=Test Workflow, got %s", pw.Meta.Name)
	}
	if pw.Meta.Active {
		t.Error("expected active=false")
	}
	if len(pw.Meta.Tags) != 1 || pw.Meta.Tags[0].Name != "test" {
		t.Errorf("expected 1 tag named test, got %v", pw.Meta.Tags)
	}
}

func TestParseNodes(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	if len(pw.Nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(pw.Nodes))
	}

	n0 := pw.Nodes[0]
	if n0.Ref != "n0" {
		t.Errorf("expected ref=n0, got %s", n0.Ref)
	}
	if n0.Name != "Start" {
		t.Errorf("expected name=Start, got %s", n0.Name)
	}
	if n0.Type != "n8n-nodes-base.start" {
		t.Errorf("expected type=n8n-nodes-base.start, got %s", n0.Type)
	}
	if n0.Position[0] != 250 || n0.Position[1] != 300 {
		t.Errorf("expected pos=[250,300], got %v", n0.Position)
	}
}

func TestParseEdges(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	if len(pw.Edges) != 2 {
		t.Fatalf("expected 2 edges, got %d", len(pw.Edges))
	}

	e0 := pw.Edges[0]
	if e0.FromName != "Start" || e0.ToName != "HTTP Request" {
		t.Errorf("expected Start->HTTP Request, got %s->%s", e0.FromName, e0.ToName)
	}
	if e0.FromRef != "n0" || e0.ToRef != "n1" {
		t.Errorf("expected n0->n1, got %s->%s", e0.FromRef, e0.ToRef)
	}
}

func TestParseIndexes(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	if _, ok := pw.Indexes.ByRef["n0"]; !ok {
		t.Error("expected ByRef[n0]")
	}
	if _, ok := pw.Indexes.ByName["Start"]; !ok {
		t.Error("expected ByName[Start]")
	}
	if _, ok := pw.Indexes.ByID["uuid-start"]; !ok {
		t.Error("expected ByID[uuid-start]")
	}
}

func TestNodeInboundOutbound(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	start := pw.Indexes.ByRef["n0"]
	if len(start.Inbound) != 0 {
		t.Errorf("start should have 0 inbound, got %d", len(start.Inbound))
	}
	if len(start.Outbound) != 1 {
		t.Errorf("start should have 1 outbound, got %d", len(start.Outbound))
	}

	http := pw.Indexes.ByRef["n1"]
	if len(http.Inbound) != 1 {
		t.Errorf("http should have 1 inbound, got %d", len(http.Inbound))
	}
	if len(http.Outbound) != 1 {
		t.Errorf("http should have 1 outbound, got %d", len(http.Outbound))
	}

	setData := pw.Indexes.ByRef["n2"]
	if len(setData.Inbound) != 1 {
		t.Errorf("set should have 1 inbound, got %d", len(setData.Inbound))
	}
	if len(setData.Outbound) != 0 {
		t.Errorf("set should have 0 outbound, got %d", len(setData.Outbound))
	}
}

func TestResolveRef(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	tests := []struct {
		ref      string
		wantName string
		wantErr  bool
	}{
		{"n0", "Start", false},
		{"n1", "HTTP Request", false},
		{"Start", "Start", false},
		{"HTTP Request", "HTTP Request", false},
		{"ref:n2", "Set Data", false},
		{"id:uuid-http", "HTTP Request", false},
		{`name:Set Data`, "Set Data", false},
		{"nonexistent", "", true},
		{"id:nonexistent", "", true},
	}

	for _, tt := range tests {
		node, err := ResolveRef(pw, tt.ref)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ResolveRef(%q): expected error", tt.ref)
			}
			continue
		}
		if err != nil {
			t.Errorf("ResolveRef(%q): unexpected error: %v", tt.ref, err)
			continue
		}
		if node.Name != tt.wantName {
			t.Errorf("ResolveRef(%q): expected name=%s, got %s", tt.ref, tt.wantName, node.Name)
		}
	}
}

func TestRehydrateRoundTrip(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	rehydrated := Rehydrate(pw)

	if rehydrated["name"] != "Test Workflow" {
		t.Errorf("expected name=Test Workflow, got %v", rehydrated["name"])
	}

	nodes, ok := rehydrated["nodes"].([]interface{})
	if !ok {
		t.Fatal("expected nodes array")
	}
	if len(nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(nodes))
	}

	conns, ok := rehydrated["connections"].(map[string]interface{})
	if !ok {
		t.Fatal("expected connections map")
	}
	if _, ok := conns["Start"]; !ok {
		t.Error("expected Start in connections")
	}
	if _, ok := conns["HTTP Request"]; !ok {
		t.Error("expected HTTP Request in connections")
	}

	pw2, err := Parse(rehydrated)
	if err != nil {
		t.Fatalf("re-parse failed: %v", err)
	}
	if len(pw2.Nodes) != 3 {
		t.Errorf("re-parsed nodes: expected 3, got %d", len(pw2.Nodes))
	}
	if len(pw2.Edges) != 2 {
		t.Errorf("re-parsed edges: expected 2, got %d", len(pw2.Edges))
	}
}

func TestAddNode(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	node, err := AddNode(pw, NodeInput{
		Name:     "New Node",
		Type:     "n8n-nodes-base.code",
		Position: [2]float64{850, 300},
	})
	if err != nil {
		t.Fatal(err)
	}

	if node.Ref != "n3" {
		t.Errorf("expected ref=n3, got %s", node.Ref)
	}
	if node.Name != "New Node" {
		t.Errorf("expected name=New Node, got %s", node.Name)
	}
	if len(pw.Nodes) != 4 {
		t.Errorf("expected 4 nodes, got %d", len(pw.Nodes))
	}

	_, err = AddNode(pw, NodeInput{Name: "New Node", Type: "n8n-nodes-base.code"})
	if err == nil {
		t.Error("expected duplicate name error")
	}
}

func TestRemoveNode(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	err := RemoveNode(pw, "n1", DeleteOptions{})
	if err == nil {
		t.Error("expected error removing connected node without cascade")
	}

	err = RemoveNode(pw, "n1", DeleteOptions{Cascade: true})
	if err != nil {
		t.Fatal(err)
	}

	if len(pw.Nodes) != 2 {
		t.Errorf("expected 2 nodes after delete, got %d", len(pw.Nodes))
	}

	if pw.Nodes[0].Ref != "n0" || pw.Nodes[1].Ref != "n1" {
		t.Errorf("expected refs renumbered to n0,n1, got %s,%s", pw.Nodes[0].Ref, pw.Nodes[1].Ref)
	}

	for _, e := range pw.Edges {
		if e.FromName == "HTTP Request" || e.ToName == "HTTP Request" {
			t.Error("expected HTTP Request edges removed")
		}
	}
}

func TestRemoveNodeBridge(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	err := RemoveNode(pw, "n1", DeleteOptions{RewireStrategy: "bridge"})
	if err != nil {
		t.Fatal(err)
	}

	if len(pw.Nodes) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(pw.Nodes))
	}

	found := false
	for _, e := range pw.Edges {
		if e.FromName == "Start" && e.ToName == "Set Data" {
			found = true
		}
	}
	if !found {
		t.Error("expected bridged connection Start -> Set Data")
	}
}

func TestRenameNode(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	node, err := RenameNode(pw, "n1", "API Call")
	if err != nil {
		t.Fatal(err)
	}

	if node.Name != "API Call" {
		t.Errorf("expected name=API Call, got %s", node.Name)
	}

	for _, e := range pw.Edges {
		if e.FromName == "HTTP Request" || e.ToName == "HTTP Request" {
			t.Error("expected edge names updated from HTTP Request")
		}
	}

	rehydrated := Rehydrate(pw)
	conns := rehydrated["connections"].(map[string]interface{})
	if _, ok := conns["HTTP Request"]; ok {
		t.Error("expected HTTP Request key removed from connections")
	}
	if _, ok := conns["API Call"]; !ok {
		t.Error("expected API Call key in connections")
	}
}

func TestMoveNode(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	node, err := MoveNode(pw, "n0", 100, 200)
	if err != nil {
		t.Fatal(err)
	}

	if node.Position[0] != 100 || node.Position[1] != 200 {
		t.Errorf("expected pos=[100,200], got %v", node.Position)
	}
}

func TestEnableDisableNode(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	node, err := DisableNode(pw, "n1")
	if err != nil {
		t.Fatal(err)
	}
	if !node.Disabled {
		t.Error("expected disabled=true")
	}

	node, err = EnableNode(pw, "n1")
	if err != nil {
		t.Fatal(err)
	}
	if node.Disabled {
		t.Error("expected disabled=false")
	}
}

func TestAddEdge(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	AddNode(pw, NodeInput{Name: "Final", Type: "n8n-nodes-base.noOp", Position: [2]float64{850, 300}})

	edge, err := AddEdge(pw, EdgeInput{FromRef: "n2", ToRef: "n3", FromOutput: 0, ToInput: 0})
	if err != nil {
		t.Fatal(err)
	}

	if edge.FromName != "Set Data" || edge.ToName != "Final" {
		t.Errorf("expected Set Data->Final, got %s->%s", edge.FromName, edge.ToName)
	}
	if len(pw.Edges) != 3 {
		t.Errorf("expected 3 edges, got %d", len(pw.Edges))
	}

	_, err = AddEdge(pw, EdgeInput{FromRef: "n2", ToRef: "n3", FromOutput: 0, ToInput: 0})
	if err == nil {
		t.Error("expected duplicate edge error")
	}
}

func TestRemoveEdge(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	err := RemoveEdge(pw, EdgeInput{FromRef: "n0", ToRef: "n1", FromOutput: 0, ToInput: 0})
	if err != nil {
		t.Fatal(err)
	}

	if len(pw.Edges) != 1 {
		t.Errorf("expected 1 edge, got %d", len(pw.Edges))
	}
}

func TestUpdateNodePatches(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	node, err := UpdateNodePatches(pw, "n1", []NodePatch{
		{Path: "parameters.url", Value: "https://new.example.com"},
		{Path: "parameters.method", Value: "POST"},
	})
	if err != nil {
		t.Fatal(err)
	}

	if node.Parameters["url"] != "https://new.example.com" {
		t.Errorf("expected url updated, got %v", node.Parameters["url"])
	}
	if node.Parameters["method"] != "POST" {
		t.Errorf("expected method=POST, got %v", node.Parameters["method"])
	}
}

func TestGraphAnalysis(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	analysis := AnalyzeGraph(pw)

	if analysis.NodeCount != 3 {
		t.Errorf("expected 3 nodes, got %d", analysis.NodeCount)
	}
	if analysis.EdgeCount != 2 {
		t.Errorf("expected 2 edges, got %d", analysis.EdgeCount)
	}
	if analysis.HasCycles {
		t.Error("expected no cycles")
	}
	if len(analysis.Roots) != 1 || analysis.Roots[0] != "n0" {
		t.Errorf("expected roots=[n0], got %v", analysis.Roots)
	}
	if len(analysis.Leaves) != 1 || analysis.Leaves[0] != "n2" {
		t.Errorf("expected leaves=[n2], got %v", analysis.Leaves)
	}
	if len(analysis.TopologicalOrder) != 3 {
		t.Errorf("expected topological order length=3, got %d", len(analysis.TopologicalOrder))
	}
}

func TestParseSetFlag(t *testing.T) {
	tests := []struct {
		input   string
		path    string
		value   interface{}
		wantErr bool
	}{
		{`parameters.url=https://example.com`, "parameters.url", "https://example.com", false},
		{`parameters.count=42`, "parameters.count", float64(42), false},
		{`parameters.enabled=true`, "parameters.enabled", true, false},
		{`noequals`, "", nil, true},
	}

	for _, tt := range tests {
		p, err := ParseSetFlag(tt.input)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ParseSetFlag(%q): expected error", tt.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseSetFlag(%q): unexpected error: %v", tt.input, err)
			continue
		}
		if p.Path != tt.path {
			t.Errorf("ParseSetFlag(%q): expected path=%s, got %s", tt.input, tt.path, p.Path)
		}
	}
}
