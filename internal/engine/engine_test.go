package engine

import (
	"testing"

	"github.com/radityajayantara/notionctl/internal/config"
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
