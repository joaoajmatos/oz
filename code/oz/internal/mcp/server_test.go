package mcp_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/joaoajmatos/oz/internal/mcp"
	"github.com/joaoajmatos/oz/internal/testws"
)

// rpcMsg is a minimal JSON-RPC 2.0 message used in tests.
type rpcMsg struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  interface{}     `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   json.RawMessage `json:"error,omitempty"`
}

// session drives an MCP server for a single test.
type session struct {
	srv    *mcp.Server
	input  *bytes.Buffer
	output *bytes.Buffer
	nextID int
}

func newSession(t *testing.T) *session {
	t.Helper()
	ws := testws.New(t).
		WithAgent("backend",
			testws.Scope("code/api/**"),
			testws.Role("Builds REST endpoints"),
		).
		WithAgent("frontend",
			testws.Scope("code/ui/**"),
			testws.Role("Builds React components"),
		).
		Build()

	var out bytes.Buffer
	srv := mcp.New(ws.Path())
	srv.SetOutput(&out)

	return &session{srv: srv, input: &bytes.Buffer{}, output: &out, nextID: 1}
}

// send appends a JSON-RPC message to the input buffer.
func (s *session) send(method string, params interface{}) int {
	id := s.nextID
	s.nextID++
	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
	data, _ := json.Marshal(msg)
	s.input.Write(data)
	s.input.WriteByte('\n')
	return id
}

// notify appends a JSON-RPC notification (no id) to the input buffer.
func (s *session) notify(method string, params interface{}) {
	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	data, _ := json.Marshal(msg)
	s.input.Write(data)
	s.input.WriteByte('\n')
}

// run processes all buffered input and returns parsed responses keyed by id.
func (s *session) run(t *testing.T) map[int]rpcMsg {
	t.Helper()
	if err := s.srv.Serve(s.input); err != nil {
		t.Fatalf("server error: %v", err)
	}
	responses := make(map[int]rpcMsg)
	sc := bufio.NewScanner(s.output)
	for sc.Scan() {
		line := sc.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		var msg rpcMsg
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			t.Fatalf("parse server output %q: %v", line, err)
		}
		if msg.ID != nil {
			id := int(msg.ID.(float64))
			responses[id] = msg
		}
	}
	return responses
}

// TestMCPProtocol_Initialize verifies capability negotiation (S6-05).
func TestMCPProtocol_Initialize(t *testing.T) {
	s := newSession(t)
	id := s.send("initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]interface{}{"name": "test", "version": "0"},
	})
	s.notify("notifications/initialized", map[string]interface{}{})

	responses := s.run(t)
	resp, ok := responses[id]
	if !ok {
		t.Fatal("no response to initialize")
	}
	if resp.Error != nil {
		t.Fatalf("initialize error: %s", resp.Error)
	}

	var result struct {
		ProtocolVersion string `json:"protocolVersion"`
		ServerInfo      struct {
			Name string `json:"name"`
		} `json:"serverInfo"`
		Capabilities struct {
			Tools interface{} `json:"tools"`
		} `json:"capabilities"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal initialize result: %v", err)
	}
	if result.ProtocolVersion != "2024-11-05" {
		t.Errorf("protocolVersion = %q, want 2024-11-05", result.ProtocolVersion)
	}
	if result.ServerInfo.Name != "oz" {
		t.Errorf("serverInfo.name = %q, want oz", result.ServerInfo.Name)
	}
	if result.Capabilities.Tools == nil {
		t.Error("capabilities.tools must be present")
	}
}

// TestMCPProtocol_ToolsList verifies tools/list returns all four tools (S6-06–09).
func TestMCPProtocol_ToolsList(t *testing.T) {
	s := newSession(t)
	s.send("initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
	})
	s.notify("notifications/initialized", nil)
	listID := s.send("tools/list", map[string]interface{}{})

	responses := s.run(t)
	resp, ok := responses[listID]
	if !ok {
		t.Fatal("no response to tools/list")
	}
	if resp.Error != nil {
		t.Fatalf("tools/list error: %s", resp.Error)
	}

	var result struct {
		Tools []struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			InputSchema json.RawMessage `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal tools/list result: %v", err)
	}

	wantTools := map[string]bool{
		"query_graph":    false,
		"get_node":       false,
		"get_neighbors":  false,
		"agent_for_task": false,
	}
	for _, tool := range result.Tools {
		wantTools[tool.Name] = true
		if tool.Description == "" {
			t.Errorf("tool %q has empty description", tool.Name)
		}
		if len(tool.InputSchema) == 0 {
			t.Errorf("tool %q has no inputSchema", tool.Name)
		}
	}
	for name, found := range wantTools {
		if !found {
			t.Errorf("expected tool %q not found in tools/list", name)
		}
	}
}

// TestMCPTool_QueryGraph verifies the query_graph tool returns a valid routing
// packet (S6-06).
func TestMCPTool_QueryGraph(t *testing.T) {
	s := newSession(t)
	s.send("initialize", map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": nil})
	s.notify("notifications/initialized", nil)
	callID := s.send("tools/call", map[string]interface{}{
		"name":      "query_graph",
		"arguments": map[string]interface{}{"task": "build a REST endpoint"},
	})

	responses := s.run(t)
	resp, ok := responses[callID]
	if !ok {
		t.Fatal("no response to tools/call query_graph")
	}
	if resp.Error != nil {
		t.Fatalf("query_graph error: %s", resp.Error)
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal tools/call result: %v", err)
	}
	if len(result.Content) == 0 {
		t.Fatal("tools/call result has no content")
	}
	if result.Content[0].Type != "text" {
		t.Errorf("content[0].type = %q, want text", result.Content[0].Type)
	}
	// Verify the text is valid JSON with an agent field.
	var packet struct {
		Agent string `json:"agent"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &packet); err != nil {
		t.Fatalf("routing packet is not valid JSON: %v\n%s", err, result.Content[0].Text)
	}
}

// TestMCPTool_AgentForTask verifies the agent_for_task tool (S6-09).
func TestMCPTool_AgentForTask(t *testing.T) {
	s := newSession(t)
	s.send("initialize", map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": nil})
	s.notify("notifications/initialized", nil)
	callID := s.send("tools/call", map[string]interface{}{
		"name":      "agent_for_task",
		"arguments": map[string]interface{}{"task": "build a REST endpoint"},
	})

	responses := s.run(t)
	resp := responses[callID]
	if resp.Error != nil {
		t.Fatalf("agent_for_task error: %s", resp.Error)
	}

	var result struct {
		Content []struct{ Text string `json:"text"` } `json:"content"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var out struct {
		Agent      string  `json:"agent"`
		Confidence float64 `json:"confidence"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &out); err != nil {
		t.Fatalf("parse agent_for_task output: %v", err)
	}
	if out.Confidence < 0 || out.Confidence > 1 {
		t.Errorf("confidence %f out of [0,1]", out.Confidence)
	}
}

// TestMCPTool_GetNode verifies the get_node tool (S6-07).
func TestMCPTool_GetNode(t *testing.T) {
	s := newSession(t)
	s.send("initialize", map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": nil})
	s.notify("notifications/initialized", nil)

	// Request an agent node that must exist.
	callID := s.send("tools/call", map[string]interface{}{
		"name":      "get_node",
		"arguments": map[string]interface{}{"id": "agent:backend"},
	})

	responses := s.run(t)
	resp := responses[callID]
	if resp.Error != nil {
		t.Fatalf("get_node error: %s", resp.Error)
	}

	var result struct {
		Content []struct{ Text string `json:"text"` } `json:"content"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	var node struct {
		ID   string `json:"id"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &node); err != nil {
		t.Fatalf("parse node JSON: %v", err)
	}
	if node.ID != "agent:backend" {
		t.Errorf("node.id = %q, want agent:backend", node.ID)
	}
	if node.Type != "agent" {
		t.Errorf("node.type = %q, want agent", node.Type)
	}
}

// TestMCPTool_GetNeighbors verifies the get_neighbors tool (S6-08).
func TestMCPTool_GetNeighbors(t *testing.T) {
	s := newSession(t)
	s.send("initialize", map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": nil})
	s.notify("notifications/initialized", nil)
	callID := s.send("tools/call", map[string]interface{}{
		"name":      "get_neighbors",
		"arguments": map[string]interface{}{"id": "agent:backend"},
	})

	responses := s.run(t)
	resp := responses[callID]
	if resp.Error != nil {
		t.Fatalf("get_neighbors error: %s", resp.Error)
	}

	var result struct {
		Content []struct{ Text string `json:"text"` } `json:"content"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// The result should be a JSON array (possibly empty).
	var neighbors []interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &neighbors); err != nil {
		t.Fatalf("get_neighbors result is not a JSON array: %v\n%s", err, result.Content[0].Text)
	}
}

// TestMCPProtocol_UnknownMethod verifies method-not-found error handling.
func TestMCPProtocol_UnknownMethod(t *testing.T) {
	s := newSession(t)
	s.send("initialize", map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": nil})
	s.notify("notifications/initialized", nil)
	callID := s.send("nonexistent/method", nil)

	responses := s.run(t)
	resp := responses[callID]
	if resp.Error == nil {
		t.Fatal("expected error for unknown method, got none")
	}
	var rpcErr struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(resp.Error, &rpcErr); err != nil {
		t.Fatalf("parse error field: %v", err)
	}
	if rpcErr.Code != -32601 {
		t.Errorf("error code = %d, want -32601 (method not found)", rpcErr.Code)
	}
}

// --- Edge cases --------------------------------------------------------------

// TestMCPTool_GetNode_NotFound verifies that get_node returns an internal error
// (not a panic) when the requested node does not exist.
func TestMCPTool_GetNode_NotFound(t *testing.T) {
	s := newSession(t)
	s.send("initialize", map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": nil})
	s.notify("notifications/initialized", nil)
	callID := s.send("tools/call", map[string]interface{}{
		"name":      "get_node",
		"arguments": map[string]interface{}{"id": "agent:does-not-exist"},
	})

	responses := s.run(t)
	resp := responses[callID]
	if resp.Error == nil {
		t.Fatal("expected error for missing node, got none")
	}
	var rpcErr struct{ Code int `json:"code"` }
	if err := json.Unmarshal(resp.Error, &rpcErr); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if rpcErr.Code != -32603 {
		t.Errorf("error code = %d, want -32603 (internal error)", rpcErr.Code)
	}
}

// TestMCPTool_UnknownTool verifies that tools/call with an unknown tool name
// returns method-not-found (-32601).
func TestMCPTool_UnknownTool(t *testing.T) {
	s := newSession(t)
	s.send("initialize", map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": nil})
	s.notify("notifications/initialized", nil)
	callID := s.send("tools/call", map[string]interface{}{
		"name":      "no_such_tool",
		"arguments": map[string]interface{}{},
	})

	responses := s.run(t)
	resp := responses[callID]
	if resp.Error == nil {
		t.Fatal("expected error for unknown tool, got none")
	}
	var rpcErr struct{ Code int `json:"code"` }
	json.Unmarshal(resp.Error, &rpcErr) //nolint
	if rpcErr.Code != -32601 {
		t.Errorf("error code = %d, want -32601 (method not found)", rpcErr.Code)
	}
}

// TestMCPTool_MissingRequiredArgument verifies that tools/call without a
// required argument returns invalid-params (-32602).
func TestMCPTool_MissingRequiredArgument(t *testing.T) {
	s := newSession(t)
	s.send("initialize", map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": nil})
	s.notify("notifications/initialized", nil)
	// query_graph requires "task" — omit it.
	callID := s.send("tools/call", map[string]interface{}{
		"name":      "query_graph",
		"arguments": map[string]interface{}{},
	})

	responses := s.run(t)
	resp := responses[callID]
	if resp.Error == nil {
		t.Fatal("expected error for missing argument, got none")
	}
	var rpcErr struct{ Code int `json:"code"` }
	json.Unmarshal(resp.Error, &rpcErr) //nolint
	if rpcErr.Code != -32603 {
		t.Errorf("error code = %d, want -32603 (internal wrapping missing-arg)", rpcErr.Code)
	}
}

// TestMCPTool_GetNeighbors_EdgeTypeFilter verifies that the optional edge_type
// filter narrows results to only edges of that type.
func TestMCPTool_GetNeighbors_EdgeTypeFilter(t *testing.T) {
	s := newSession(t)
	s.send("initialize", map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": nil})
	s.notify("notifications/initialized", nil)

	// With a non-existent edge type there should be zero neighbours.
	callID := s.send("tools/call", map[string]interface{}{
		"name": "get_neighbors",
		"arguments": map[string]interface{}{
			"id":        "agent:backend",
			"edge_type": "crystallized_from", // no such edges in the test fixture
		},
	})

	responses := s.run(t)
	resp := responses[callID]
	if resp.Error != nil {
		t.Fatalf("get_neighbors with edge_type filter returned error: %s", resp.Error)
	}

	var result struct {
		Content []struct{ Text string `json:"text"` } `json:"content"`
	}
	json.Unmarshal(resp.Result, &result) //nolint
	var neighbors []interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &neighbors); err != nil {
		t.Fatalf("result is not a JSON array: %v", err)
	}
	if len(neighbors) != 0 {
		t.Errorf("expected 0 neighbours for unused edge type, got %d", len(neighbors))
	}
}

// TestMCPProtocol_MalformedJSON verifies that the server returns a parse error
// (-32700) without crashing when given invalid JSON.
func TestMCPProtocol_MalformedJSON(t *testing.T) {
	ws := testws.New(t).
		WithAgent("backend", testws.Scope("code/api/**"), testws.Role("Builds REST endpoints")).
		Build()

	var out bytes.Buffer
	srv := mcp.New(ws.Path())
	srv.SetOutput(&out)

	if err := srv.Serve(strings.NewReader("{this is not valid json}\n")); err != nil {
		t.Fatalf("Serve returned unexpected error: %v", err)
	}

	// Scan the raw output — the response has a null id so it won't appear in
	// the keyed responses map; we scan the buffer directly instead.
	outStr := out.String()
	var found bool
	sc := bufio.NewScanner(strings.NewReader(outStr))
	for sc.Scan() {
		var msg rpcMsg
		if err := json.Unmarshal(sc.Bytes(), &msg); err != nil {
			continue
		}
		if msg.Error != nil {
			var e struct{ Code int `json:"code"` }
			json.Unmarshal(msg.Error, &e) //nolint
			if e.Code == -32700 {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected parse-error (-32700) response, got output: %q", outStr)
	}
}

// TestMCPTool_AgentForTask_NoOwner verifies that agent_for_task returns a
// reason field when no agent clearly owns the task.
func TestMCPTool_AgentForTask_NoOwner(t *testing.T) {
	s := newSession(t)
	s.send("initialize", map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": nil})
	s.notify("notifications/initialized", nil)
	// A completely unrelated query that won't match either agent.
	callID := s.send("tools/call", map[string]interface{}{
		"name":      "agent_for_task",
		"arguments": map[string]interface{}{"task": "the"},
	})

	responses := s.run(t)
	resp := responses[callID]
	if resp.Error != nil {
		t.Fatalf("agent_for_task error: %s", resp.Error)
	}
	var result struct {
		Content []struct{ Text string `json:"text"` } `json:"content"`
	}
	json.Unmarshal(resp.Result, &result) //nolint

	// Must return valid JSON regardless of routing outcome.
	var out struct {
		Agent      string  `json:"agent"`
		Confidence float64 `json:"confidence"`
		Reason     string  `json:"reason"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &out); err != nil {
		t.Fatalf("agent_for_task response is not valid JSON: %v", err)
	}
	if out.Confidence < 0 || out.Confidence > 1 {
		t.Errorf("confidence %f out of [0,1]", out.Confidence)
	}
}
