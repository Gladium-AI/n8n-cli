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

	// Don't rely on iteration order — look for both edges by content
	edgeSet := make(map[string]bool)
	for _, e := range pw.Edges {
		edgeSet[e.FromName+"->"+e.ToName] = true
	}

	if !edgeSet["Start->HTTP Request"] {
		t.Error("expected edge Start->HTTP Request")
	}
	if !edgeSet["HTTP Request->Set Data"] {
		t.Error("expected edge HTTP Request->Set Data")
	}

	// Verify refs are correct by looking up each edge
	for _, e := range pw.Edges {
		if e.FromName == "Start" && e.ToName == "HTTP Request" {
			if e.FromRef != "n0" || e.ToRef != "n1" {
				t.Errorf("Start->HTTP Request: expected n0->n1, got %s->%s", e.FromRef, e.ToRef)
			}
		}
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

func TestRawJSONPreservation(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	// Every parsed node should have a non-nil RawJSON
	for _, n := range pw.Nodes {
		if n.RawJSON == nil {
			t.Errorf("node %s: RawJSON is nil", n.Name)
		}
	}

	// RawJSON should contain native fields
	httpNode := pw.Indexes.ByRef["n1"]
	raw := httpNode.RawJSON
	if raw["name"] != "HTTP Request" {
		t.Errorf("expected RawJSON[name]=HTTP Request, got %v", raw["name"])
	}
	if raw["type"] != "n8n-nodes-base.httpRequest" {
		t.Errorf("expected RawJSON[type]=n8n-nodes-base.httpRequest, got %v", raw["type"])
	}

	// RawJSON should be a deep copy — mutating it shouldn't affect the ParsedNode
	params, ok := raw["parameters"].(map[string]interface{})
	if !ok {
		t.Fatal("expected RawJSON[parameters] to be a map")
	}
	params["url"] = "MUTATED"
	if httpNode.Parameters["url"] == "MUTATED" {
		t.Error("RawJSON is not a deep copy — mutation leaked to ParsedNode.Parameters")
	}
}

func TestNodeToNativeJSON(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)

	setNode := pw.Indexes.ByRef["n2"]
	native := NodeToNativeJSON(setNode)

	if native["name"] != "Set Data" {
		t.Errorf("expected name=Set Data, got %v", native["name"])
	}
	if native["type"] != "n8n-nodes-base.set" {
		t.Errorf("expected type=n8n-nodes-base.set, got %v", native["type"])
	}

	// Should include credentials
	creds, ok := native["credentials"].(map[string]interface{})
	if !ok {
		t.Fatal("expected credentials map in native JSON")
	}
	if _, ok := creds["httpBasicAuth"]; !ok {
		t.Error("expected httpBasicAuth in credentials")
	}

	// Should include parameters
	params, ok := native["parameters"].(map[string]interface{})
	if !ok {
		t.Fatal("expected parameters map in native JSON")
	}
	if _, ok := params["values"]; !ok {
		t.Error("expected values in parameters")
	}

	// Should include position as array
	pos, ok := native["position"].([]interface{})
	if !ok || len(pos) != 2 {
		t.Error("expected position as 2-element array")
	}
}

func TestExtractPath(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)
	httpNode := pw.Indexes.ByRef["n1"]

	tests := []struct {
		path    string
		wantStr string
		wantErr bool
	}{
		{"name", "HTTP Request", false},
		{"type", "n8n-nodes-base.httpRequest", false},
		{"parameters.url", "https://example.com", false},
		{"parameters.method", "GET", false},
		{"parameters.nonexistent", "", true},
		{"nonexistent.deep.path", "", true},
	}

	for _, tt := range tests {
		val, err := ExtractPath(httpNode, tt.path)
		if tt.wantErr {
			if err == nil {
				t.Errorf("ExtractPath(%q): expected error", tt.path)
			}
			continue
		}
		if err != nil {
			t.Errorf("ExtractPath(%q): unexpected error: %v", tt.path, err)
			continue
		}
		s, ok := val.(string)
		if !ok {
			t.Errorf("ExtractPath(%q): expected string, got %T", tt.path, val)
			continue
		}
		if s != tt.wantStr {
			t.Errorf("ExtractPath(%q): expected %q, got %q", tt.path, tt.wantStr, s)
		}
	}

	// Test nested path on credentials node
	setNode := pw.Indexes.ByRef["n2"]
	val, err := ExtractPath(setNode, "credentials.httpBasicAuth.id")
	if err != nil {
		t.Fatalf("ExtractPath credentials: %v", err)
	}
	if val != "cred-1" {
		t.Errorf("expected cred-1, got %v", val)
	}
}

func TestDetectChanges(t *testing.T) {
	before := map[string]interface{}{
		"name":       "Old Name",
		"type":       "n8n-nodes-base.httpRequest",
		"parameters": map[string]interface{}{"url": "https://old.com", "method": "GET"},
	}
	after := map[string]interface{}{
		"name":       "Old Name",
		"type":       "n8n-nodes-base.httpRequest",
		"parameters": map[string]interface{}{"url": "https://new.com", "method": "GET"},
	}

	changes := DetectChanges(before, after)
	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d: %v", len(changes), changes)
	}
	if changes[0] != "parameters.url" {
		t.Errorf("expected changed path=parameters.url, got %s", changes[0])
	}

	// Test field addition
	after["newField"] = "value"
	changes = DetectChanges(before, after)
	found := false
	for _, c := range changes {
		if c == "newField (added)" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'newField (added)' in changes, got %v", changes)
	}

	// Test field removal
	delete(after, "newField")
	before["extra"] = "gone"
	changes = DetectChanges(before, after)
	found = false
	for _, c := range changes {
		if c == "extra (removed)" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'extra (removed)' in changes, got %v", changes)
	}
}

func TestSnapshotNodeIsolation(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)
	httpNode := pw.Indexes.ByRef["n1"]

	snapshot := SnapshotNode(httpNode)

	// Mutating the snapshot should NOT affect the original node
	params, ok := snapshot["parameters"].(map[string]interface{})
	if !ok {
		t.Fatal("expected params map in snapshot")
	}
	params["url"] = "MUTATED"

	if httpNode.Parameters["url"] == "MUTATED" {
		t.Error("SnapshotNode is not isolated — mutation leaked to original node")
	}
}

func TestSyncRawJSON(t *testing.T) {
	pw, _ := loadSampleWorkflow(t)
	httpNode := pw.Indexes.ByRef["n1"]

	// Mutate the node
	httpNode.Parameters["url"] = "https://updated.example.com"

	// RawJSON still has original value
	rawParams, _ := httpNode.RawJSON["parameters"].(map[string]interface{})
	if rawParams["url"] == "https://updated.example.com" {
		t.Error("RawJSON should not auto-update when Parameters mutated")
	}

	// Sync should update RawJSON
	SyncRawJSON(httpNode)
	synced := httpNode.RawJSON
	syncedParams, ok := synced["parameters"].(map[string]interface{})
	if !ok {
		t.Fatal("expected parameters in synced RawJSON")
	}
	if syncedParams["url"] != "https://updated.example.com" {
		t.Errorf("expected synced url=https://updated.example.com, got %v", syncedParams["url"])
	}
}

func TestDeepCopyRaw(t *testing.T) {
	original := map[string]interface{}{
		"key": "value",
		"nested": map[string]interface{}{
			"inner": "data",
		},
	}

	copied, err := DeepCopyRaw(original)
	if err != nil {
		t.Fatal(err)
	}

	// Mutate the copy
	copied["key"] = "changed"
	nested, _ := copied["nested"].(map[string]interface{})
	nested["inner"] = "changed"

	// Original should be unaffected
	if original["key"] != "value" {
		t.Error("DeepCopyRaw: mutation leaked to original (top-level)")
	}
	origNested, _ := original["nested"].(map[string]interface{})
	if origNested["inner"] != "data" {
		t.Error("DeepCopyRaw: mutation leaked to original (nested)")
	}
}

func TestParseNodeView(t *testing.T) {
	tests := []struct {
		input string
		want  NodeView
	}{
		{"summary", ViewSummary},
		{"details", ViewDetails},
		{"json", ViewJSON},
		{"params", ViewParams},
		{"connections", ViewConnections},
		{"SUMMARY", ViewSummary},
		{"JSON", ViewJSON},
		{"unknown", ViewSummary},
		{"", ViewSummary},
	}

	for _, tt := range tests {
		got := ParseNodeView(tt.input)
		if got != tt.want {
			t.Errorf("ParseNodeView(%q): expected %q, got %q", tt.input, tt.want, got)
		}
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
