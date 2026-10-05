package importer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/radityajay/notionctl/internal/config"
	"github.com/radityajay/notionctl/internal/notion"
	// Register property types for config validation
	_ "github.com/radityajay/notionctl/internal/property"
)

// mockServer creates a test server that responds to Notion API calls.
func mockServer(blocks []map[string]interface{}, databases map[string]map[string]interface{}) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		path := r.URL.Path

		// GET /blocks/{id}/children
		if strings.Contains(path, "/blocks/") && strings.HasSuffix(path, "/children") {
			resp := map[string]interface{}{
				"results":  blocks,
				"has_more": false,
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// GET /databases/{id}
		if strings.HasPrefix(path, "/v1/databases/") {
			dbID := strings.TrimPrefix(path, "/v1/databases/")
			if db, ok := databases[dbID]; ok {
				json.NewEncoder(w).Encode(db)
				return
			}
			w.WriteHeader(404)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"object":  "error",
				"status":  404,
				"message": "not found",
			})
			return
		}

		w.WriteHeader(404)
	}))
}

func TestFromPage_BasicImport(t *testing.T) {
	blocks := []map[string]interface{}{
		{"type": "child_database", "id": "db-001"},
		{"type": "paragraph", "id": "block-002"}, // non-database, should be ignored
	}

	databases := map[string]map[string]interface{}{
		"db-001": {
			"id": "db-001",
			"title": []interface{}{
				map[string]interface{}{"plain_text": "Tasks"},
			},
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"id":    "title",
					"type":  "title",
					"title": map[string]interface{}{},
				},
				"Notes": map[string]interface{}{
					"id":        "abc1",
					"type":      "rich_text",
					"rich_text": map[string]interface{}{},
				},
				"Done": map[string]interface{}{
					"id":       "abc2",
					"type":     "checkbox",
					"checkbox": map[string]interface{}{},
				},
			},
		},
	}

	srv := mockServer(blocks, databases)
	defer srv.Close()

	client := notion.NewClientWithBase(srv.URL+"/v1", "test-token")
	imp := New(client)

	result, err := imp.FromPage("page-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Config.Databases) != 1 {
		t.Fatalf("expected 1 database, got %d", len(result.Config.Databases))
	}

	db := result.Config.Databases[0]
	if db.Name != "Tasks" {
		t.Errorf("expected name 'Tasks', got %q", db.Name)
	}
	if len(db.Properties) != 3 {
		t.Errorf("expected 3 properties, got %d", len(db.Properties))
	}
	if db.Properties["Name"].Type != "title" {
		t.Errorf("expected Name type 'title', got %q", db.Properties["Name"].Type)
	}

	// State should be populated
	stateDB, ok := result.State.Databases["Tasks"]
	if !ok {
		t.Fatal("expected Tasks in state")
	}
	if stateDB.ID != "db-001" {
		t.Errorf("expected state ID 'db-001', got %q", stateDB.ID)
	}
}

func TestFromPage_NumberFormat(t *testing.T) {
	blocks := []map[string]interface{}{
		{"type": "child_database", "id": "db-001"},
	}
	databases := map[string]map[string]interface{}{
		"db-001": {
			"id": "db-001",
			"title": []interface{}{
				map[string]interface{}{"plain_text": "Products"},
			},
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"id":    "title",
					"type":  "title",
					"title": map[string]interface{}{},
				},
				"Price": map[string]interface{}{
					"id":   "p1",
					"type": "number",
					"number": map[string]interface{}{
						"format": "dollar",
					},
				},
			},
		},
	}

	srv := mockServer(blocks, databases)
	defer srv.Close()

	client := notion.NewClientWithBase(srv.URL+"/v1", "test-token")
	imp := New(client)
	result, err := imp.FromPage("page-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	price := result.Config.Databases[0].Properties["Price"]
	if price.Extra["format"] != "dollar" {
		t.Errorf("expected format 'dollar', got %v", price.Extra["format"])
	}
}

func TestFromPage_Formula(t *testing.T) {
	blocks := []map[string]interface{}{
		{"type": "child_database", "id": "db-001"},
	}
	databases := map[string]map[string]interface{}{
		"db-001": {
			"id": "db-001",
			"title": []interface{}{
				map[string]interface{}{"plain_text": "Calc"},
			},
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"id":    "title",
					"type":  "title",
					"title": map[string]interface{}{},
				},
				"Double": map[string]interface{}{
					"id":   "f1",
					"type": "formula",
					"formula": map[string]interface{}{
						"expression": `prop("Value") * 2`,
					},
				},
			},
		},
	}

	srv := mockServer(blocks, databases)
	defer srv.Close()

	client := notion.NewClientWithBase(srv.URL+"/v1", "test-token")
	imp := New(client)
	result, err := imp.FromPage("page-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	double := result.Config.Databases[0].Properties["Double"]
	if double.Type != "formula" {
		t.Errorf("expected type 'formula', got %q", double.Type)
	}
	if double.Extra["expression"] != `prop("Value") * 2` {
		t.Errorf("expected expression preserved, got %v", double.Extra["expression"])
	}
}

func TestFromPage_Rollup(t *testing.T) {
	blocks := []map[string]interface{}{
		{"type": "child_database", "id": "db-001"},
	}
	databases := map[string]map[string]interface{}{
		"db-001": {
			"id": "db-001",
			"title": []interface{}{
				map[string]interface{}{"plain_text": "Projects"},
			},
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"id":    "title",
					"type":  "title",
					"title": map[string]interface{}{},
				},
				"Total": map[string]interface{}{
					"id":   "r1",
					"type": "rollup",
					"rollup": map[string]interface{}{
						"relation_property_name": "tasks",
						"rollup_property_name":   "Estimate",
						"function":               "sum",
					},
				},
			},
		},
	}

	srv := mockServer(blocks, databases)
	defer srv.Close()

	client := notion.NewClientWithBase(srv.URL+"/v1", "test-token")
	imp := New(client)
	result, err := imp.FromPage("page-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	total := result.Config.Databases[0].Properties["Total"]
	if total.Type != "rollup" {
		t.Errorf("expected type 'rollup', got %q", total.Type)
	}
	if total.Extra["relation"] != "tasks" {
		t.Errorf("expected relation 'tasks', got %v", total.Extra["relation"])
	}
	if total.Extra["rollup_property"] != "Estimate" {
		t.Errorf("expected rollup_property 'Estimate', got %v", total.Extra["rollup_property"])
	}
	if total.Extra["function"] != "sum" {
		t.Errorf("expected function 'sum', got %v", total.Extra["function"])
	}
}

func TestFromPage_RelationInScope(t *testing.T) {
	blocks := []map[string]interface{}{
		{"type": "child_database", "id": "db-001"},
		{"type": "child_database", "id": "db-002"},
	}
	databases := map[string]map[string]interface{}{
		"db-001": {
			"id": "db-001",
			"title": []interface{}{
				map[string]interface{}{"plain_text": "Projects"},
			},
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"id": "title", "type": "title", "title": map[string]interface{}{},
				},
				"tasks": map[string]interface{}{
					"id":   "rel1",
					"type": "relation",
					"relation": map[string]interface{}{
						"database_id":     "db-002",
						"type":            "single_property",
						"single_property": map[string]interface{}{},
					},
				},
			},
		},
		"db-002": {
			"id": "db-002",
			"title": []interface{}{
				map[string]interface{}{"plain_text": "Tasks"},
			},
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"id": "title", "type": "title", "title": map[string]interface{}{},
				},
			},
		},
	}

	srv := mockServer(blocks, databases)
	defer srv.Close()

	client := notion.NewClientWithBase(srv.URL+"/v1", "test-token")
	imp := New(client)
	result, err := imp.FromPage("page-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Config.Databases) != 2 {
		t.Fatalf("expected 2 databases, got %d", len(result.Config.Databases))
	}

	// Find Projects
	var projects *struct{ props map[string]interface{} }
	for _, db := range result.Config.Databases {
		if db.Name == "Projects" {
			tasksProp := db.Properties["tasks"]
			if tasksProp.Extra["relation"] != "Tasks" {
				t.Errorf("expected relation target 'Tasks', got %v", tasksProp.Extra["relation"])
			}
			return
		}
	}
	if projects == nil {
		t.Fatal("Projects database not found")
	}
}

func TestFromPage_RelationOutOfScope(t *testing.T) {
	blocks := []map[string]interface{}{
		{"type": "child_database", "id": "db-001"},
	}
	databases := map[string]map[string]interface{}{
		"db-001": {
			"id": "db-001",
			"title": []interface{}{
				map[string]interface{}{"plain_text": "Projects"},
			},
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"id": "title", "type": "title", "title": map[string]interface{}{},
				},
				"external": map[string]interface{}{
					"id":   "rel1",
					"type": "relation",
					"relation": map[string]interface{}{
						"database_id":     "db-999", // not in scope
						"type":            "single_property",
						"single_property": map[string]interface{}{},
					},
				},
			},
		},
	}

	srv := mockServer(blocks, databases)
	defer srv.Close()

	client := notion.NewClientWithBase(srv.URL+"/v1", "test-token")
	imp := New(client)
	result, err := imp.FromPage("page-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// external relation should be skipped
	db := result.Config.Databases[0]
	if _, exists := db.Properties["external"]; exists {
		t.Error("expected out-of-scope relation to be skipped")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning for out-of-scope relation")
	}
}

func TestFromPage_UnsupportedType(t *testing.T) {
	blocks := []map[string]interface{}{
		{"type": "child_database", "id": "db-001"},
	}
	databases := map[string]map[string]interface{}{
		"db-001": {
			"id": "db-001",
			"title": []interface{}{
				map[string]interface{}{"plain_text": "Test"},
			},
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"id": "title", "type": "title", "title": map[string]interface{}{},
				},
				"Owner": map[string]interface{}{
					"id":     "ppl1",
					"type":   "people",
					"people": map[string]interface{}{},
				},
			},
		},
	}

	srv := mockServer(blocks, databases)
	defer srv.Close()

	client := notion.NewClientWithBase(srv.URL+"/v1", "test-token")
	imp := New(client)
	result, err := imp.FromPage("page-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	db := result.Config.Databases[0]
	if _, exists := db.Properties["Owner"]; exists {
		t.Error("expected unsupported 'people' type to be skipped")
	}
	if len(result.Warnings) == 0 {
		t.Error("expected warning for unsupported type")
	}
	found := false
	for _, w := range result.Warnings {
		if strings.Contains(w, "people") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning mentioning 'people', got %v", result.Warnings)
	}
}

func TestFromPage_NoDatabases(t *testing.T) {
	blocks := []map[string]interface{}{
		{"type": "paragraph", "id": "block-001"},
	}
	databases := map[string]map[string]interface{}{}

	srv := mockServer(blocks, databases)
	defer srv.Close()

	client := notion.NewClientWithBase(srv.URL+"/v1", "test-token")
	imp := New(client)
	_, err := imp.FromPage("page-001")
	if err == nil {
		t.Fatal("expected error for page with no databases")
	}
	if !strings.Contains(err.Error(), "no databases found") {
		t.Errorf("expected 'no databases found' error, got %v", err)
	}
}

func TestMarshalYAML_Output(t *testing.T) {
	cfg := &config.Config{
		Version: "1",
		Databases: []config.Database{
			{
				Name:         "Tasks",
				ParentPageID: "page-001",
				Properties: map[string]config.PropertyDef{
					"Name":  {Type: "title", Extra: map[string]interface{}{}},
					"Notes": {Type: "rich_text", Extra: map[string]interface{}{}},
				},
			},
		},
	}

	data, err := MarshalYAML(cfg)
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}

	yaml := string(data)
	if !strings.Contains(yaml, "Generated by notionctl import") {
		t.Error("expected header comment")
	}
	if !strings.Contains(yaml, "version:") {
		t.Error("expected version field")
	}
	if !strings.Contains(yaml, "Tasks") {
		t.Error("expected database name")
	}
}

func TestFromPage_SelectOptions(t *testing.T) {
	blocks := []map[string]interface{}{
		{"type": "child_database", "id": "db-001"},
	}
	databases := map[string]map[string]interface{}{
		"db-001": {
			"id": "db-001",
			"title": []interface{}{
				map[string]interface{}{"plain_text": "Test"},
			},
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{
					"id": "title", "type": "title", "title": map[string]interface{}{},
				},
				"Priority": map[string]interface{}{
					"id":   "s1",
					"type": "select",
					"select": map[string]interface{}{
						"options": []interface{}{
							map[string]interface{}{"name": "High", "color": "red"},
							map[string]interface{}{"name": "Low", "color": "green"},
						},
					},
				},
			},
		},
	}

	srv := mockServer(blocks, databases)
	defer srv.Close()

	client := notion.NewClientWithBase(srv.URL+"/v1", "test-token")
	imp := New(client)
	result, err := imp.FromPage("page-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	priority := result.Config.Databases[0].Properties["Priority"]
	if priority.Type != "select" {
		t.Errorf("expected type 'select', got %q", priority.Type)
	}
	opts, ok := priority.Extra["options"].([]interface{})
	if !ok || len(opts) != 2 {
		t.Errorf("expected 2 options, got %v", priority.Extra["options"])
	}
}

func TestMarshalYAML_Roundtrip(t *testing.T) {
	cfg := &config.Config{
		Version: "1",
		Databases: []config.Database{
			{
				Name:         "Projects",
				ParentPageID: "p1",
				Properties: map[string]config.PropertyDef{
					"Name": {Type: "title", Extra: map[string]interface{}{}},
					"Price": {Type: "number", Extra: map[string]interface{}{"format": "dollar"}},
				},
			},
		},
	}

	data, err := MarshalYAML(cfg)
	if err != nil {
		t.Fatalf("MarshalYAML: %v", err)
	}
	yaml := string(data)
	if !strings.Contains(yaml, "dollar") {
		t.Error("expected number format 'dollar' in YAML output")
	}
}
