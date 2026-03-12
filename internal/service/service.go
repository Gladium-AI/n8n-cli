package service

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"n8n-cli/internal/client"
	"n8n-cli/internal/parser"
)

type Service struct {
	Client *client.Client
}

func New(c *client.Client) *Service {
	return &Service{Client: c}
}

// --- Workflow operations ---

func (s *Service) CreateWorkflow(name string, filePath string, fromStdin bool, active bool, tags []string) (map[string]interface{}, error) {
	var body map[string]interface{}

	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read workflow file: %w", err)
		}
		if err := json.Unmarshal(data, &body); err != nil {
			return nil, fmt.Errorf("parse workflow file: %w", err)
		}
	} else if fromStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		if err := json.Unmarshal(data, &body); err != nil {
			return nil, fmt.Errorf("parse stdin: %w", err)
		}
	} else {
		body = map[string]interface{}{
			"nodes":       []interface{}{},
			"connections": map[string]interface{}{},
			"settings":    map[string]interface{}{},
		}
	}

	if name != "" {
		body["name"] = name
	}
	if _, ok := body["name"]; !ok {
		return nil, fmt.Errorf("workflow name is required: use --name or include in JSON")
	}

	if active {
		body["active"] = true
	}

	return s.Client.CreateWorkflow(body)
}

func (s *Service) ListWorkflows(active *bool, tags []string, name string, limit int, cursor string, all bool) ([]map[string]interface{}, error) {
	if !all {
		return s.listWorkflowsPage(active, tags, name, limit, cursor)
	}

	var allWorkflows []map[string]interface{}
	cur := cursor
	for {
		items, nextCur, err := s.Client.ListWorkflows(active, tags, name, limit, cur)
		if err != nil {
			return nil, err
		}
		allWorkflows = append(allWorkflows, items...)
		if nextCur == "" {
			break
		}
		cur = nextCur
	}
	return allWorkflows, nil
}

func (s *Service) listWorkflowsPage(active *bool, tags []string, name string, limit int, cursor string) ([]map[string]interface{}, error) {
	items, _, err := s.Client.ListWorkflows(active, tags, name, limit, cursor)
	return items, err
}

func (s *Service) GetWorkflow(id string) (map[string]interface{}, error) {
	return s.Client.GetWorkflow(id)
}

func (s *Service) UpdateWorkflow(id string, filePath string, fromStdin bool, sets []string, patchFile string) (map[string]interface{}, error) {
	raw, err := s.Client.GetWorkflow(id)
	if err != nil {
		return nil, err
	}

	if filePath != "" {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read file: %w", err)
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse file: %w", err)
		}
	} else if fromStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("parse stdin: %w", err)
		}
	}

	if patchFile != "" {
		patchData, err := os.ReadFile(patchFile)
		if err != nil {
			return nil, fmt.Errorf("read patch file: %w", err)
		}
		var patch map[string]interface{}
		if err := json.Unmarshal(patchData, &patch); err != nil {
			return nil, fmt.Errorf("parse patch file: %w", err)
		}
		for k, v := range patch {
			raw[k] = v
		}
	}

	for _, setExpr := range sets {
		eqIdx := strings.Index(setExpr, "=")
		if eqIdx < 0 {
			return nil, fmt.Errorf("invalid --set format %q: expected path=value", setExpr)
		}
		path := setExpr[:eqIdx]
		rawValue := setExpr[eqIdx+1:]
		var value interface{}
		if err := json.Unmarshal([]byte(rawValue), &value); err != nil {
			value = rawValue
		}
		setWorkflowPath(raw, path, value)
	}

	return s.Client.UpdateWorkflow(id, raw)
}

func (s *Service) DeleteWorkflow(id string) error {
	return s.Client.DeleteWorkflow(id)
}

func (s *Service) ActivateWorkflow(id string) (map[string]interface{}, error) {
	return s.Client.ActivateWorkflow(id)
}

func (s *Service) DeactivateWorkflow(id string) (map[string]interface{}, error) {
	return s.Client.DeactivateWorkflow(id)
}

func (s *Service) InspectWorkflow(id string) (*parser.ParsedWorkflow, error) {
	raw, err := s.Client.GetWorkflow(id)
	if err != nil {
		return nil, err
	}
	return parser.Parse(raw)
}

// --- Execution operations ---

func (s *Service) ListExecutions(workflowID, status string, limit int, cursor string, all bool) ([]map[string]interface{}, error) {
	if !all {
		items, _, err := s.Client.ListExecutions(workflowID, status, limit, cursor)
		return items, err
	}

	var allExecs []map[string]interface{}
	cur := cursor
	for {
		items, nextCur, err := s.Client.ListExecutions(workflowID, status, limit, cur)
		if err != nil {
			return nil, err
		}
		allExecs = append(allExecs, items...)
		if nextCur == "" {
			break
		}
		cur = nextCur
	}
	return allExecs, nil
}

func (s *Service) GetExecution(id string, withData bool) (map[string]interface{}, error) {
	return s.Client.GetExecution(id, withData)
}

func (s *Service) DeleteExecution(id string) error {
	return s.Client.DeleteExecution(id)
}

func (s *Service) RetryExecution(id string) (map[string]interface{}, error) {
	return s.Client.RetryExecution(id)
}

func (s *Service) StopExecution(id string) (map[string]interface{}, error) {
	return s.Client.StopExecution(id)
}

// --- Node operations ---

func (s *Service) fetchAndParse(workflowID string) (*parser.ParsedWorkflow, error) {
	raw, err := s.Client.GetWorkflow(workflowID)
	if err != nil {
		return nil, err
	}
	return parser.Parse(raw)
}

func (s *Service) saveWorkflow(pw *parser.ParsedWorkflow) (map[string]interface{}, error) {
	rehydrated := parser.Rehydrate(pw)
	return s.Client.UpdateWorkflow(pw.Meta.ID, rehydrated)
}

func (s *Service) ListNodes(workflowID string) ([]*parser.ParsedNode, error) {
	pw, err := s.fetchAndParse(workflowID)
	if err != nil {
		return nil, err
	}
	return pw.Nodes, nil
}

func (s *Service) GetNode(workflowID, nodeRef string) (*parser.ParsedNode, error) {
	pw, err := s.fetchAndParse(workflowID)
	if err != nil {
		return nil, err
	}
	return parser.ResolveRef(pw, nodeRef)
}

func (s *Service) CreateNode(workflowID string, input parser.NodeInput, connectFrom, connectTo string) (*parser.ParsedNode, bool, error) {
	pw, err := s.fetchAndParse(workflowID)
	if err != nil {
		return nil, false, err
	}

	node, err := parser.AddNode(pw, input)
	if err != nil {
		return nil, false, err
	}

	if connectFrom != "" {
		parts := strings.SplitN(connectFrom, ":", 2)
		fromRef := parts[0]
		fromOutput := 0
		if len(parts) == 2 {
			fmt.Sscanf(parts[1], "%d", &fromOutput)
		}
		_, err := parser.AddEdge(pw, parser.EdgeInput{
			FromRef:    fromRef,
			FromOutput: fromOutput,
			ToRef:      node.Ref,
			ToInput:    0,
		})
		if err != nil {
			return nil, false, fmt.Errorf("connect-from: %w", err)
		}
	}

	if connectTo != "" {
		parts := strings.SplitN(connectTo, ":", 2)
		toRef := parts[0]
		toInput := 0
		if len(parts) == 2 {
			fmt.Sscanf(parts[1], "%d", &toInput)
		}
		_, err := parser.AddEdge(pw, parser.EdgeInput{
			FromRef:    node.Ref,
			FromOutput: 0,
			ToRef:      toRef,
			ToInput:    toInput,
		})
		if err != nil {
			return nil, false, fmt.Errorf("connect-to: %w", err)
		}
	}

	_, err = s.saveWorkflow(pw)
	if err != nil {
		return nil, false, err
	}

	return node, false, nil
}

func (s *Service) CreateNodeDryRun(workflowID string, input parser.NodeInput, connectFrom, connectTo string) (*parser.ParsedNode, error) {
	pw, err := s.fetchAndParse(workflowID)
	if err != nil {
		return nil, err
	}
	node, err := parser.AddNode(pw, input)
	if err != nil {
		return nil, err
	}
	return node, nil
}

func (s *Service) UpdateNode(workflowID, nodeRef string, patches []parser.NodePatch, unsets []string, mergeFile, replaceFile, patchFile string, rename string, moveX, moveY *float64, enable, disable bool, dryRun bool) (*parser.ParsedNode, error) {
	pw, err := s.fetchAndParse(workflowID)
	if err != nil {
		return nil, err
	}

	var node *parser.ParsedNode

	if replaceFile != "" {
		data, err := os.ReadFile(replaceFile)
		if err != nil {
			return nil, fmt.Errorf("read replace file: %w", err)
		}
		var replacement map[string]interface{}
		if err := json.Unmarshal(data, &replacement); err != nil {
			return nil, fmt.Errorf("parse replace file: %w", err)
		}
		node, err = parser.ReplaceNode(pw, nodeRef, replacement)
		if err != nil {
			return nil, err
		}
	} else if mergeFile != "" {
		data, err := os.ReadFile(mergeFile)
		if err != nil {
			return nil, fmt.Errorf("read merge file: %w", err)
		}
		var merge map[string]interface{}
		if err := json.Unmarshal(data, &merge); err != nil {
			return nil, fmt.Errorf("parse merge file: %w", err)
		}
		node, err = parser.UpdateNodeMerge(pw, nodeRef, merge)
		if err != nil {
			return nil, err
		}
	} else if patchFile != "" {
		data, err := os.ReadFile(patchFile)
		if err != nil {
			return nil, fmt.Errorf("read patch file: %w", err)
		}
		var patchMap map[string]interface{}
		if err := json.Unmarshal(data, &patchMap); err != nil {
			return nil, fmt.Errorf("parse patch file: %w", err)
		}
		node, err = parser.UpdateNodeMerge(pw, nodeRef, patchMap)
		if err != nil {
			return nil, err
		}
	}

	if len(patches) > 0 {
		node, err = parser.UpdateNodePatches(pw, nodeRef, patches)
		if err != nil {
			return nil, err
		}
	}

	for _, unsetPath := range unsets {
		node, err = parser.UnsetNodePath(pw, nodeRef, unsetPath)
		if err != nil {
			return nil, err
		}
	}

	if rename != "" {
		node, err = parser.RenameNode(pw, nodeRef, rename)
		if err != nil {
			return nil, err
		}
	}

	if moveX != nil && moveY != nil {
		node, err = parser.MoveNode(pw, nodeRef, *moveX, *moveY)
		if err != nil {
			return nil, err
		}
	}

	if enable {
		node, err = parser.EnableNode(pw, nodeRef)
		if err != nil {
			return nil, err
		}
	}
	if disable {
		node, err = parser.DisableNode(pw, nodeRef)
		if err != nil {
			return nil, err
		}
	}

	if node == nil {
		node, err = parser.ResolveRef(pw, nodeRef)
		if err != nil {
			return nil, err
		}
	}

	if !dryRun {
		_, err = s.saveWorkflow(pw)
		if err != nil {
			return nil, err
		}
	}

	return node, nil
}

func (s *Service) DeleteNode(workflowID, nodeRef string, opts parser.DeleteOptions, dryRun bool) error {
	pw, err := s.fetchAndParse(workflowID)
	if err != nil {
		return err
	}

	if err := parser.RemoveNode(pw, nodeRef, opts); err != nil {
		return err
	}

	if !dryRun {
		_, err = s.saveWorkflow(pw)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) RenameNode(workflowID, nodeRef, newName string, dryRun bool) (*parser.ParsedNode, error) {
	pw, err := s.fetchAndParse(workflowID)
	if err != nil {
		return nil, err
	}

	node, err := parser.RenameNode(pw, nodeRef, newName)
	if err != nil {
		return nil, err
	}

	if !dryRun {
		_, err = s.saveWorkflow(pw)
		if err != nil {
			return nil, err
		}
	}

	return node, nil
}

func (s *Service) MoveNode(workflowID, nodeRef string, x, y float64, dryRun bool) (*parser.ParsedNode, error) {
	pw, err := s.fetchAndParse(workflowID)
	if err != nil {
		return nil, err
	}

	node, err := parser.MoveNode(pw, nodeRef, x, y)
	if err != nil {
		return nil, err
	}

	if !dryRun {
		_, err = s.saveWorkflow(pw)
		if err != nil {
			return nil, err
		}
	}

	return node, nil
}

func (s *Service) EnableNode(workflowID, nodeRef string, dryRun bool) (*parser.ParsedNode, error) {
	pw, err := s.fetchAndParse(workflowID)
	if err != nil {
		return nil, err
	}

	node, err := parser.EnableNode(pw, nodeRef)
	if err != nil {
		return nil, err
	}

	if !dryRun {
		_, err = s.saveWorkflow(pw)
		if err != nil {
			return nil, err
		}
	}

	return node, nil
}

func (s *Service) DisableNode(workflowID, nodeRef string, dryRun bool) (*parser.ParsedNode, error) {
	pw, err := s.fetchAndParse(workflowID)
	if err != nil {
		return nil, err
	}

	node, err := parser.DisableNode(pw, nodeRef)
	if err != nil {
		return nil, err
	}

	if !dryRun {
		_, err = s.saveWorkflow(pw)
		if err != nil {
			return nil, err
		}
	}

	return node, nil
}

// --- Connection operations ---

func (s *Service) ListConnections(workflowID, nodeRef, direction string) ([]*parser.ParsedEdge, error) {
	pw, err := s.fetchAndParse(workflowID)
	if err != nil {
		return nil, err
	}

	if nodeRef == "" {
		return pw.Edges, nil
	}

	node, err := parser.ResolveRef(pw, nodeRef)
	if err != nil {
		return nil, err
	}

	var edges []*parser.ParsedEdge
	switch direction {
	case "in":
		edges = node.Inbound
	case "out":
		edges = node.Outbound
	default:
		edges = append(edges, node.Inbound...)
		edges = append(edges, node.Outbound...)
	}
	return edges, nil
}

func (s *Service) CreateConnection(workflowID string, input parser.EdgeInput, dryRun bool) (*parser.ParsedEdge, error) {
	pw, err := s.fetchAndParse(workflowID)
	if err != nil {
		return nil, err
	}

	edge, err := parser.AddEdge(pw, input)
	if err != nil {
		return nil, err
	}

	if !dryRun {
		_, err = s.saveWorkflow(pw)
		if err != nil {
			return nil, err
		}
	}

	return edge, nil
}

func (s *Service) DeleteConnection(workflowID string, input parser.EdgeInput, dryRun bool) error {
	pw, err := s.fetchAndParse(workflowID)
	if err != nil {
		return err
	}

	if err := parser.RemoveEdge(pw, input); err != nil {
		return err
	}

	if !dryRun {
		_, err = s.saveWorkflow(pw)
		if err != nil {
			return err
		}
	}

	return nil
}

// --- Graph operations ---

func (s *Service) InspectGraph(workflowID string) (*parser.ParsedWorkflow, *parser.GraphAnalysis, error) {
	pw, err := s.fetchAndParse(workflowID)
	if err != nil {
		return nil, nil, err
	}
	analysis := parser.AnalyzeGraph(pw)
	return pw, analysis, nil
}

// --- Test operations ---

func (s *Service) WebhookTest(workflowID string, payloadFile string, fromStdin bool, method string, headers map[string]string, useTestURL bool) (int, []byte, error) {
	pw, err := s.fetchAndParse(workflowID)
	if err != nil {
		return 0, nil, err
	}

	var webhookNode *parser.ParsedNode
	for _, n := range pw.Nodes {
		if strings.Contains(strings.ToLower(n.Type), "webhook") {
			webhookNode = n
			break
		}
	}
	if webhookNode == nil {
		return 0, nil, fmt.Errorf("no webhook node found in workflow %s", workflowID)
	}

	webhookPath := ""
	if p, ok := webhookNode.Parameters["path"].(string); ok {
		webhookPath = p
	}
	if webhookPath == "" {
		return 0, nil, fmt.Errorf("webhook node has no path parameter")
	}

	baseURL := s.Client.BaseURL()
	var url string
	if useTestURL {
		url = fmt.Sprintf("%s/webhook-test/%s", baseURL, webhookPath)
	} else {
		url = fmt.Sprintf("%s/webhook/%s", baseURL, webhookPath)
	}

	var payload []byte
	if payloadFile != "" {
		data, err := os.ReadFile(payloadFile)
		if err != nil {
			return 0, nil, fmt.Errorf("read payload file: %w", err)
		}
		payload = data
	} else if fromStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return 0, nil, fmt.Errorf("read stdin: %w", err)
		}
		payload = data
	}

	if method == "" {
		if httpMethod, ok := webhookNode.Parameters["httpMethod"].(string); ok {
			method = httpMethod
		} else {
			method = "POST"
		}
	}

	return s.Client.SendWebhook(url, method, headers, payload)
}

// --- helpers ---

func setWorkflowPath(raw map[string]interface{}, path string, value interface{}) {
	parts := strings.Split(path, ".")
	current := raw
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
