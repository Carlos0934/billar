package mcphttp

import (
	"fmt"
	"testing"
)

func TestBearerTokenFromHeader(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header string
		want   string
	}{
		{name: "extracts bearer token", header: "Bearer token-123", want: "token-123"},
		{name: "accepts lowercase scheme", header: "bearer token-123", want: "token-123"},
		{name: "rejects non bearer scheme", header: "Basic token-123", want: ""},
		{name: "rejects missing token", header: "Bearer   ", want: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := bearerTokenFromHeader(tc.header); got != tc.want {
				t.Fatalf("bearerTokenFromHeader() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestPeekJSONRPCMethod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body []byte
		want string
	}{
		{name: "valid JSON with method", body: []byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`), want: "initialize"},
		{name: "valid JSON with different method", body: []byte(`{"jsonrpc":"2.0","method":"tools/call"}`), want: "tools/call"},
		{name: "malformed JSON", body: []byte(`{invalid json}`), want: ""},
		{name: "JSON array (batch)", body: []byte(`[{"jsonrpc":"2.0","method":"initialize"}]`), want: ""},
		{name: "missing method field", body: []byte(`{"jsonrpc":"2.0","id":1}`), want: ""},
		{name: "empty body", body: []byte(``), want: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := peekJSONRPCMethod(tc.body); got != tc.want {
				t.Fatalf("peekJSONRPCMethod(%q) = %q, want %q", string(tc.body), got, tc.want)
			}
		})
	}
}

func TestClassifyJSONRPCMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name              string
		body              string
		wantType          JSONRPCMessageType
		wantMethod        string
		wantHasID         bool
		wantBodyParseable bool
	}{
		// Request cases (has method and id)
		{name: "request with method and id", body: `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{}}`, wantType: JSONRPCMessageRequest, wantMethod: "tools/call", wantHasID: true, wantBodyParseable: true},
		{name: "request with string id", body: `{"jsonrpc":"2.0","id":"abc","method":"initialize","params":{}}`, wantType: JSONRPCMessageRequest, wantMethod: "initialize", wantHasID: true, wantBodyParseable: true},

		// Notification cases (has method, no id)
		{name: "notification with method no id", body: `{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`, wantType: JSONRPCMessageNotification, wantMethod: "notifications/initialized", wantHasID: false, wantBodyParseable: true},

		// Response cases (has result or error)
		{name: "response with result", body: `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}}`, wantType: JSONRPCMessageResponse, wantMethod: "", wantHasID: true, wantBodyParseable: true},
		{name: "response with error", body: `{"jsonrpc":"2.0","id":1,"error":{"code":-32600,"message":"Invalid Request"}}`, wantType: JSONRPCMessageResponse, wantMethod: "", wantHasID: true, wantBodyParseable: true},
		{name: "response with result and no id", body: `{"jsonrpc":"2.0","result":{}}`, wantType: JSONRPCMessageResponse, wantMethod: "", wantHasID: false, wantBodyParseable: true},

		// Batch cases (array)
		{name: "batch request array", body: `[{"jsonrpc":"2.0","id":1,"method":"tools/call"},{"jsonrpc":"2.0","id":2,"method":"resources/list"}]`, wantType: JSONRPCMessageBatch, wantMethod: "", wantHasID: false, wantBodyParseable: true},
		{name: "empty batch array", body: `[]`, wantType: JSONRPCMessageBatch, wantMethod: "", wantHasID: false, wantBodyParseable: true},
		{name: "batch with whitespace", body: `  [ {"jsonrpc":"2.0"} ]  `, wantType: JSONRPCMessageBatch, wantMethod: "", wantHasID: false, wantBodyParseable: true},

		// Invalid cases
		{name: "malformed JSON", body: `{invalid json}`, wantType: JSONRPCMessageInvalid, wantMethod: "", wantHasID: false, wantBodyParseable: false},
		{name: "empty body", body: ``, wantType: JSONRPCMessageInvalid, wantMethod: "", wantHasID: false, wantBodyParseable: false},

		// Unknown cases (valid JSON but no recognizable fields)
		{name: "unknown with only jsonrpc field", body: `{"jsonrpc":"2.0"}`, wantType: JSONRPCMessageUnknown, wantMethod: "", wantHasID: false, wantBodyParseable: true},
		{name: "unknown with only id field", body: `{"jsonrpc":"2.0","id":1}`, wantType: JSONRPCMessageUnknown, wantMethod: "", wantHasID: true, wantBodyParseable: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			info := classifyJSONRPCMessage([]byte(tc.body))
			if info.Type != tc.wantType {
				t.Fatalf("classifyJSONRPCMessage(%q).Type = %q, want %q", tc.body, info.Type, tc.wantType)
			}
			if info.Method != tc.wantMethod {
				t.Fatalf("classifyJSONRPCMessage(%q).Method = %q, want %q", tc.body, info.Method, tc.wantMethod)
			}
			if info.HasID != tc.wantHasID {
				t.Fatalf("classifyJSONRPCMessage(%q).HasID = %v, want %v", tc.body, info.HasID, tc.wantHasID)
			}
			if info.BodyParseable != tc.wantBodyParseable {
				t.Fatalf("classifyJSONRPCMessage(%q).BodyParseable = %v, want %v", tc.body, info.BodyParseable, tc.wantBodyParseable)
			}
		})
	}
}

func TestIsArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		body string
		want bool
	}{
		{body: `[]`, want: true},
		{body: `  []`, want: true},
		{body: "\n\t[]", want: true},
		{body: `[{"jsonrpc":"2.0"}]`, want: true},
		{body: `{}`, want: false},
		{body: `{"jsonrpc":"2.0"}`, want: false},
		{body: ``, want: false},
		{body: `"test"`, want: false},
		{body: `  {}`, want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.body, func(t *testing.T) {
			t.Parallel()
			if got := isArray([]byte(tc.body)); got != tc.want {
				t.Fatalf("isArray(%q) = %v, want %v", tc.body, got, tc.want)
			}
		})
	}
}

// TestClassifyJSONRPCMessageTypes verifies all message type constants are correctly named.
func TestClassifyJSONRPCMessageTypes(t *testing.T) {
	t.Parallel()

	types := []JSONRPCMessageType{
		JSONRPCMessageRequest,
		JSONRPCMessageNotification,
		JSONRPCMessageResponse,
		JSONRPCMessageBatch,
		JSONRPCMessageInvalid,
		JSONRPCMessageUnknown,
	}
	for _, mt := range types {
		if string(mt) == "" {
			t.Fatalf("JSONRPCMessageType constant is empty: %v", mt)
		}
		_ = fmt.Sprintf("%s", mt)
	}
}
