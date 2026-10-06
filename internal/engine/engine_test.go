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

	"github.com/radityajay/notionctl/internal/config"
	"github.com/radityajay/notionctl/internal/notion"
	"github.com/radityajay/notionctl/internal/state"

	// Register property types for tests.
	_ "github.com/radityajay/notionctl/internal/property"
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
			"Projects": {
				ID: "existing-id",
				Properties: map[string]state.PropertyState{
					"Name": {Type: "title"},
				},
			},
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
			"Projects": {
				ID: "existing-id",
				Properties: map[string]state.PropertyState{
					"Name": {Type: "title"},
				},
			},
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
			"Projects": {
				ID: "existing-id",
				Properties: map[string]state.PropertyState{
					"Name": {Type: "title"},
				},
			},
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

// TestPlan_DetectsNewProperty verifies plan detects when a new property is added.
func TestPlan_DetectsNewProperty(t *testing.T) {
	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
      Description:
        type: rich_text
`)

	st := &state.State{
		Version: "1",
		Databases: map[string]state.DatabaseState{
			"Projects": {
				ID: "existing-id",
				Properties: map[string]state.PropertyState{
					"Name": {Type: "title"},
				},
			},
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
	if actions[0].Type != "update" {
		t.Errorf("expected update, got %s", actions[0].Type)
	}

	found := false
	for _, d := range actions[0].Details {
		if strings.Contains(d, "Description") && strings.Contains(d, "+") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected detail about new property Description, got %v", actions[0].Details)
	}
}

// TestPlan_DetectsRemovedProperty verifies plan detects when a property is removed from YAML.
func TestPlan_DetectsRemovedProperty(t *testing.T) {
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
			"Projects": {
				ID: "existing-id",
				Properties: map[string]state.PropertyState{
					"Name":        {Type: "title"},
					"Description": {Type: "rich_text"},
				},
			},
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

	found := false
	for _, d := range actions[0].Details {
		if strings.Contains(d, "Description") && strings.Contains(d, "removed") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected detail about removed property Description, got %v", actions[0].Details)
	}
}

// TestPlan_DetectsTypeChange verifies plan detects when a property type changes.
func TestPlan_DetectsTypeChange(t *testing.T) {
	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
      Status:
        type: status
`)

	st := &state.State{
		Version: "1",
		Databases: map[string]state.DatabaseState{
			"Projects": {
				ID: "existing-id",
				Properties: map[string]state.PropertyState{
					"Name":   {Type: "title"},
					"Status": {Type: "select"},
				},
			},
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

	found := false
	for _, d := range actions[0].Details {
		if strings.Contains(d, "Status") && strings.Contains(d, "select") && strings.Contains(d, "status") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected detail about type change select → status, got %v", actions[0].Details)
	}
}

// TestApply_UpdateAddsNewProperty verifies apply sends update to Notion when property is added.
func TestApply_UpdateAddsNewProperty(t *testing.T) {
	server, log := mockNotionServer(t)
	defer server.Close()

	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
      Description:
        type: rich_text
`)

	dir := t.TempDir()
	st := &state.State{
		Version: "1",
		Databases: map[string]state.DatabaseState{
			"Projects": {
				ID: "existing-id",
				Properties: map[string]state.PropertyState{
					"Name": {Type: "title"},
				},
			},
		},
	}

	client := notion.NewClientWithBase(server.URL+"/v1", "fake-token")
	eng := New(cfg, st, client, dir)

	actions, err := eng.Apply()
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Type != "update" {
		t.Errorf("expected update, got %s", actions[0].Type)
	}

	calls := log.all()
	if len(calls) != 1 {
		t.Fatalf("expected 1 API call, got %d", len(calls))
	}
	if calls[0].Method != "PATCH" {
		t.Errorf("expected PATCH, got %s", calls[0].Method)
	}

	// Verify state now has both properties
	savedState, err := state.Load(dir)
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}
	props := savedState.GetDatabaseProperties("Projects")
	if len(props) != 2 {
		t.Errorf("expected 2 properties in state, got %d", len(props))
	}
	if props["Description"].Type != "rich_text" {
		t.Errorf("expected Description type rich_text, got %s", props["Description"].Type)
	}
}

// TestApply_CreateSavesPropertyState verifies that create saves property types to state.
func TestApply_CreateSavesPropertyState(t *testing.T) {
	server, _ := mockNotionServer(t)
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
      Done:
        type: checkbox
`)

	dir := t.TempDir()
	st := &state.State{Version: "1", Databases: map[string]state.DatabaseState{}}

	client := notion.NewClientWithBase(server.URL+"/v1", "fake-token")
	eng := New(cfg, st, client, dir)

	_, err := eng.Apply()
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	savedState, err := state.Load(dir)
	if err != nil {
		t.Fatalf("failed to load state: %v", err)
	}
	props := savedState.GetDatabaseProperties("Notes")
	if len(props) != 3 {
		t.Fatalf("expected 3 properties in state, got %d", len(props))
	}
	if props["Name"].Type != "title" {
		t.Errorf("expected Name type title, got %s", props["Name"].Type)
	}
	if props["Done"].Type != "checkbox" {
		t.Errorf("expected Done type checkbox, got %s", props["Done"].Type)
	}
}

func TestPlan_DetectsDestroyAction(t *testing.T) {
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
			"Projects": {
				ID:         "proj-id",
				Properties: map[string]state.PropertyState{"Name": {Type: "title"}},
			},
			"OldDatabase": {
				ID:         "old-id",
				Properties: map[string]state.PropertyState{"Name": {Type: "title"}},
			},
		},
	}
	eng := New(cfg, st, nil, ".")

	actions, err := eng.Plan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var destroyActions []Action
	for _, a := range actions {
		if a.Type == "destroy" {
			destroyActions = append(destroyActions, a)
		}
	}
	if len(destroyActions) != 1 {
		t.Fatalf("expected 1 destroy action, got %d", len(destroyActions))
	}
	if destroyActions[0].DatabaseName != "OldDatabase" {
		t.Errorf("expected OldDatabase, got %s", destroyActions[0].DatabaseName)
	}
}

func TestPlan_NoDestroyWhenAllInConfig(t *testing.T) {
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
			"Projects": {
				ID:         "proj-id",
				Properties: map[string]state.PropertyState{"Name": {Type: "title"}},
			},
		},
	}
	eng := New(cfg, st, nil, ".")

	actions, err := eng.Plan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, a := range actions {
		if a.Type == "destroy" {
			t.Errorf("unexpected destroy action: %v", a)
		}
	}
}

func TestApply_DestroyWithConfirmation(t *testing.T) {
	var deletedIDs []string
	var mu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			mu.Lock()
			deletedIDs = append(deletedIDs, r.URL.Path)
			mu.Unlock()
			w.WriteHeader(200)
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "ok"})
			return
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
`)
	tmpDir := t.TempDir()
	st := &state.State{
		Version: "1",
		Databases: map[string]state.DatabaseState{
			"Projects": {
				ID:         "proj-id",
				Properties: map[string]state.PropertyState{"Name": {Type: "title"}},
			},
			"OldDB": {
				ID:         "old-db-id",
				Properties: map[string]state.PropertyState{"Name": {Type: "title"}},
			},
		},
	}

	client := notion.NewClientWithBase(srv.URL, "token")
	eng := New(cfg, st, client, tmpDir)
	eng.SetConfirmDestroy(func(name, id string) bool {
		return true // auto-approve
	})

	actions, err := eng.Apply()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check destroy action was executed
	var foundDestroy bool
	for _, a := range actions {
		if a.Type == "destroy" && a.DatabaseName == "OldDB" {
			foundDestroy = true
			foundArchived := false
			for _, d := range a.Details {
				if strings.Contains(d, "archived") {
					foundArchived = true
				}
			}
			if !foundArchived {
				t.Error("expected '→ archived' in destroy details")
			}
		}
	}
	if !foundDestroy {
		t.Error("expected destroy action for OldDB")
	}

	// Check API was called
	if len(deletedIDs) != 1 || deletedIDs[0] != "/blocks/old-db-id" {
		t.Errorf("expected DELETE /blocks/old-db-id, got %v", deletedIDs)
	}

	// Check state was cleaned up
	if _, exists := st.ResolveDatabase("OldDB"); exists {
		t.Error("expected OldDB to be removed from state")
	}
	if _, exists := st.ResolveDatabase("Projects"); !exists {
		t.Error("expected Projects to remain in state")
	}

	// Check state file was written
	if _, err := os.ReadFile(filepath.Join(tmpDir, ".notionctl", "state.json")); err != nil {
		t.Errorf("expected state file to be written: %v", err)
	}
}

func TestApply_DestroyDeclined(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "DELETE" {
			t.Error("DELETE should not be called when user declines")
		}
		w.WriteHeader(404)
	}))
	defer srv.Close()

	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
`)
	tmpDir := t.TempDir()
	st := &state.State{
		Version: "1",
		Databases: map[string]state.DatabaseState{
			"Projects": {
				ID:         "proj-id",
				Properties: map[string]state.PropertyState{"Name": {Type: "title"}},
			},
			"OldDB": {
				ID:         "old-db-id",
				Properties: map[string]state.PropertyState{"Name": {Type: "title"}},
			},
		},
	}

	client := notion.NewClientWithBase(srv.URL, "token")
	eng := New(cfg, st, client, tmpDir)
	eng.SetConfirmDestroy(func(name, id string) bool {
		return false // decline
	})

	actions, err := eng.Apply()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, a := range actions {
		if a.Type == "destroy" {
			foundSkipped := false
			for _, d := range a.Details {
				if strings.Contains(d, "skipped") {
					foundSkipped = true
				}
			}
			if !foundSkipped {
				t.Error("expected 'skipped' in declined destroy details")
			}
		}
	}

	// OldDB should still be in state
	if _, exists := st.ResolveDatabase("OldDB"); !exists {
		t.Error("expected OldDB to remain in state when declined")
	}
}

func TestApply_DestroyNoConfirmCallback(t *testing.T) {
	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
`)
	tmpDir := t.TempDir()
	st := &state.State{
		Version: "1",
		Databases: map[string]state.DatabaseState{
			"Projects": {
				ID:         "proj-id",
				Properties: map[string]state.PropertyState{"Name": {Type: "title"}},
			},
			"OldDB": {
				ID:         "old-db-id",
				Properties: map[string]state.PropertyState{"Name": {Type: "title"}},
			},
		},
	}

	eng := New(cfg, st, nil, tmpDir)
	// No confirmDestroy set — destroy should be skipped

	actions, err := eng.Plan()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Plan should still show destroy
	var hasDestroy bool
	for _, a := range actions {
		if a.Type == "destroy" {
			hasDestroy = true
		}
	}
	if !hasDestroy {
		t.Error("expected destroy action in plan even without confirm callback")
	}
}

func TestFormatPlan_DestroyAction(t *testing.T) {
	actions := []Action{
		{Type: "create", DatabaseName: "New", Details: []string{"+ property \"Name\" (title)"}},
		{Type: "destroy", DatabaseName: "Old", Details: []string{"database ID: old-id (will be archived in Notion)"}},
	}
	out := FormatPlan(actions)
	if !strings.Contains(out, "- destroy") {
		t.Errorf("expected '- destroy' in plan output, got:\n%s", out)
	}
	if !strings.Contains(out, "+ create") {
		t.Errorf("expected '+ create' in plan output, got:\n%s", out)
	}
}
