package parser

func AnalyzeGraph(pw *ParsedWorkflow) *GraphAnalysis {
	g := &GraphAnalysis{
		NodeCount:     len(pw.Nodes),
		EdgeCount:     len(pw.Edges),
		AdjacencyList: make(map[string][]string),
	}

	inDegree := make(map[string]int)
	outDegree := make(map[string]int)
	for _, n := range pw.Nodes {
		inDegree[n.Ref] = 0
		outDegree[n.Ref] = 0
		g.AdjacencyList[n.Ref] = nil
	}

	for _, e := range pw.Edges {
		g.AdjacencyList[e.FromRef] = append(g.AdjacencyList[e.FromRef], e.ToRef)
		inDegree[e.ToRef]++
		outDegree[e.FromRef]++
	}

	for _, n := range pw.Nodes {
		if inDegree[n.Ref] == 0 && outDegree[n.Ref] == 0 {
			g.Orphans = append(g.Orphans, n.Ref)
		} else if inDegree[n.Ref] == 0 {
			g.Roots = append(g.Roots, n.Ref)
		}
		if outDegree[n.Ref] == 0 && inDegree[n.Ref] > 0 {
			g.Leaves = append(g.Leaves, n.Ref)
		}
		if outDegree[n.Ref] > 1 {
			g.BranchingPoints = append(g.BranchingPoints, n.Ref)
		}
	}

	order, hasCycles, cycleNodes := topologicalSort(pw.Nodes, g.AdjacencyList)
	g.HasCycles = hasCycles
	g.CycleNodes = cycleNodes
	if !hasCycles {
		g.TopologicalOrder = order
	}

	return g
}

func topologicalSort(nodes []*ParsedNode, adj map[string][]string) ([]string, bool, []string) {
	inDegree := make(map[string]int)
	for _, n := range nodes {
		inDegree[n.Ref] = 0
	}
	for _, neighbors := range adj {
		for _, to := range neighbors {
			inDegree[to]++
		}
	}

	var queue []string
	for _, n := range nodes {
		if inDegree[n.Ref] == 0 {
			queue = append(queue, n.Ref)
		}
	}

	var order []string
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		order = append(order, current)

		for _, neighbor := range adj[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(order) != len(nodes) {
		var cycleNodes []string
		for _, n := range nodes {
			if inDegree[n.Ref] > 0 {
				cycleNodes = append(cycleNodes, n.Ref)
			}
		}
		return nil, true, cycleNodes
	}

	return order, false, nil
}
