package parser

import (
	"encoding/json"
	"fmt"
	"strings"
)

func Parse(raw map[string]interface{}) (*ParsedWorkflow, error) {
	pw := &ParsedWorkflow{
		RawWorkflow: raw,
		Indexes: WorkflowIndexes{
			ByRef:  make(map[string]*ParsedNode),
			ByName: make(map[string][]*ParsedNode),
			ByID:   make(map[string]*ParsedNode),
		},
	}

	pw.Meta = extractMeta(raw)

	rawNodes, _ := raw["nodes"].([]interface{})
	for i, rn := range rawNodes {
		nodeMap, ok := rn.(map[string]interface{})
		if !ok {
			continue
		}
		node := parseNode(nodeMap, i)
		pw.Nodes = append(pw.Nodes, node)
	}

	buildIndexes(pw)
	pw.Edges = extractEdges(raw, pw)
	linkEdgesToNodes(pw)

	return pw, nil
}

func extractMeta(raw map[string]interface{}) WorkflowMeta {
	meta := WorkflowMeta{
		ID:     stringField(raw, "id"),
		Name:   stringField(raw, "name"),
		Active: boolField(raw, "active"),
	}

	if tags, ok := raw["tags"].([]interface{}); ok {
		for _, t := range tags {
			if tm, ok := t.(map[string]interface{}); ok {
				meta.Tags = append(meta.Tags, TagInfo{
					ID:   stringField(tm, "id"),
					Name: stringField(tm, "name"),
				})
			}
		}
	}

	if settings, ok := raw["settings"].(map[string]interface{}); ok {
		meta.Settings = settings
	}

	meta.CreatedAt = stringField(raw, "createdAt")
	meta.UpdatedAt = stringField(raw, "updatedAt")
	meta.VersionID = stringField(raw, "versionId")

	return meta
}

func parseNode(m map[string]interface{}, index int) *ParsedNode {
	node := &ParsedNode{
		Ref:      fmt.Sprintf("n%d", index),
		ID:       stringField(m, "id"),
		Name:     stringField(m, "name"),
		Type:     stringField(m, "type"),
		Disabled: boolField(m, "disabled"),
		Notes:    stringField(m, "notes"),
	}

	if tv, ok := m["typeVersion"]; ok {
		node.TypeVersion = intField(m, "typeVersion")
		_ = tv
	}

	if pos, ok := m["position"].([]interface{}); ok && len(pos) >= 2 {
		node.Position[0] = toFloat64(pos[0])
		node.Position[1] = toFloat64(pos[1])
	}

	if params, ok := m["parameters"].(map[string]interface{}); ok {
		node.Parameters = params
	} else {
		node.Parameters = make(map[string]interface{})
	}

	if creds, ok := m["credentials"].(map[string]interface{}); ok {
		node.Credentials = creds
	} else {
		node.Credentials = make(map[string]interface{})
	}

	return node
}

func extractEdges(raw map[string]interface{}, pw *ParsedWorkflow) []*ParsedEdge {
	var edges []*ParsedEdge

	connections, ok := raw["connections"].(map[string]interface{})
	if !ok {
		return edges
	}

	for fromName, outputTypes := range connections {
		outputTypeMap, ok := outputTypes.(map[string]interface{})
		if !ok {
			continue
		}
		for _, outputs := range outputTypeMap {
			outputArr, ok := outputs.([]interface{})
			if !ok {
				continue
			}
			for outputIdx, connections := range outputArr {
				connArr, ok := connections.([]interface{})
				if !ok {
					continue
				}
				for _, conn := range connArr {
					connMap, ok := conn.(map[string]interface{})
					if !ok {
						continue
					}
					toName := stringField(connMap, "node")
					toInput := intField(connMap, "index")

					fromRef := resolveRefByName(pw, fromName)
					toRef := resolveRefByName(pw, toName)

					edge := &ParsedEdge{
						FromRef:    fromRef,
						ToRef:      toRef,
						FromOutput: outputIdx,
						ToInput:    toInput,
						FromName:   fromName,
						ToName:     toName,
					}
					edges = append(edges, edge)
				}
			}
		}
	}

	return edges
}

func resolveRefByName(pw *ParsedWorkflow, name string) string {
	for _, n := range pw.Nodes {
		if n.Name == name {
			return n.Ref
		}
	}
	return ""
}

func buildIndexes(pw *ParsedWorkflow) {
	pw.Indexes.ByRef = make(map[string]*ParsedNode)
	pw.Indexes.ByName = make(map[string][]*ParsedNode)
	pw.Indexes.ByID = make(map[string]*ParsedNode)

	for _, n := range pw.Nodes {
		pw.Indexes.ByRef[n.Ref] = n
		pw.Indexes.ByName[n.Name] = append(pw.Indexes.ByName[n.Name], n)
		if n.ID != "" {
			pw.Indexes.ByID[n.ID] = n
		}
	}
}

func linkEdgesToNodes(pw *ParsedWorkflow) {
	for _, n := range pw.Nodes {
		n.Inbound = nil
		n.Outbound = nil
	}
	for _, e := range pw.Edges {
		if from, ok := pw.Indexes.ByRef[e.FromRef]; ok {
			from.Outbound = append(from.Outbound, e)
		}
		if to, ok := pw.Indexes.ByRef[e.ToRef]; ok {
			to.Inbound = append(to.Inbound, e)
		}
	}
}

func Rehydrate(pw *ParsedWorkflow) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range pw.RawWorkflow {
		if k != "nodes" && k != "connections" {
			result[k] = v
		}
	}

	result["name"] = pw.Meta.Name
	result["active"] = pw.Meta.Active

	nodes := make([]interface{}, 0, len(pw.Nodes))
	for _, n := range pw.Nodes {
		nodeMap := rehydrateNode(n)
		nodes = append(nodes, nodeMap)
	}
	result["nodes"] = nodes

	connections := rehydrateConnections(pw)
	result["connections"] = connections

	return result
}

func rehydrateNode(n *ParsedNode) map[string]interface{} {
	m := map[string]interface{}{
		"name":        n.Name,
		"type":        n.Type,
		"typeVersion": n.TypeVersion,
		"position":    []interface{}{n.Position[0], n.Position[1]},
		"parameters":  n.Parameters,
	}
	if n.ID != "" {
		m["id"] = n.ID
	}
	if n.Disabled {
		m["disabled"] = true
	}
	if len(n.Credentials) > 0 {
		m["credentials"] = n.Credentials
	}
	if n.Notes != "" {
		m["notes"] = n.Notes
	}
	return m
}

func rehydrateConnections(pw *ParsedWorkflow) map[string]interface{} {
	connMap := make(map[string]interface{})

	type edgeKey struct {
		fromName   string
		outputType string
		outputIdx  int
	}
	grouped := make(map[edgeKey][]*ParsedEdge)

	for _, e := range pw.Edges {
		fromNode := pw.Indexes.ByRef[e.FromRef]
		if fromNode == nil {
			continue
		}
		key := edgeKey{fromName: fromNode.Name, outputType: "main", outputIdx: e.FromOutput}
		grouped[key] = append(grouped[key], e)
	}

	type sourceKey struct {
		fromName   string
		outputType string
	}
	maxOutputIdx := make(map[sourceKey]int)
	seenSources := make(map[sourceKey]bool)
	for k := range grouped {
		sk := sourceKey{fromName: k.fromName, outputType: k.outputType}
		seenSources[sk] = true
		if k.outputIdx > maxOutputIdx[sk] {
			maxOutputIdx[sk] = k.outputIdx
		}
	}

	for sk := range seenSources {
		maxIdx := maxOutputIdx[sk]
		if _, exists := connMap[sk.fromName]; !exists {
			connMap[sk.fromName] = make(map[string]interface{})
		}
		nodeConn := connMap[sk.fromName].(map[string]interface{})

		outputs := make([]interface{}, maxIdx+1)
		for i := 0; i <= maxIdx; i++ {
			key := edgeKey{fromName: sk.fromName, outputType: sk.outputType, outputIdx: i}
			edges := grouped[key]
			conns := make([]interface{}, 0, len(edges))
			for _, e := range edges {
				toNode := pw.Indexes.ByRef[e.ToRef]
				if toNode == nil {
					continue
				}
				conns = append(conns, map[string]interface{}{
					"node":  toNode.Name,
					"type":  "main",
					"index": e.ToInput,
				})
			}
			outputs[i] = conns
		}
		nodeConn[sk.outputType] = outputs
	}

	return connMap
}

func ResolveRef(pw *ParsedWorkflow, ref string) (*ParsedNode, error) {
	if ref == "" {
		return nil, fmt.Errorf("node reference is required")
	}

	if strings.HasPrefix(ref, "id:") {
		id := strings.TrimPrefix(ref, "id:")
		if node, ok := pw.Indexes.ByID[id]; ok {
			return node, nil
		}
		return nil, fmt.Errorf("no node with ID %q", id)
	}

	if strings.HasPrefix(ref, "ref:") {
		r := strings.TrimPrefix(ref, "ref:")
		if node, ok := pw.Indexes.ByRef[r]; ok {
			return node, nil
		}
		return nil, fmt.Errorf("no node with ref %q", r)
	}

	if strings.HasPrefix(ref, "name:") {
		name := strings.TrimPrefix(ref, "name:")
		name = strings.Trim(name, `"'`)
		nodes := pw.Indexes.ByName[name]
		if len(nodes) == 0 {
			return nil, fmt.Errorf("no node with name %q", name)
		}
		if len(nodes) > 1 {
			return nil, fmt.Errorf("ambiguous name %q: %d nodes match; use ref: or id: prefix", name, len(nodes))
		}
		return nodes[0], nil
	}

	if node, ok := pw.Indexes.ByRef[ref]; ok {
		return node, nil
	}

	nodes := pw.Indexes.ByName[ref]
	if len(nodes) == 1 {
		return nodes[0], nil
	}
	if len(nodes) > 1 {
		refs := make([]string, len(nodes))
		for i, n := range nodes {
			refs[i] = n.Ref
		}
		return nil, fmt.Errorf("ambiguous name %q: matches %s; use ref: or id: prefix", ref, strings.Join(refs, ", "))
	}

	if node, ok := pw.Indexes.ByID[ref]; ok {
		return node, nil
	}

	return nil, fmt.Errorf("no node matching %q (tried ref, name, id)", ref)
}

func NextRef(pw *ParsedWorkflow) string {
	return fmt.Sprintf("n%d", len(pw.Nodes))
}

func RebuildIndexes(pw *ParsedWorkflow) {
	buildIndexes(pw)
	linkEdgesToNodes(pw)
}

func DeepCopyRaw(raw map[string]interface{}) (map[string]interface{}, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var copy map[string]interface{}
	if err := json.Unmarshal(b, &copy); err != nil {
		return nil, err
	}
	return copy, nil
}

// --- helpers ---

func stringField(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func boolField(m map[string]interface{}, key string) bool {
	v, ok := m[key]
	if !ok || v == nil {
		return false
	}
	b, ok := v.(bool)
	if ok {
		return b
	}
	return false
}

func intField(m map[string]interface{}, key string) int {
	v, ok := m[key]
	if !ok || v == nil {
		return 0
	}
	return int(toFloat64(v))
}

func toFloat64(v interface{}) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case json.Number:
		f, _ := n.Float64()
		return f
	}
	return 0
}
