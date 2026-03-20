package parser

import (
	"encoding/json"
	"fmt"
	"strings"
)

func AddNode(pw *ParsedWorkflow, input NodeInput) (*ParsedNode, error) {
	if input.Type == "" {
		return nil, fmt.Errorf("node type is required")
	}

	name := input.Name
	if name == "" {
		name = generateNodeName(pw, input.Type)
	}

	if len(pw.Indexes.ByName[name]) > 0 {
		return nil, fmt.Errorf("node name %q already exists", name)
	}

	node := &ParsedNode{
		Ref:         NextRef(pw),
		Name:        name,
		Type:        input.Type,
		TypeVersion: input.TypeVersion,
		Position:    input.Position,
		Disabled:    input.Disabled,
		Parameters:  input.Parameters,
		Credentials: input.Credentials,
		Notes:       input.Notes,
	}

	if input.RawJSON != nil {
		if id, ok := input.RawJSON["id"].(string); ok {
			node.ID = id
		}
	}

	if node.TypeVersion == 0 {
		node.TypeVersion = 1
	}
	if node.Parameters == nil {
		node.Parameters = make(map[string]interface{})
	}
	if node.Credentials == nil {
		node.Credentials = make(map[string]interface{})
	}

	pw.Nodes = append(pw.Nodes, node)
	RebuildIndexes(pw)

	return node, nil
}

func UpdateNodePatches(pw *ParsedWorkflow, ref string, patches []NodePatch) (*ParsedNode, error) {
	node, err := ResolveRef(pw, ref)
	if err != nil {
		return nil, err
	}

	for _, patch := range patches {
		if err := applyPatch(node, patch); err != nil {
			return nil, fmt.Errorf("apply patch %q: %w", patch.Path, err)
		}
	}

	RebuildIndexes(pw)
	return node, nil
}

func UpdateNodeMerge(pw *ParsedWorkflow, ref string, merge map[string]interface{}) (*ParsedNode, error) {
	node, err := ResolveRef(pw, ref)
	if err != nil {
		return nil, err
	}

	if name, ok := merge["name"].(string); ok {
		oldName := node.Name
		node.Name = name
		updateEdgeNames(pw, oldName, name)
	}
	if typ, ok := merge["type"].(string); ok {
		node.Type = typ
	}
	if tv, ok := merge["typeVersion"]; ok {
		node.TypeVersion = int(toFloat64(tv))
	}
	if pos, ok := merge["position"].([]interface{}); ok && len(pos) >= 2 {
		node.Position[0] = toFloat64(pos[0])
		node.Position[1] = toFloat64(pos[1])
	}
	if disabled, ok := merge["disabled"].(bool); ok {
		node.Disabled = disabled
	}
	if params, ok := merge["parameters"].(map[string]interface{}); ok {
		deepMerge(node.Parameters, params)
	}
	if creds, ok := merge["credentials"].(map[string]interface{}); ok {
		deepMerge(node.Credentials, creds)
	}
	if notes, ok := merge["notes"].(string); ok {
		node.Notes = notes
	}

	RebuildIndexes(pw)
	return node, nil
}

func ReplaceNode(pw *ParsedWorkflow, ref string, replacement map[string]interface{}) (*ParsedNode, error) {
	node, err := ResolveRef(pw, ref)
	if err != nil {
		return nil, err
	}

	oldName := node.Name
	idx := -1
	for i, n := range pw.Nodes {
		if n.Ref == node.Ref {
			idx = i
			break
		}
	}
	if idx < 0 {
		return nil, fmt.Errorf("internal error: node ref %q not found in nodes list", ref)
	}

	newNode := parseNode(replacement, idx)
	newNode.Ref = node.Ref
	newNode.Inbound = node.Inbound
	newNode.Outbound = node.Outbound

	pw.Nodes[idx] = newNode

	if newNode.Name != oldName {
		updateEdgeNames(pw, oldName, newNode.Name)
	}

	RebuildIndexes(pw)
	return newNode, nil
}

func RemoveNode(pw *ParsedWorkflow, ref string, opts DeleteOptions) error {
	node, err := ResolveRef(pw, ref)
	if err != nil {
		return err
	}

	if opts.RewireStrategy == "bridge" && len(node.Inbound) > 0 && len(node.Outbound) > 0 {
		for _, inEdge := range node.Inbound {
			for _, outEdge := range node.Outbound {
				newEdge := &ParsedEdge{
					FromRef:    inEdge.FromRef,
					ToRef:      outEdge.ToRef,
					FromOutput: inEdge.FromOutput,
					ToInput:    outEdge.ToInput,
					FromName:   inEdge.FromName,
					ToName:     outEdge.ToName,
				}
				pw.Edges = append(pw.Edges, newEdge)
			}
		}
	}

	if opts.Cascade || opts.RewireStrategy != "" {
		pw.Edges = filterEdges(pw.Edges, func(e *ParsedEdge) bool {
			return e.FromRef != node.Ref && e.ToRef != node.Ref
		})
	} else {
		hasEdges := false
		for _, e := range pw.Edges {
			if e.FromRef == node.Ref || e.ToRef == node.Ref {
				hasEdges = true
				break
			}
		}
		if hasEdges {
			return fmt.Errorf("node %q has connections; use --cascade to remove them or --rewire to bridge", ref)
		}
	}

	pw.Nodes = filterNodes(pw.Nodes, func(n *ParsedNode) bool {
		return n.Ref != node.Ref
	})

	for i, n := range pw.Nodes {
		n.Ref = fmt.Sprintf("n%d", i)
	}

	RebuildIndexes(pw)
	return nil
}

func RenameNode(pw *ParsedWorkflow, ref string, newName string) (*ParsedNode, error) {
	if newName == "" {
		return nil, fmt.Errorf("new name is required")
	}

	node, err := ResolveRef(pw, ref)
	if err != nil {
		return nil, err
	}

	if existing := pw.Indexes.ByName[newName]; len(existing) > 0 {
		return nil, fmt.Errorf("node name %q already exists", newName)
	}

	oldName := node.Name
	node.Name = newName
	updateEdgeNames(pw, oldName, newName)
	RebuildIndexes(pw)

	return node, nil
}

func MoveNode(pw *ParsedWorkflow, ref string, x, y float64) (*ParsedNode, error) {
	node, err := ResolveRef(pw, ref)
	if err != nil {
		return nil, err
	}
	node.Position[0] = x
	node.Position[1] = y
	return node, nil
}

func EnableNode(pw *ParsedWorkflow, ref string) (*ParsedNode, error) {
	node, err := ResolveRef(pw, ref)
	if err != nil {
		return nil, err
	}
	node.Disabled = false
	return node, nil
}

func DisableNode(pw *ParsedWorkflow, ref string) (*ParsedNode, error) {
	node, err := ResolveRef(pw, ref)
	if err != nil {
		return nil, err
	}
	node.Disabled = true
	return node, nil
}

func AddEdge(pw *ParsedWorkflow, input EdgeInput) (*ParsedEdge, error) {
	fromNode, err := ResolveRef(pw, input.FromRef)
	if err != nil {
		return nil, fmt.Errorf("from node: %w", err)
	}
	toNode, err := ResolveRef(pw, input.ToRef)
	if err != nil {
		return nil, fmt.Errorf("to node: %w", err)
	}

	for _, e := range pw.Edges {
		if e.FromRef == fromNode.Ref && e.ToRef == toNode.Ref &&
			e.FromOutput == input.FromOutput && e.ToInput == input.ToInput {
			return nil, fmt.Errorf("connection already exists: %s[%d] -> %s[%d]",
				fromNode.Name, input.FromOutput, toNode.Name, input.ToInput)
		}
	}

	edge := &ParsedEdge{
		FromRef:    fromNode.Ref,
		ToRef:      toNode.Ref,
		FromOutput: input.FromOutput,
		ToInput:    input.ToInput,
		FromName:   fromNode.Name,
		ToName:     toNode.Name,
	}

	pw.Edges = append(pw.Edges, edge)
	linkEdgesToNodes(pw)

	return edge, nil
}

func RemoveEdge(pw *ParsedWorkflow, input EdgeInput) error {
	fromNode, err := ResolveRef(pw, input.FromRef)
	if err != nil {
		return fmt.Errorf("from node: %w", err)
	}
	toNode, err := ResolveRef(pw, input.ToRef)
	if err != nil {
		return fmt.Errorf("to node: %w", err)
	}

	found := false
	pw.Edges = filterEdges(pw.Edges, func(e *ParsedEdge) bool {
		if e.FromRef == fromNode.Ref && e.ToRef == toNode.Ref &&
			e.FromOutput == input.FromOutput && e.ToInput == input.ToInput {
			found = true
			return false
		}
		return true
	})

	if !found {
		return fmt.Errorf("connection not found: %s[%d] -> %s[%d]",
			fromNode.Name, input.FromOutput, toNode.Name, input.ToInput)
	}

	linkEdgesToNodes(pw)
	return nil
}

func UnsetNodePath(pw *ParsedWorkflow, ref string, path string) (*ParsedNode, error) {
	node, err := ResolveRef(pw, ref)
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(path, ".", 2)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty path")
	}

	switch parts[0] {
	case "parameters":
		if len(parts) == 2 {
			deleteNestedPath(node.Parameters, parts[1])
		} else {
			node.Parameters = make(map[string]interface{})
		}
	case "credentials":
		if len(parts) == 2 {
			deleteNestedPath(node.Credentials, parts[1])
		} else {
			node.Credentials = make(map[string]interface{})
		}
	case "notes":
		node.Notes = ""
	default:
		return nil, fmt.Errorf("cannot unset top-level field %q; use update instead", parts[0])
	}

	return node, nil
}

// --- internal helpers ---

func applyPatch(node *ParsedNode, patch NodePatch) error {
	parts := strings.SplitN(patch.Path, ".", 2)
	if len(parts) == 0 {
		return fmt.Errorf("empty patch path")
	}

	switch parts[0] {
	case "name":
		s, ok := patch.Value.(string)
		if !ok {
			return fmt.Errorf("name must be a string")
		}
		node.Name = s
	case "type":
		s, ok := patch.Value.(string)
		if !ok {
			return fmt.Errorf("type must be a string")
		}
		node.Type = s
	case "typeVersion":
		node.TypeVersion = int(toFloat64(patch.Value))
	case "position":
		if s, ok := patch.Value.(string); ok {
			var x, y float64
			if _, err := fmt.Sscanf(s, "%f,%f", &x, &y); err != nil {
				return fmt.Errorf("position must be x,y: %w", err)
			}
			node.Position[0] = x
			node.Position[1] = y
		}
	case "disabled":
		b, ok := patch.Value.(bool)
		if !ok {
			return fmt.Errorf("disabled must be a boolean")
		}
		node.Disabled = b
	case "notes":
		s, ok := patch.Value.(string)
		if !ok {
			return fmt.Errorf("notes must be a string")
		}
		node.Notes = s
	case "parameters":
		if len(parts) == 2 {
			setNestedPath(node.Parameters, parts[1], patch.Value)
		} else {
			if m, ok := patch.Value.(map[string]interface{}); ok {
				node.Parameters = m
			} else {
				return fmt.Errorf("parameters must be a JSON object")
			}
		}
	case "credentials":
		if len(parts) == 2 {
			setNestedPath(node.Credentials, parts[1], patch.Value)
		} else {
			if m, ok := patch.Value.(map[string]interface{}); ok {
				node.Credentials = m
			} else {
				return fmt.Errorf("credentials must be a JSON object")
			}
		}
	case "onError":
		s, ok := patch.Value.(string)
		if !ok {
			return fmt.Errorf("onError must be a string")
		}
		node.OnError = s
	default:
		return fmt.Errorf("unknown patch path: %q", parts[0])
	}

	return nil
}

func setNestedPath(m map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := m
	for i := 0; i < len(parts)-1; i++ {
		next, ok := current[parts[i]].(map[string]interface{})
		if !ok {
			next = make(map[string]interface{})
			current[parts[i]] = next
		}
		current = next
	}
	current[parts[len(parts)-1]] = value
}

func deleteNestedPath(m map[string]interface{}, path string) {
	parts := strings.Split(path, ".")
	current := m
	for i := 0; i < len(parts)-1; i++ {
		next, ok := current[parts[i]].(map[string]interface{})
		if !ok {
			return
		}
		current = next
	}
	delete(current, parts[len(parts)-1])
}

func deepMerge(dst, src map[string]interface{}) {
	for k, v := range src {
		if dstMap, ok := dst[k].(map[string]interface{}); ok {
			if srcMap, ok := v.(map[string]interface{}); ok {
				deepMerge(dstMap, srcMap)
				continue
			}
		}
		dst[k] = v
	}
}

func updateEdgeNames(pw *ParsedWorkflow, oldName, newName string) {
	for _, e := range pw.Edges {
		if e.FromName == oldName {
			e.FromName = newName
		}
		if e.ToName == oldName {
			e.ToName = newName
		}
	}
}

func filterEdges(edges []*ParsedEdge, keep func(*ParsedEdge) bool) []*ParsedEdge {
	result := make([]*ParsedEdge, 0, len(edges))
	for _, e := range edges {
		if keep(e) {
			result = append(result, e)
		}
	}
	return result
}

func filterNodes(nodes []*ParsedNode, keep func(*ParsedNode) bool) []*ParsedNode {
	result := make([]*ParsedNode, 0, len(nodes))
	for _, n := range nodes {
		if keep(n) {
			result = append(result, n)
		}
	}
	return result
}

func generateNodeName(pw *ParsedWorkflow, nodeType string) string {
	parts := strings.Split(nodeType, ".")
	base := parts[len(parts)-1]
	if strings.HasPrefix(base, "n8n-nodes-") {
		base = strings.TrimPrefix(base, "n8n-nodes-base.")
	}

	name := base
	for i := 1; ; i++ {
		if len(pw.Indexes.ByName[name]) == 0 {
			return name
		}
		name = fmt.Sprintf("%s %d", base, i)
	}
}

func ParseSetFlag(s string) (NodePatch, error) {
	eqIdx := strings.Index(s, "=")
	if eqIdx < 0 {
		return NodePatch{}, fmt.Errorf("invalid --set format %q: expected path=value", s)
	}
	path := s[:eqIdx]
	rawValue := s[eqIdx+1:]

	var value interface{}
	if err := json.Unmarshal([]byte(rawValue), &value); err != nil {
		value = rawValue
	}

	return NodePatch{Path: path, Value: value}, nil
}
