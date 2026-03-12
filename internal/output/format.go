package output

import (
	"encoding/json"
	"fmt"
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

func NodeSummary(n *parser.ParsedNode) string {
	var sb strings.Builder
	disabled := ""
	if n.Disabled {
		disabled = " (disabled)"
	}
	sb.WriteString(fmt.Sprintf("Node: %s [%s]%s\n", n.Name, n.Ref, disabled))
	sb.WriteString(fmt.Sprintf("  Type: %s (v%d)\n", n.Type, n.TypeVersion))
	sb.WriteString(fmt.Sprintf("  Position: [%.0f, %.0f]\n", n.Position[0], n.Position[1]))
	if n.ID != "" {
		sb.WriteString(fmt.Sprintf("  ID: %s\n", n.ID))
	}
	if len(n.Inbound) > 0 {
		sb.WriteString(fmt.Sprintf("  Inbound: %d connections\n", len(n.Inbound)))
	}
	if len(n.Outbound) > 0 {
		sb.WriteString(fmt.Sprintf("  Outbound: %d connections\n", len(n.Outbound)))
	}
	return sb.String()
}

func NodeResolved(n *parser.ParsedNode) string {
	var sb strings.Builder
	sb.WriteString(NodeSummary(n))
	if len(n.Parameters) > 0 {
		sb.WriteString("  Parameters:\n")
		for k, v := range n.Parameters {
			sb.WriteString(fmt.Sprintf("    %s: %v\n", k, v))
		}
	}
	if len(n.Credentials) > 0 {
		sb.WriteString("  Credentials:\n")
		for k, v := range n.Credentials {
			sb.WriteString(fmt.Sprintf("    %s: %v\n", k, v))
		}
	}
	if len(n.Inbound) > 0 {
		sb.WriteString("  Inbound connections:\n")
		for _, e := range n.Inbound {
			sb.WriteString(fmt.Sprintf("    <- %s [%s] (output %d -> input %d)\n", e.FromName, e.FromRef, e.FromOutput, e.ToInput))
		}
	}
	if len(n.Outbound) > 0 {
		sb.WriteString("  Outbound connections:\n")
		for _, e := range n.Outbound {
			sb.WriteString(fmt.Sprintf("    -> %s [%s] (output %d -> input %d)\n", e.ToName, e.ToRef, e.FromOutput, e.ToInput))
		}
	}
	return sb.String()
}

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

func strVal(m map[string]interface{}, key string) string {
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	return fmt.Sprintf("%v", v)
}
