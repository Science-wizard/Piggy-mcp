package mcp

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"piggy-mcp/internal/expenses"
)

func TestServerHandlesInitializeToolsAndToolCalls(t *testing.T) {
	store, err := expenses.NewStore(filepath.Join(t.TempDir(), "expenses.db"))
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(store)

	response, ok := server.Handle([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`))
	if !ok {
		t.Fatal("expected initialize response")
	}
	if response["id"] != float64(1) {
		t.Fatalf("id = %#v, want 1", response["id"])
	}

	response, ok = server.Handle([]byte(`{"jsonrpc":"2.0","id":"tools","method":"tools/list","params":{}}`))
	if !ok {
		t.Fatal("expected tools/list response")
	}
	result := response["result"].(map[string]any)
	tools := result["tools"].([]map[string]any)
	if len(tools) != 12 {
		t.Fatalf("tool count = %d, want 12", len(tools))
	}

	response, ok = server.Handle([]byte(`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"add_expense","arguments":{"description":"Lunch","amount":12.75,"category":"Food","date":"2026-07-06"}}}`))
	if !ok {
		t.Fatal("expected tools/call response")
	}
	if _, exists := response["error"]; exists {
		t.Fatalf("unexpected error response: %#v", response)
	}
	if got := len(store.List()); got != 1 {
		t.Fatalf("stored expense count = %d, want 1", got)
	}

	expense := store.List()[0]
	response, ok = server.Handle([]byte(`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"edit_expense","arguments":{"id":"` + expense.ID + `","amount":13.5}}}`))
	if !ok {
		t.Fatal("expected edit_expense response")
	}
	if _, exists := response["error"]; exists {
		t.Fatalf("unexpected edit error response: %#v", response)
	}

	response, ok = server.Handle([]byte(`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"search_expenses","arguments":{"query":"Lunch"}}}`))
	if !ok {
		t.Fatal("expected search_expenses response")
	}
	if _, exists := response["error"]; exists {
		t.Fatalf("unexpected search error response: %#v", response)
	}
}

func TestServerServeUsesLineDelimitedJSON(t *testing.T) {
	store, err := expenses.NewStore(filepath.Join(t.TempDir(), "expenses.db"))
	if err != nil {
		t.Fatal(err)
	}
	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"get_summary","arguments":{}}}`,
		"",
	}, "\n")

	var output bytes.Buffer
	if err := NewServer(store).Serve(strings.NewReader(input), &output); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(output.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("response lines = %d, want 2; output: %s", len(lines), output.String())
	}
	for _, line := range lines {
		var decoded map[string]any
		if err := json.Unmarshal([]byte(line), &decoded); err != nil {
			t.Fatalf("invalid JSON response %q: %v", line, err)
		}
	}
}
