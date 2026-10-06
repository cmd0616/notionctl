package engine

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/radityajay/notionctl/internal/notion"
	"github.com/radityajay/notionctl/internal/state"

	_ "github.com/radityajay/notionctl/internal/property"
)

func TestDiff_InSync(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "db-001",
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{"type": "title", "title": map[string]interface{}{}},
				"Desc": map[string]interface{}{"type": "rich_text", "rich_text": map[string]interface{}{}},
			},
		})
	}))
	defer srv.Close()

	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
      Desc:
        type: rich_text
`)
	st := &state.State{
		Version:   "1",
		Databases: map[string]state.DatabaseState{"Projects": {ID: "db-001"}},
	}

	client := notion.NewClientWithBase(srv.URL, "token")
	eng := New(cfg, st, client, ".")

	results, err := eng.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != "in_sync" {
		t.Errorf("expected in_sync, got %s: %v", results[0].Status, results[0].Differences)
	}
}

func TestDiff_Drifted_ExtraRemoteProperty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "db-001",
			"properties": map[string]interface{}{
				"Name":  map[string]interface{}{"type": "title"},
				"Desc":  map[string]interface{}{"type": "rich_text"},
				"Extra": map[string]interface{}{"type": "checkbox"},
			},
		})
	}))
	defer srv.Close()

	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
      Desc:
        type: rich_text
`)
	st := &state.State{
		Version:   "1",
		Databases: map[string]state.DatabaseState{"Projects": {ID: "db-001"}},
	}

	client := notion.NewClientWithBase(srv.URL, "token")
	eng := New(cfg, st, client, ".")

	results, err := eng.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Status != "drifted" {
		t.Errorf("expected drifted, got %s", results[0].Status)
	}
	found := false
	for _, d := range results[0].Differences {
		if strings.Contains(d, "Extra") && strings.Contains(d, "in Notion but not in config") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected diff about Extra property, got: %v", results[0].Differences)
	}
}

func TestDiff_Drifted_MissingRemoteProperty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "db-001",
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{"type": "title"},
			},
		})
	}))
	defer srv.Close()

	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
      Desc:
        type: rich_text
`)
	st := &state.State{
		Version:   "1",
		Databases: map[string]state.DatabaseState{"Projects": {ID: "db-001"}},
	}

	client := notion.NewClientWithBase(srv.URL, "token")
	eng := New(cfg, st, client, ".")

	results, err := eng.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Status != "drifted" {
		t.Errorf("expected drifted, got %s", results[0].Status)
	}
	found := false
	for _, d := range results[0].Differences {
		if strings.Contains(d, "Desc") && strings.Contains(d, "in config but not in Notion") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected diff about Desc property, got: %v", results[0].Differences)
	}
}

func TestDiff_Drifted_TypeChange(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "db-001",
			"properties": map[string]interface{}{
				"Name":   map[string]interface{}{"type": "title"},
				"Status": map[string]interface{}{"type": "select"},
			},
		})
	}))
	defer srv.Close()

	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
      Status:
        type: status
        options:
          - name: Todo
            color: default
`)
	st := &state.State{
		Version:   "1",
		Databases: map[string]state.DatabaseState{"Projects": {ID: "db-001"}},
	}

	client := notion.NewClientWithBase(srv.URL, "token")
	eng := New(cfg, st, client, ".")

	results, err := eng.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Status != "drifted" {
		t.Errorf("expected drifted, got %s", results[0].Status)
	}
	found := false
	for _, d := range results[0].Differences {
		if strings.Contains(d, "Status") && strings.Contains(d, "config=status") && strings.Contains(d, "remote=select") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected type mismatch diff, got: %v", results[0].Differences)
	}
}

func TestDiff_NotDeployed(t *testing.T) {
	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
`)
	st := &state.State{Version: "1", Databases: map[string]state.DatabaseState{}}
	eng := New(cfg, st, nil, ".")

	results, err := eng.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Status != "not_deployed" {
		t.Errorf("expected not_deployed, got %s", results[0].Status)
	}
}

func TestDiff_NumberFormatDrift(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "db-001",
			"properties": map[string]interface{}{
				"Name":  map[string]interface{}{"type": "title"},
				"Price": map[string]interface{}{"type": "number", "number": map[string]interface{}{"format": "euro"}},
			},
		})
	}))
	defer srv.Close()

	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Products
    properties:
      Name:
        type: title
      Price:
        type: number
        format: dollar
`)
	st := &state.State{
		Version:   "1",
		Databases: map[string]state.DatabaseState{"Products": {ID: "db-001"}},
	}

	client := notion.NewClientWithBase(srv.URL, "token")
	eng := New(cfg, st, client, ".")

	results, err := eng.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Status != "drifted" {
		t.Errorf("expected drifted, got %s", results[0].Status)
	}
	found := false
	for _, d := range results[0].Differences {
		if strings.Contains(d, "format") && strings.Contains(d, "dollar") && strings.Contains(d, "euro") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected format drift diff, got: %v", results[0].Differences)
	}
}

func TestDiff_SelectOptionsDrift(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "db-001",
			"properties": map[string]interface{}{
				"Name": map[string]interface{}{"type": "title"},
				"Priority": map[string]interface{}{
					"type": "select",
					"select": map[string]interface{}{
						"options": []interface{}{
							map[string]interface{}{"name": "High"},
							map[string]interface{}{"name": "Medium"},
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	cfg := makeConfig(t, `
version: "1"
databases:
  - name: Tasks
    properties:
      Name:
        type: title
      Priority:
        type: select
        options:
          - name: High
            color: red
          - name: Medium
            color: yellow
          - name: Low
            color: gray
`)
	st := &state.State{
		Version:   "1",
		Databases: map[string]state.DatabaseState{"Tasks": {ID: "db-001"}},
	}

	client := notion.NewClientWithBase(srv.URL, "token")
	eng := New(cfg, st, client, ".")

	results, err := eng.Diff()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results[0].Status != "drifted" {
		t.Errorf("expected drifted, got %s", results[0].Status)
	}
	found := false
	for _, d := range results[0].Differences {
		if strings.Contains(d, "Low") && strings.Contains(d, "in config but not remote") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected select options drift, got: %v", results[0].Differences)
	}
}

func TestFormatDiff_AllInSync(t *testing.T) {
	results := []DiffResult{
		{DatabaseName: "A", Status: "in_sync"},
		{DatabaseName: "B", Status: "in_sync"},
	}
	out := FormatDiff(results)
	if !strings.Contains(out, "All databases in sync") {
		t.Errorf("expected in-sync message, got: %s", out)
	}
}

func TestFormatDiff_Mixed(t *testing.T) {
	results := []DiffResult{
		{DatabaseName: "A", Status: "in_sync"},
		{DatabaseName: "B", Status: "drifted", Differences: []string{"+ extra prop"}},
		{DatabaseName: "C", Status: "not_deployed"},
	}
	out := FormatDiff(results)
	if !strings.Contains(out, "✓") {
		t.Error("expected ✓ for in_sync")
	}
	if !strings.Contains(out, "⚠") {
		t.Error("expected ⚠ for drifted")
	}
	if !strings.Contains(out, "○") {
		t.Error("expected ○ for not_deployed")
	}
}

func TestFormatDiff_Empty(t *testing.T) {
	out := FormatDiff(nil)
	if !strings.Contains(out, "No databases") {
		t.Errorf("expected no databases message, got: %s", out)
	}
}
