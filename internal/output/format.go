package output

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"n8n-cli/internal/parser"
)

type Mode string

const (
	ModeSummary  Mode = "summary"
	ModeResolved Mode = "resolved"
	ModeRaw      Mode = "raw"
)

func ParseMode(s string) Mode {
	switch strings.ToLower(s) {
	case "resolved":
		return ModeResolved
	case "raw":
		return ModeRaw
	default:
		return ModeSummary
	}
}

func JSON(v interface{}) string {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	return string(b)
}

// --- Workflow output ---

func WorkflowSummary(meta parser.WorkflowMeta) string {
	active := "inactive"
	if meta.Active {
		active = "active"
	}
	tags := ""
	if len(meta.Tags) > 0 {
		names := make([]string, len(meta.Tags))
		for i, t := range meta.Tags {
			names[i] = t.Name
		}
		tags = fmt.Sprintf(" | tags: %s", strings.Join(names, ", "))
	}
	return fmt.Sprintf("Workflow: %s (ID: %s, %s)%s", meta.Name, meta.ID, active, tags)
}

func WorkflowListSummary(workflows []map[string]interface{}) string {
	if len(workflows) == 0 {
		return "No workflows found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-6s %-40s %-8s %-20s\n", "ID", "NAME", "ACTIVE", "UPDATED"))
	sb.WriteString(strings.Repeat("-", 78) + "\n")
	for _, w := range workflows {
		id := strVal(w, "id")
		name := strVal(w, "name")
		active := strVal(w, "active")
		updated := strVal(w, "updatedAt")
		if len(name) > 38 {
			name = name[:35] + "..."
		}
		if len(updated) > 19 {
			updated = updated[:19]
		}
		sb.WriteString(fmt.Sprintf("%-6s %-40s %-8s %-20s\n", id, name, active, updated))
	}
	return sb.String()
}

func InspectSummary(pw *parser.ParsedWorkflow) string {
	var sb strings.Builder
	sb.WriteString(WorkflowSummary(pw.Meta) + "\n")
	sb.WriteString(fmt.Sprintf("Nodes: %d | Edges: %d\n", len(pw.Nodes), len(pw.Edges)))
	if len(pw.Nodes) > 0 {
		sb.WriteString(fmt.Sprintf("\n%-5s %-30s %-45s %-12s\n", "REF", "NAME", "TYPE", "POS"))
		sb.WriteString(strings.Repeat("-", 95) + "\n")
		for _, n := range pw.Nodes {
			name := n.Name
			if len(name) > 28 {
				name = name[:25] + "..."
			}
			typ := n.Type
			if len(typ) > 43 {
				typ = typ[:40] + "..."
			}
			disabled := ""
			if n.Disabled {
				disabled = " (disabled)"
			}
			sb.WriteString(fmt.Sprintf("%-5s %-30s %-45s [%.0f,%.0f]%s\n",
				n.Ref, name, typ, n.Position[0], n.Position[1], disabled))
		}
	}
	return sb.String()
}

// --- Node list output ---

func NodeListSummary(nodes []*parser.ParsedNode) string {
	if len(nodes) == 0 {
		return "No nodes found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-5s %-30s %-40s %-4s %-12s %-4s %-4s\n",
		"REF", "NAME", "TYPE", "VER", "POS", "IN", "OUT"))
	sb.WriteString(strings.Repeat("-", 103) + "\n")
	for _, n := range nodes {
		name := n.Name
		if len(name) > 28 {
			name = name[:25] + "..."
		}
		typ := n.Type
		if len(typ) > 38 {
			typ = typ[:35] + "..."
		}
		disabled := ""
		if n.Disabled {
			disabled = "*"
		}
		sb.WriteString(fmt.Sprintf("%-5s %-30s %-40s %-4d [%.0f,%.0f]%-4s %-4d %-4d\n",
			n.Ref+disabled, name, typ, n.TypeVersion,
			n.Position[0], n.Position[1], "",
			len(n.Inbound), len(n.Outbound)))
	}
	return sb.String()
}

// --- Node get view modes ---

// NodeViewSummary returns a compact, LLM-friendly summary of a single node.
// This is the default view when no --view flag is passed.
func NodeViewSummary(n *parser.ParsedNode) string {
	var sb strings.Builder

	disabled := ""
	if n.Disabled {
		disabled = " (disabled)"
	}

	sb.WriteString(fmt.Sprintf("Node: %s%s\n", n.Name, disabled))
	if n.ID != "" {
		sb.WriteString(fmt.Sprintf("  Ref: id:%s\n", n.ID))
	} else {
		sb.WriteString(fmt.Sprintf("  Ref: %s\n", n.Ref))
	}
	sb.WriteString(fmt.Sprintf("  Type: %s\n", n.Type))
	sb.WriteString(fmt.Sprintf("  TypeVersion: %d\n", n.TypeVersion))
	sb.WriteString(fmt.Sprintf("  Position: [%.0f, %.0f]\n", n.Position[0], n.Position[1]))
	sb.WriteString(fmt.Sprintf("  Disabled: %v\n", n.Disabled))

	if len(n.Parameters) > 0 {
		sb.WriteString("\n  Key parameters:\n")
		keys := sortedKeys(n.Parameters)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("    - %s: %s\n", k, formatValue(n.Parameters[k])))
		}
	}

	if len(n.Credentials) > 0 {
		sb.WriteString("\n  Credentials:\n")
		for k, v := range n.Credentials {
			if cm, ok := v.(map[string]interface{}); ok {
				parts := []string{}
				if id, ok := cm["id"]; ok {
					parts = append(parts, fmt.Sprintf("id=%v", id))
				}
				if name, ok := cm["name"]; ok {
					parts = append(parts, fmt.Sprintf("name=%q", name))
				}
				sb.WriteString(fmt.Sprintf("    - %s: %s\n", k, strings.Join(parts, ", ")))
			} else {
				sb.WriteString(fmt.Sprintf("    - %s: %v\n", k, v))
			}
		}
	}

	sb.WriteString("\n  Connections:\n")
	if len(n.Inbound) > 0 {
		for _, e := range n.Inbound {
			sb.WriteString(fmt.Sprintf("    - in: %s\n", e.FromName))
		}
	} else {
		sb.WriteString("    - in: none\n")
	}
	if len(n.Outbound) > 0 {
		for _, e := range n.Outbound {
			sb.WriteString(fmt.Sprintf("    - out: %s\n", e.ToName))
		}
	} else {
		sb.WriteString("    - out: none\n")
	}

	return sb.String()
}

// NodeViewDetails returns a structured, readable view that closely reflects native JSON structure.
func NodeViewDetails(n *parser.ParsedNode) string {
	var sb strings.Builder

	sb.WriteString("Node\n")
	if n.ID != "" {
		sb.WriteString(fmt.Sprintf("  id: %s\n", n.ID))
	}
	sb.WriteString(fmt.Sprintf("  name: %s\n", n.Name))
	sb.WriteString(fmt.Sprintf("  type: %s\n", n.Type))
	sb.WriteString(fmt.Sprintf("  typeVersion: %d\n", n.TypeVersion))
	sb.WriteString(fmt.Sprintf("  position: [%.0f, %.0f]\n", n.Position[0], n.Position[1]))
	sb.WriteString(fmt.Sprintf("  disabled: %v\n", n.Disabled))
	if n.Notes != "" {
		sb.WriteString(fmt.Sprintf("  notes: %s\n", n.Notes))
	}

	if len(n.Parameters) > 0 {
		sb.WriteString("\nParameters\n")
		printMapIndented(&sb, n.Parameters, "  ", 0)
	}

	if len(n.Credentials) > 0 {
		sb.WriteString("\nCredentials\n")
		printMapIndented(&sb, n.Credentials, "  ", 0)
	}

	sb.WriteString("\nConnections\n")
	sb.WriteString("  incoming:\n")
	if len(n.Inbound) > 0 {
		for _, e := range n.Inbound {
			sb.WriteString(fmt.Sprintf("    - %s [%s] output %d -> input %d\n", e.FromName, e.FromRef, e.FromOutput, e.ToInput))
		}
	} else {
		sb.WriteString("    - none\n")
	}
	sb.WriteString("  outgoing:\n")
	if len(n.Outbound) > 0 {
		for _, e := range n.Outbound {
			sb.WriteString(fmt.Sprintf("    - %s [%s] output %d -> input %d\n", e.ToName, e.ToRef, e.FromOutput, e.ToInput))
		}
	} else {
		sb.WriteString("    - none\n")
	}

	return sb.String()
}

// NodeViewJSON returns the exact native n8n node JSON, copy-paste safe.
func NodeViewJSON(n *parser.ParsedNode) string {
	native := parser.NodeToNativeJSON(n)
	return JSON(native)
}

// NodeViewParams returns only the parameters object.
func NodeViewParams(n *parser.ParsedNode) string {
	return JSON(n.Parameters)
}

// NodeViewConnections returns the node summary plus detailed connection info.
func NodeViewConnections(n *parser.ParsedNode) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Node: %s [%s]\n", n.Name, n.Ref))
	sb.WriteString(fmt.Sprintf("  Type: %s (v%d)\n\n", n.Type, n.TypeVersion))

	sb.WriteString("Incoming connections:\n")
	if len(n.Inbound) > 0 {
		for _, e := range n.Inbound {
			sb.WriteString(fmt.Sprintf("  %s [%s] output %d -> %s input %d\n",
				e.FromName, e.FromRef, e.FromOutput, n.Name, e.ToInput))
		}
	} else {
		sb.WriteString("  none\n")
	}

	sb.WriteString("\nOutgoing connections:\n")
	if len(n.Outbound) > 0 {
		for _, e := range n.Outbound {
			sb.WriteString(fmt.Sprintf("  %s output %d -> %s [%s] input %d\n",
				n.Name, e.FromOutput, e.ToName, e.ToRef, e.ToInput))
		}
	} else {
		sb.WriteString("  none\n")
	}

	return sb.String()
}

// NodeParamExtract extracts specific parameter paths from a node.
// Single path: returns just the value. Multiple paths: returns key-value pairs.
func NodeParamExtract(n *parser.ParsedNode, paths []string) string {
	if len(paths) == 1 {
		val, err := parser.ExtractPath(n, paths[0])
		if err != nil {
			return fmt.Sprintf("error: %s", err)
		}
		return formatExtractedValue(val)
	}

	result := make(map[string]interface{})
	for _, path := range paths {
		val, err := parser.ExtractPath(n, path)
		if err != nil {
			result[path] = fmt.Sprintf("<error: %s>", err)
		} else {
			result[path] = val
		}
	}
	return JSON(result)
}

// --- Node update diff ---

// NodeUpdateDiff formats a before/after update result.
func NodeUpdateDiff(n *parser.ParsedNode, changedPaths []string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Updated node: %s\n", n.Name))
	if n.ID != "" {
		sb.WriteString(fmt.Sprintf("  Ref: id:%s\n", n.ID))
	} else {
		sb.WriteString(fmt.Sprintf("  Ref: %s\n", n.Ref))
	}

	if len(changedPaths) > 0 {
		sb.WriteString("\n  Changed fields:\n")
		for _, p := range changedPaths {
			sb.WriteString(fmt.Sprintf("    - %s\n", p))
		}
	} else {
		sb.WriteString("\n  No fields changed.\n")
	}

	sb.WriteString(fmt.Sprintf("\n  Result:\n"))
	sb.WriteString(fmt.Sprintf("    type: %s\n", n.Type))
	sb.WriteString(fmt.Sprintf("    typeVersion: %d\n", n.TypeVersion))
	sb.WriteString(fmt.Sprintf("    position: [%.0f, %.0f]\n", n.Position[0], n.Position[1]))

	return sb.String()
}

// --- Legacy aliases for backward compatibility ---

// NodeSummary is the legacy compact summary. Use NodeViewSummary for the new format.
func NodeSummary(n *parser.ParsedNode) string {
	return NodeViewSummary(n)
}

// NodeResolved is the legacy resolved view. Use NodeViewDetails for the new format.
func NodeResolved(n *parser.ParsedNode) string {
	return NodeViewDetails(n)
}

// --- Edge output ---

func EdgeListSummary(edges []*parser.ParsedEdge) string {
	if len(edges) == 0 {
		return "No connections found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-5s %-25s %-8s %-5s %-25s %-8s\n",
		"FROM", "FROM_NAME", "OUTPUT", "TO", "TO_NAME", "INPUT"))
	sb.WriteString(strings.Repeat("-", 80) + "\n")
	for _, e := range edges {
		fromName := e.FromName
		if len(fromName) > 23 {
			fromName = fromName[:20] + "..."
		}
		toName := e.ToName
		if len(toName) > 23 {
			toName = toName[:20] + "..."
		}
		sb.WriteString(fmt.Sprintf("%-5s %-25s %-8d %-5s %-25s %-8d\n",
			e.FromRef, fromName, e.FromOutput, e.ToRef, toName, e.ToInput))
	}
	return sb.String()
}

// --- Graph output ---

func GraphSummary(g *parser.GraphAnalysis) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Graph: %d nodes, %d edges\n", g.NodeCount, g.EdgeCount))
	if g.HasCycles {
		sb.WriteString(fmt.Sprintf("  Cycles detected: %s\n", strings.Join(g.CycleNodes, ", ")))
	} else {
		sb.WriteString("  Acyclic: yes\n")
	}
	if len(g.Roots) > 0 {
		sb.WriteString(fmt.Sprintf("  Roots (entry points): %s\n", strings.Join(g.Roots, ", ")))
	}
	if len(g.Leaves) > 0 {
		sb.WriteString(fmt.Sprintf("  Leaves (endpoints): %s\n", strings.Join(g.Leaves, ", ")))
	}
	if len(g.Orphans) > 0 {
		sb.WriteString(fmt.Sprintf("  Orphans (disconnected): %s\n", strings.Join(g.Orphans, ", ")))
	}
	if len(g.BranchingPoints) > 0 {
		sb.WriteString(fmt.Sprintf("  Branching points: %s\n", strings.Join(g.BranchingPoints, ", ")))
	}
	if !g.HasCycles && len(g.TopologicalOrder) > 0 {
		sb.WriteString(fmt.Sprintf("  Topological order: %s\n", strings.Join(g.TopologicalOrder, " -> ")))
	}
	return sb.String()
}

// --- Execution output ---

func ExecutionListSummary(executions []map[string]interface{}) string {
	if len(executions) == 0 {
		return "No executions found."
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%-8s %-10s %-12s %-20s %-20s %-8s\n",
		"ID", "WORKFLOW", "STATUS", "STARTED", "FINISHED", "MODE"))
	sb.WriteString(strings.Repeat("-", 82) + "\n")
	for _, e := range executions {
		id := strVal(e, "id")
		wfID := ""
		if wf, ok := e["workflowId"]; ok {
			wfID = fmt.Sprintf("%v", wf)
		}
		status := strVal(e, "status")
		started := strVal(e, "startedAt")
		if len(started) > 19 {
			started = started[:19]
		}
		finished := strVal(e, "stoppedAt")
		if len(finished) > 19 {
			finished = finished[:19]
		}
		mode := strVal(e, "mode")
		sb.WriteString(fmt.Sprintf("%-8s %-10s %-12s %-20s %-20s %-8s\n",
			id, wfID, status, started, finished, mode))
	}
	return sb.String()
}

func ExecutionSummary(exec map[string]interface{}) string {
	var sb strings.Builder
	id := strVal(exec, "id")
	status := strVal(exec, "status")
	mode := strVal(exec, "mode")
	started := strVal(exec, "startedAt")
	finished := strVal(exec, "stoppedAt")
	wfID := ""
	if wf, ok := exec["workflowId"]; ok {
		wfID = fmt.Sprintf("%v", wf)
	}
	sb.WriteString(fmt.Sprintf("Execution: %s (workflow: %s)\n", id, wfID))
	sb.WriteString(fmt.Sprintf("  Status: %s\n", status))
	sb.WriteString(fmt.Sprintf("  Mode: %s\n", mode))
	sb.WriteString(fmt.Sprintf("  Started: %s\n", started))
	sb.WriteString(fmt.Sprintf("  Finished: %s\n", finished))
	return sb.String()
}

// --- internal helpers ---

func strVal(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case bool:
		return fmt.Sprintf("%v", val)
	case float64:
		if val == float64(int(val)) {
			return fmt.Sprintf("%.0f", val)
		}
		return fmt.Sprintf("%v", val)
	case map[string]interface{}, []interface{}:
		b, _ := json.Marshal(val)
		s := string(b)
		if len(s) > 80 {
			return s[:77] + "..."
		}
		return s
	default:
		return fmt.Sprintf("%v", v)
	}
}

func formatExtractedValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case map[string]interface{}, []interface{}:
		return JSON(val)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func printMapIndented(sb *strings.Builder, m map[string]interface{}, indent string, depth int) {
	keys := sortedKeys(m)
	for _, k := range keys {
		v := m[k]
		switch val := v.(type) {
		case map[string]interface{}:
			sb.WriteString(fmt.Sprintf("%s%s:\n", indent, k))
			printMapIndented(sb, val, indent+"  ", depth+1)
		case []interface{}:
			sb.WriteString(fmt.Sprintf("%s%s:\n", indent, k))
			for i, item := range val {
				if itemMap, ok := item.(map[string]interface{}); ok {
					sb.WriteString(fmt.Sprintf("%s  [%d]:\n", indent, i))
					printMapIndented(sb, itemMap, indent+"    ", depth+2)
				} else {
					sb.WriteString(fmt.Sprintf("%s  - %s\n", indent, formatValue(item)))
				}
			}
		default:
			sb.WriteString(fmt.Sprintf("%s%s: %s\n", indent, k, formatValue(v)))
		}
	}
}
