// Package mcp implements an MCP (Model Context Protocol) stdio server for oz.
//
// The server speaks JSON-RPC 2.0 over stdin/stdout with newline-delimited
// framing (one JSON object per line). It exposes four tools that wrap the
// oz context query engine and structural graph:
//
//   - query_graph     — route a task description to the right agent
//   - get_node        — retrieve a single graph node by ID
//   - get_neighbors   — list nodes adjacent to a given node
//   - agent_for_task  — shorthand: task → agent name + confidence only
//
// # Protocol flow
//
//  1. Client sends initialize request
//  2. Server responds with capabilities and serverInfo
//  3. Client sends notifications/initialized (no response)
//  4. Client calls tools/list or tools/call
//
// The supported protocol version is 2024-11-05.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"

	ozcontext "github.com/joaoajmatos/oz/internal/context"
	"github.com/joaoajmatos/oz/internal/graph"
	"github.com/joaoajmatos/oz/internal/query"
	"github.com/joaoajmatos/oz/internal/semantic"
)

const protocolVersion = "2024-11-05"

// Server is an MCP stdio server bound to an oz workspace.
type Server struct {
	workspacePath string
	out           io.Writer
}

// New creates a new Server for the oz workspace at workspacePath.
func New(workspacePath string) *Server {
	return &Server{workspacePath: workspacePath, out: os.Stdout}
}

// SetOutput redirects server responses to w instead of os.Stdout.
// Used by tests to capture output without subprocess overhead.
func (s *Server) SetOutput(w io.Writer) {
	s.out = w
}

// Serve reads JSON-RPC messages from r and writes responses to the server's
// output (defaulting to os.Stdout). It returns when r is exhausted or a fatal
// read error occurs.
func (s *Server) Serve(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024) // 4 MiB per line
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		s.handleLine(line)
	}
	return scanner.Err()
}

// --- JSON-RPC types ----------------------------------------------------------

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"` // number, string, or null
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Standard JSON-RPC 2.0 error codes.
const (
	errParse          = -32700
	errMethodNotFound = -32601
	errInvalidParams  = -32602
	errInternal       = -32603
)

// --- Dispatch ----------------------------------------------------------------

func (s *Server) handleLine(line []byte) {
	var req request
	if err := json.Unmarshal(line, &req); err != nil {
		s.writeError(nil, errParse, "parse error: "+err.Error())
		return
	}

	// Notifications have no id — dispatch but do not respond.
	isNotification := req.ID == nil || string(req.ID) == "null"

	switch req.Method {
	case "initialize":
		if isNotification {
			return
		}
		s.handleInitialize(req)
	case "notifications/initialized":
		// No response required.
		return
	case "tools/list":
		if isNotification {
			return
		}
		s.handleToolsList(req)
	case "tools/call":
		if isNotification {
			return
		}
		s.handleToolsCall(req)
	default:
		if !isNotification {
			s.writeError(req.ID, errMethodNotFound, "method not found: "+req.Method)
		}
	}
}

// --- initialize --------------------------------------------------------------

type initParams struct {
	ProtocolVersion string      `json:"protocolVersion"`
	Capabilities    interface{} `json:"capabilities"`
	ClientInfo      interface{} `json:"clientInfo"`
}

type initResult struct {
	ProtocolVersion string      `json:"protocolVersion"`
	Capabilities    capabResult `json:"capabilities"`
	ServerInfo      serverInfo  `json:"serverInfo"`
}

type capabResult struct {
	Tools map[string]interface{} `json:"tools"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func (s *Server) handleInitialize(req request) {
	// Print staleness warning to stderr if semantic.json exists and is stale.
	s.checkStaleness()

	result := initResult{
		ProtocolVersion: protocolVersion,
		Capabilities:    capabResult{Tools: map[string]interface{}{}},
		ServerInfo:      serverInfo{Name: "oz", Version: "1.0.0"},
	}
	s.writeResult(req.ID, result)
}

// checkStaleness warns on stderr when context/semantic.json is stale.
func (s *Server) checkStaleness() {
	g, err := ozcontext.LoadGraph(s.workspacePath)
	if err != nil {
		return
	}
	o, err := semantic.Load(s.workspacePath)
	if err != nil || o == nil {
		return
	}
	if semantic.IsStale(o, g.ContentHash) {
		fmt.Fprintln(os.Stderr, "warning: semantic overlay may be stale — run 'oz context enrich' to update")
	}
}

// --- tools/list --------------------------------------------------------------

type toolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema inputSchema `json:"inputSchema"`
}

type inputSchema struct {
	Type       string              `json:"type"`
	Properties map[string]schemaProp `json:"properties"`
	Required   []string            `json:"required,omitempty"`
}

type schemaProp struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type toolsListResult struct {
	Tools []toolDef `json:"tools"`
}

func (s *Server) handleToolsList(req request) {
	s.writeResult(req.ID, toolsListResult{Tools: allTools()})
}

func allTools() []toolDef {
	return []toolDef{
		{
			Name:        "query_graph",
			Description: "Route a task description to the best-matching agent. Returns a full routing packet with agent name, confidence score, relevant context blocks, and scope paths.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"task": {Type: "string", Description: "A natural-language description of the task to route."},
				},
				Required: []string{"task"},
			},
		},
		{
			Name:        "get_node",
			Description: "Retrieve a single structural graph node by its ID (e.g. 'agent:coding', 'spec_section:specs/api.md:Overview').",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"id": {Type: "string", Description: "The node ID to look up."},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "get_neighbors",
			Description: "Return all nodes adjacent to a given node, optionally filtered by edge type (reads, owns, references, supports, crystallized_from).",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"id":        {Type: "string", Description: "The node ID whose neighbours to list."},
					"edge_type": {Type: "string", Description: "Optional edge type filter (e.g. 'owns'). Omit to return all neighbours."},
				},
				Required: []string{"id"},
			},
		},
		{
			Name:        "agent_for_task",
			Description: "Shorthand routing: given a task description return only the agent name and confidence score. Lower token cost than query_graph when only routing is needed.",
			InputSchema: inputSchema{
				Type: "object",
				Properties: map[string]schemaProp{
					"task": {Type: "string", Description: "A natural-language description of the task."},
				},
				Required: []string{"task"},
			},
		},
	}
}

// --- tools/call --------------------------------------------------------------

type toolsCallParams struct {
	Name      string                     `json:"name"`
	Arguments map[string]json.RawMessage `json:"arguments"`
}

type toolsCallResult struct {
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (s *Server) handleToolsCall(req request) {
	var p toolsCallParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		s.writeError(req.ID, errInvalidParams, "invalid params: "+err.Error())
		return
	}

	var result interface{}
	var callErr error

	switch p.Name {
	case "query_graph":
		result, callErr = s.toolQueryGraph(p.Arguments)
	case "get_node":
		result, callErr = s.toolGetNode(p.Arguments)
	case "get_neighbors":
		result, callErr = s.toolGetNeighbors(p.Arguments)
	case "agent_for_task":
		result, callErr = s.toolAgentForTask(p.Arguments)
	default:
		s.writeError(req.ID, errMethodNotFound, "unknown tool: "+p.Name)
		return
	}

	if callErr != nil {
		s.writeError(req.ID, errInternal, callErr.Error())
		return
	}

	text, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		s.writeError(req.ID, errInternal, "marshal result: "+err.Error())
		return
	}
	s.writeResult(req.ID, toolsCallResult{
		Content: []contentBlock{{Type: "text", Text: string(text)}},
	})
}

// --- tool implementations ----------------------------------------------------

func (s *Server) toolQueryGraph(args map[string]json.RawMessage) (interface{}, error) {
	task, err := stringArg(args, "task")
	if err != nil {
		return nil, err
	}
	return query.Run(s.workspacePath, task), nil
}

func (s *Server) toolGetNode(args map[string]json.RawMessage) (interface{}, error) {
	id, err := stringArg(args, "id")
	if err != nil {
		return nil, err
	}
	g, err := s.loadOrBuildGraph()
	if err != nil {
		return nil, err
	}
	for _, n := range g.Nodes {
		if n.ID == id {
			return n, nil
		}
	}
	return nil, fmt.Errorf("node not found: %s", id)
}

type neighborResult struct {
	Node     graph.Node `json:"node"`
	EdgeType string     `json:"edge_type"`
	// Direction is "outbound" (node is From) or "inbound" (node is To).
	Direction string `json:"direction"`
}

func (s *Server) toolGetNeighbors(args map[string]json.RawMessage) (interface{}, error) {
	id, err := stringArg(args, "id")
	if err != nil {
		return nil, err
	}
	edgeType, _ := stringArg(args, "edge_type") // optional

	g, err := s.loadOrBuildGraph()
	if err != nil {
		return nil, err
	}

	// Index nodes for fast lookup.
	nodeByID := make(map[string]graph.Node, len(g.Nodes))
	for _, n := range g.Nodes {
		nodeByID[n.ID] = n
	}

	var results []neighborResult
	for _, e := range g.Edges {
		if edgeType != "" && e.Type != edgeType {
			continue
		}
		if e.From == id {
			if n, ok := nodeByID[e.To]; ok {
				results = append(results, neighborResult{Node: n, EdgeType: e.Type, Direction: "outbound"})
			}
		} else if e.To == id {
			if n, ok := nodeByID[e.From]; ok {
				results = append(results, neighborResult{Node: n, EdgeType: e.Type, Direction: "inbound"})
			}
		}
	}
	return results, nil
}

type agentForTaskResult struct {
	Agent      string  `json:"agent"`
	Confidence float64 `json:"confidence"`
	Reason     string  `json:"reason,omitempty"`
}

func (s *Server) toolAgentForTask(args map[string]json.RawMessage) (interface{}, error) {
	task, err := stringArg(args, "task")
	if err != nil {
		return nil, err
	}
	r := query.Run(s.workspacePath, task)
	return agentForTaskResult{
		Agent:      r.Agent,
		Confidence: r.Confidence,
		Reason:     r.Reason,
	}, nil
}

// --- helpers -----------------------------------------------------------------

// loadOrBuildGraph loads context/graph.json, building it on-the-fly if absent.
func (s *Server) loadOrBuildGraph() (*graph.Graph, error) {
	g, err := ozcontext.LoadGraph(s.workspacePath)
	if err == nil {
		return g, nil
	}
	result, buildErr := ozcontext.Build(s.workspacePath)
	if buildErr != nil {
		return nil, fmt.Errorf("load graph: %w (build also failed: %v)", err, buildErr)
	}
	return result.Graph, nil
}

func stringArg(args map[string]json.RawMessage, key string) (string, error) {
	raw, ok := args[key]
	if !ok {
		return "", fmt.Errorf("missing required argument: %s", key)
	}
	var v string
	if err := json.Unmarshal(raw, &v); err != nil {
		return "", fmt.Errorf("argument %q must be a string: %w", key, err)
	}
	return v, nil
}

func (s *Server) writeResult(id json.RawMessage, result interface{}) {
	s.write(response{JSONRPC: "2.0", ID: id, Result: result})
}

func (s *Server) writeError(id json.RawMessage, code int, msg string) {
	s.write(response{JSONRPC: "2.0", ID: id, Error: &rpcError{Code: code, Message: msg}})
}

func (s *Server) write(r response) {
	data, err := json.Marshal(r)
	if err != nil {
		// Last resort: write a minimal error to stdout.
		fmt.Fprintf(s.out, `{"jsonrpc":"2.0","error":{"code":%d,"message":"marshal error"}}`, errInternal)
		fmt.Fprintln(s.out)
		return
	}
	fmt.Fprintln(s.out, string(data))
}
