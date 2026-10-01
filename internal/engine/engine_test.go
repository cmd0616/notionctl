package engine

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/radityajayantara/notionctl/internal/config"
	"github.com/radityajayantara/notionctl/internal/notion"
	"github.com/radityajayantara/notionctl/internal/state"

	// Register property types for tests.
	_ "github.com/radityajayantara/notionctl/internal/property"
)

func makeConfig(t *testing.T, yaml string) *config.Config {
	t.Helper()
	cfg, err := config.Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}
	return cfg
}

func TestPlan_NewDatabases(t *testing.T) {
	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Projects
    parent_page_id: "page1"
    properties:
      Name:
        type: title
      Desc:
        type: rich_text
  - name: Tasks
    parent_page_id: "page2"
    properties:
      Name:
        type: title
      project:
        type: relation
        relation: Projects
`)

	st := &state.State{Version: "1", Databases: map[string]state.DatabaseState{}}
	eng := New(cfg, st, nil, ".")

	actions, err := eng.Plan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
	if actions[0].Type != "create" {
		t.Errorf("expected create, got %s", actions[0].Type)
	}
	if actions[0].DatabaseName != "Projects" {
		t.Errorf("expected Projects, got %s", actions[0].DatabaseName)
	}
}

func TestPlan_NoChanges(t *testing.T) {
	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
`)

	st := &state.State{
		Version: "1",
		Databases: map[string]state.DatabaseState{
			"Projects": {ID: "existing-id", Properties: map[string]string{}},
		},
	}
	eng := New(cfg, st, nil, ".")

	actions, err := eng.Plan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 0 {
		t.Errorf("expected 0 actions, got %d", len(actions))
	}
}

func TestPlan_MixedCreateAndExisting(t *testing.T) {
	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
  - name: Tasks
    parent_page_id: "page1"
    properties:
      Name:
        type: title
      project:
        type: relation
        relation: Projects
`)

	st := &state.State{
		Version: "1",
		Databases: map[string]state.DatabaseState{
			"Projects": {ID: "existing-id", Properties: map[string]string{}},
		},
	}
	eng := New(cfg, st, nil, ".")

	actions, err := eng.Plan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].DatabaseName != "Tasks" {
		t.Errorf("expected Tasks, got %s", actions[0].DatabaseName)
	}
}

func TestFormatPlan_Empty(t *testing.T) {
	out := FormatPlan(nil)
	if out != "No changes. Your Notion workspace matches the config." {
		t.Errorf("unexpected output: %s", out)
	}
}

func TestFormatPlan_WithActions(t *testing.T) {
	actions := []Action{
		{Type: "create", DatabaseName: "Projects", Details: []string{"+ property \"Name\" (title)"}},
		{Type: "update", DatabaseName: "Tasks", Details: []string{"options changed"}},
	}
	out := FormatPlan(actions)
	if out == "" {
		t.Error("expected non-empty output")
	}
}

// --- Mock Integration Tests ---

// mockNotionServer simulates the Notion API for testing the full apply flow.
func mockNotionServer(t *testing.T) (*httptest.Server, *apiLog) {
	t.Helper()
	log := &apiLog{}

	mux := http.NewServeMux()

	dbCounter := 0
	var mu sync.Mutex

	// POST /v1/databases → create database
	mux.HandleFunc("POST /v1/databases", func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request: %v", err)
			http.Error(w, "bad request", 400)
			return
		}

		mu.Lock()
		dbCounter++
		id := idForCounter(dbCounter)
		mu.Unlock()

		log.add(apiCall{Method: "POST", Path: "/v1/databases", Body: body, ResponseID: id})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     id,
			"object": "database",
		})
	})

	// PATCH /v1/databases/{id} → update database
	mux.HandleFunc("PATCH /v1/databases/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/v1/databases/")

		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request: %v", err)
			http.Error(w, "bad request", 400)
			return
		}

		log.add(apiCall{Method: "PATCH", Path: "/v1/databases/" + id, Body: body})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":     id,
			"object": "database",
		})
	})

	server := httptest.NewServer(mux)
	return server, log
}

func idForCounter(n int) string {
	return strings.Replace("00000000-0000-0000-0000-00000000000X", "X", string(rune('0'+n)), 1)
}

type apiCall struct {
	Method     string
	Path       string
	Body       map[string]interface{}
	ResponseID string
}

type apiLog struct {
	mu    sync.Mutex
	calls []apiCall
}

func (l *apiLog) add(c apiCall) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, c)
}

func (l *apiLog) all() []apiCall {
	l.mu.Lock()
	defer l.mu.Unlock()
	cp := make([]apiCall, len(l.calls))
	copy(cp, l.calls)
	return cp
}

// TestApply_TwoPassRelations verifies that fresh databases are created first,
// then relations are added in a second pass when all IDs are available.
func TestApply_TwoPassRelations(t *testing.T) {
	server, log := mockNotionServer(t)
	defer server.Close()

	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Projects
    parent_page_id: "page-1"
    properties:
      Name:
        type: title
      tasks:
        type: relation
        relation: Tasks

  - name: Tasks
    parent_page_id: "page-2"
    properties:
      Name:
        type: title
      project:
        type: relation
        relation: Projects
`)

	dir := t.TempDir()
	st := &state.State{Version: "1", Databases: map[string]state.DatabaseState{}}

	client := notion.NewClientWithBase(server.URL+"/v1", "fake-token")
	eng := New(cfg, st, client, dir)

	actions, err := eng.Apply()
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	// Should have 2 create actions
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}

	calls := log.all()

	// Expected API calls:
	// 1. POST /databases (create Projects without relations)
	// 2. POST /databases (create Tasks without relations)
	// 3. PATCH /databases/{projects-id} (add relation to Tasks)
	// 4. PATCH /databases/{tasks-id} (add relation to Projects)
	if len(calls) != 4 {
		t.Fatalf("expected 4 API calls (2 create + 2 update), got %d", len(calls))
	}

	// First two should be POSTs (create)
	if calls[0].Method != "POST" || calls[1].Method != "POST" {
		t.Errorf("expected first 2 calls to be POST, got %s and %s", calls[0].Method, calls[1].Method)
	}

	// Last two should be PATCHes (update with relations)
	if calls[2].Method != "PATCH" || calls[3].Method != "PATCH" {
		t.Errorf("expected last 2 calls to be PATCH, got %s and %s", calls[2].Method, calls[3].Method)
	}

	// Verify relations in PATCH calls contain resolved IDs, not placeholders
	for _, call := range calls[2:] {
		props, ok := call.Body["properties"].(map[string]interface{})
		if !ok {
			t.Error("PATCH body missing properties")
			continue
		}
		for propName, propVal := range props {
			propMap, ok := propVal.(map[string]interface{})
			if !ok {
				continue
			}
			relMap, ok := propMap["relation"].(map[string]interface{})
			if !ok {
				continue
			}
			dbID, _ := relMap["database_id"].(string)
			if strings.Contains(dbID, "{{resolve:") {
				t.Errorf("property %q still has unresolved placeholder: %s", propName, dbID)
			}
		}
	}

	// Verify state was saved
	savedState, err := state.Load(dir)
	if err != nil {
		t.Fatalf("failed to load saved state: %v", err)
	}

	projectsID, ok := savedState.ResolveDatabase("Projects")
	if !ok || projectsID == "" {
		t.Error("Projects not found in saved state")
	}
	tasksID, ok := savedState.ResolveDatabase("Tasks")
	if !ok || tasksID == "" {
		t.Error("Tasks not found in saved state")
	}

	// Verify state file exists on disk
	statePath := filepath.Join(dir, ".notionctl", "state.json")
	if _, err := os.Stat(statePath); err != nil {
		t.Errorf("state file not found: %v", err)
	}
}

// TestApply_NoDatabasesNoRelations verifies simple apply without relations.
func TestApply_NoDatabasesNoRelations(t *testing.T) {
	server, log := mockNotionServer(t)
	defer server.Close()

	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Notes
    parent_page_id: "page-1"
    properties:
      Name:
        type: title
      Content:
        type: rich_text
`)

	dir := t.TempDir()
	st := &state.State{Version: "1", Databases: map[string]state.DatabaseState{}}

	client := notion.NewClientWithBase(server.URL+"/v1", "fake-token")
	eng := New(cfg, st, client, dir)

	actions, err := eng.Apply()
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}

	calls := log.all()
	// Only 1 POST, no PATCH needed (no relations)
	if len(calls) != 1 {
		t.Fatalf("expected 1 API call, got %d", len(calls))
	}
	if calls[0].Method != "POST" {
		t.Errorf("expected POST, got %s", calls[0].Method)
	}
}

// TestApply_NoChanges verifies apply does nothing when state matches config.
func TestApply_NoChanges(t *testing.T) {
	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
`)

	dir := t.TempDir()
	st := &state.State{
		Version: "1",
		Databases: map[string]state.DatabaseState{
			"Projects": {ID: "existing-id"},
		},
	}

	eng := New(cfg, st, nil, dir)
	actions, err := eng.Apply()
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(actions) != 0 {
		t.Errorf("expected 0 actions, got %d", len(actions))
	}
}
