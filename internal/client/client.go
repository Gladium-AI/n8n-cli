package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
)

type Client struct {
	http    *resty.Client
	baseURL string
}

func New(baseURL, apiKey string) *Client {
	base := strings.TrimRight(baseURL, "/")
	r := resty.New().
		SetBaseURL(base+"/api/v1").
		SetHeader("Accept", "application/json").
		SetHeader("Content-Type", "application/json").
		SetHeader("X-N8N-API-KEY", apiKey)

	return &Client{http: r, baseURL: base}
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

type APIError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("n8n API error %d: %s", e.StatusCode, e.Message)
}

func checkResponse(resp *resty.Response) error {
	if resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
		return nil
	}
	msg := resp.Status()
	body := string(resp.Body())
	var parsed map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &parsed); err == nil {
		if m, ok := parsed["message"]; ok {
			msg = fmt.Sprintf("%v", m)
		}
	}
	return &APIError{StatusCode: resp.StatusCode(), Message: msg, Body: body}
}

// --- Workflow endpoints ---

func (c *Client) ListWorkflows(active *bool, tags []string, name string, limit int, cursor string) ([]map[string]interface{}, string, error) {
	req := c.http.R()
	if active != nil {
		req.SetQueryParam("active", fmt.Sprintf("%v", *active))
	}
	for _, t := range tags {
		req.SetQueryParam("tags", t)
	}
	if name != "" {
		req.SetQueryParam("name", name)
	}
	if limit > 0 {
		req.SetQueryParam("limit", fmt.Sprintf("%d", limit))
	}
	if cursor != "" {
		req.SetQueryParam("cursor", cursor)
	}

	resp, err := req.Get("/workflows")
	if err != nil {
		return nil, "", fmt.Errorf("list workflows: %w", err)
	}
	if err := checkResponse(resp); err != nil {
		return nil, "", err
	}

	return parsePaginatedResponse(resp.Body())
}

func (c *Client) GetWorkflow(id string) (map[string]interface{}, error) {
	resp, err := c.http.R().Get("/workflows/" + id)
	if err != nil {
		return nil, fmt.Errorf("get workflow: %w", err)
	}
	if err := checkResponse(resp); err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("parse workflow response: %w", err)
	}
	return result, nil
}

func (c *Client) CreateWorkflow(body map[string]interface{}) (map[string]interface{}, error) {
	resp, err := c.http.R().SetBody(body).Post("/workflows")
	if err != nil {
		return nil, fmt.Errorf("create workflow: %w", err)
	}
	if err := checkResponse(resp); err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("parse workflow response: %w", err)
	}
	return result, nil
}

// workflowWritableFields are the only top-level properties we send to
// PUT /workflows/{id}. Keep this intentionally minimal and aligned with
// the documented workflow object to avoid version-specific 400 errors
// caused by extra fields appearing in GET responses.
var workflowWritableFields = map[string]bool{
	"name":        true,
	"nodes":       true,
	"connections": true,
	"settings":    true,
	"staticData":  true,
	"pinData":     true,
	"versionId":   true,
}

// sanitizeWorkflowBody returns a new map containing only the properties
// that the n8n API will accept in a PUT request.
func sanitizeWorkflowBody(body map[string]interface{}) map[string]interface{} {
	clean := make(map[string]interface{}, len(workflowWritableFields))
	for k, v := range body {
		if workflowWritableFields[k] {
			clean[k] = v
		}
	}
	return clean
}

func (c *Client) UpdateWorkflow(id string, body map[string]interface{}) (map[string]interface{}, error) {
	clean := sanitizeWorkflowBody(body)
	resp, err := c.http.R().SetBody(clean).Put("/workflows/" + id)
	if err != nil {
		return nil, fmt.Errorf("update workflow: %w", err)
	}
	if err := checkResponse(resp); err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("parse workflow response: %w", err)
	}
	return result, nil
}

func (c *Client) DeleteWorkflow(id string) error {
	resp, err := c.http.R().Delete("/workflows/" + id)
	if err != nil {
		return fmt.Errorf("delete workflow: %w", err)
	}
	return checkResponse(resp)
}

func (c *Client) ActivateWorkflow(id string) (map[string]interface{}, error) {
	resp, err := c.http.R().Post("/workflows/" + id + "/activate")
	if err != nil {
		return nil, fmt.Errorf("activate workflow: %w", err)
	}
	if err := checkResponse(resp); err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("parse workflow response: %w", err)
	}
	return result, nil
}

func (c *Client) DeactivateWorkflow(id string) (map[string]interface{}, error) {
	resp, err := c.http.R().Post("/workflows/" + id + "/deactivate")
	if err != nil {
		return nil, fmt.Errorf("deactivate workflow: %w", err)
	}
	if err := checkResponse(resp); err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("parse workflow response: %w", err)
	}
	return result, nil
}

// --- Execution endpoints ---

func (c *Client) ListExecutions(workflowID string, status string, limit int, cursor string) ([]map[string]interface{}, string, error) {
	req := c.http.R()
	if workflowID != "" {
		req.SetQueryParam("workflowId", workflowID)
	}
	if status != "" {
		req.SetQueryParam("status", status)
	}
	if limit > 0 {
		req.SetQueryParam("limit", fmt.Sprintf("%d", limit))
	}
	if cursor != "" {
		req.SetQueryParam("cursor", cursor)
	}

	resp, err := req.Get("/executions")
	if err != nil {
		return nil, "", fmt.Errorf("list executions: %w", err)
	}
	if err := checkResponse(resp); err != nil {
		return nil, "", err
	}

	return parsePaginatedResponse(resp.Body())
}

func (c *Client) GetExecution(id string, includeData bool) (map[string]interface{}, error) {
	req := c.http.R()
	if includeData {
		req.SetQueryParam("includeData", "true")
	}
	resp, err := req.Get("/executions/" + id)
	if err != nil {
		return nil, fmt.Errorf("get execution: %w", err)
	}
	if err := checkResponse(resp); err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("parse execution response: %w", err)
	}
	return result, nil
}

func (c *Client) DeleteExecution(id string) error {
	resp, err := c.http.R().Delete("/executions/" + id)
	if err != nil {
		return fmt.Errorf("delete execution: %w", err)
	}
	return checkResponse(resp)
}

func (c *Client) RetryExecution(id string) (map[string]interface{}, error) {
	resp, err := c.http.R().Post("/executions/" + id + "/retry")
	if err != nil {
		return nil, fmt.Errorf("retry execution: %w", err)
	}
	if err := checkResponse(resp); err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("parse retry response: %w", err)
	}
	return result, nil
}

func (c *Client) StopExecution(id string) (map[string]interface{}, error) {
	resp, err := c.http.R().Post("/executions/" + id + "/stop")
	if err != nil {
		return nil, fmt.Errorf("stop execution: %w", err)
	}
	if err := checkResponse(resp); err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("parse stop response: %w", err)
	}
	return result, nil
}

// --- Webhook helper ---

func (c *Client) SendWebhook(url string, method string, headers map[string]string, payload []byte) (int, []byte, error) {
	r := resty.New()
	req := r.R()
	for k, v := range headers {
		req.SetHeader(k, v)
	}
	if payload != nil {
		req.SetBody(payload)
		if _, hasContentType := headers["Content-Type"]; !hasContentType {
			req.SetHeader("Content-Type", "application/json")
		}
	}

	var resp *resty.Response
	var err error
	switch strings.ToUpper(method) {
	case http.MethodGet:
		resp, err = req.Get(url)
	case http.MethodPost:
		resp, err = req.Post(url)
	case http.MethodPut:
		resp, err = req.Put(url)
	case http.MethodPatch:
		resp, err = req.Patch(url)
	case http.MethodDelete:
		resp, err = req.Delete(url)
	default:
		return 0, nil, fmt.Errorf("unsupported HTTP method: %s", method)
	}
	if err != nil {
		return 0, nil, fmt.Errorf("webhook request: %w", err)
	}
	return resp.StatusCode(), resp.Body(), nil
}

// --- Helpers ---

func parsePaginatedResponse(body []byte) ([]map[string]interface{}, string, error) {
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, "", fmt.Errorf("parse paginated response: %w", err)
	}

	var items []map[string]interface{}
	if data, ok := raw["data"]; ok {
		if arr, ok := data.([]interface{}); ok {
			for _, item := range arr {
				if m, ok := item.(map[string]interface{}); ok {
					items = append(items, m)
				}
			}
		}
	}

	nextCursor := ""
	if nc, ok := raw["nextCursor"]; ok && nc != nil {
		nextCursor = fmt.Sprintf("%v", nc)
	}

	return items, nextCursor, nil
}
