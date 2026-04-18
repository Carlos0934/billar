package mcphttp

import (
	"encoding/json"
	"strings"
)

// JSONRPCMessageType represents the classified shape of a JSON-RPC message.
type JSONRPCMessageType string

const (
	// JSONRPCMessageRequest is a request with method and id.
	JSONRPCMessageRequest JSONRPCMessageType = "request"
	// JSONRPCMessageNotification is a notification with method but no id.
	JSONRPCMessageNotification JSONRPCMessageType = "notification"
	// JSONRPCMessageResponse is a response with result or error.
	JSONRPCMessageResponse JSONRPCMessageType = "response"
	// JSONRPCMessageBatch is a batch request (array of messages).
	JSONRPCMessageBatch JSONRPCMessageType = "batch"
	// JSONRPCMessageInvalid is malformed or unparseable JSON.
	JSONRPCMessageInvalid JSONRPCMessageType = "invalid"
	// JSONRPCMessageUnknown is valid JSON that doesn't match known shapes.
	JSONRPCMessageUnknown JSONRPCMessageType = "unknown"
)

// JSONRPCMessageInfo contains classified information about a JSON-RPC message.
type JSONRPCMessageInfo struct {
	Type          JSONRPCMessageType
	Method        string
	HasID         bool
	HasResult     bool
	HasError      bool
	BodyParseable bool
}

type jsonrpcRequestPeek struct {
	Method string          `json:"method"`
	ID     json.RawMessage `json:"id"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

// classifyJSONRPCMessage analyzes the body and returns message classification.
func classifyJSONRPCMessage(body []byte) JSONRPCMessageInfo {
	info := JSONRPCMessageInfo{
		Type:          JSONRPCMessageUnknown,
		BodyParseable: false,
	}

	if len(body) == 0 {
		info.Type = JSONRPCMessageInvalid
		return info
	}

	// Check for batch (array)
	if isArray(body) {
		info.Type = JSONRPCMessageBatch
		info.BodyParseable = true
		return info
	}

	var req jsonrpcRequestPeek
	if err := json.Unmarshal(body, &req); err != nil {
		info.Type = JSONRPCMessageInvalid
		return info
	}

	info.Method = req.Method
	info.HasID = len(req.ID) > 0
	info.HasResult = len(req.Result) > 0
	info.HasError = len(req.Error) > 0
	info.BodyParseable = true

	// Classify based on presence of fields
	if info.HasResult || info.HasError {
		// Response has result or error
		info.Type = JSONRPCMessageResponse
	} else if info.Method != "" {
		// Has method
		if info.HasID {
			info.Type = JSONRPCMessageRequest
		} else {
			info.Type = JSONRPCMessageNotification
		}
	} else {
		// Valid JSON but no recognizable fields
		info.Type = JSONRPCMessageUnknown
	}

	return info
}

// isArray checks if the body starts with '[' (ignoring whitespace).
func isArray(body []byte) bool {
	for _, b := range body {
		if b == ' ' || b == '\t' || b == '\n' || b == '\r' {
			continue
		}
		return b == '['
	}
	return false
}

func peekJSONRPCMethod(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var req jsonrpcRequestPeek
	if err := json.Unmarshal(body, &req); err != nil {
		return ""
	}
	return req.Method
}

func bearerTokenFromHeader(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	tokenType, token, found := strings.Cut(value, " ")
	if !found || !strings.EqualFold(strings.TrimSpace(tokenType), "Bearer") {
		return ""
	}

	return strings.TrimSpace(token)
}
