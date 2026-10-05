package property

import (
	"testing"
)

func TestAllCoreTypesRegistered(t *testing.T) {
	expected := []string{
		"title", "rich_text", "number", "select", "multi_select", "relation",
		"checkbox", "date", "url", "email", "phone_number",
		"created_time", "last_edited_time", "status",
	}
	for _, typ := range expected {
		if _, err := Get(typ); err != nil {
			t.Errorf("expected %q to be registered: %v", typ, err)
		}
	}
}

func TestGetUnknownType(t *testing.T) {
	_, err := Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
}

func TestTitleToNotion(t *testing.T) {
	p, _ := Get("title")
	cfg, err := p.ToNotion(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cfg["title"]; !ok {
		t.Error("expected 'title' key in config")
	}
}

func TestNumberToNotion_DefaultFormat(t *testing.T) {
	p, _ := Get("number")
	cfg, err := p.ToNotion(map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	numCfg := cfg["number"].(map[string]interface{})
	if numCfg["format"] != "number" {
		t.Errorf("expected default format 'number', got %v", numCfg["format"])
	}
}

func TestNumberToNotion_CustomFormat(t *testing.T) {
	p, _ := Get("number")
	cfg, err := p.ToNotion(map[string]interface{}{"format": "dollar"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	numCfg := cfg["number"].(map[string]interface{})
	if numCfg["format"] != "dollar" {
		t.Errorf("expected format 'dollar', got %v", numCfg["format"])
	}
}

func TestSelectToNotion(t *testing.T) {
	p, _ := Get("select")
	cfg, err := p.ToNotion(map[string]interface{}{
		"options": []interface{}{
			map[string]interface{}{"name": "A", "color": "red"},
			map[string]interface{}{"name": "B", "color": "blue"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	selectCfg := cfg["select"].(map[string]interface{})
	options := selectCfg["options"].([]map[string]interface{})
	if len(options) != 2 {
		t.Errorf("expected 2 options, got %d", len(options))
	}
}

func TestRelationToNotion(t *testing.T) {
	p, _ := Get("relation")
	cfg, err := p.ToNotion(map[string]interface{}{"relation": "Tasks"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	relCfg := cfg["relation"].(map[string]interface{})
	dbID := relCfg["database_id"].(string)
	if dbID != "{{resolve:Tasks}}" {
		t.Errorf("expected resolve placeholder, got %v", dbID)
	}
}

func TestRelationToNotion_MissingTarget(t *testing.T) {
	p, _ := Get("relation")
	_, err := p.ToNotion(map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing relation target")
	}
}

func TestSupportedTypes(t *testing.T) {
	types := SupportedTypes()
	if len(types) < 14 {
		t.Errorf("expected at least 14 types, got %d: %v", len(types), types)
	}
}

// --- Tests for new v0.2.0 property types ---

func TestSimplePropertyTypes(t *testing.T) {
	// These types all have empty config — just verify ToNotion returns the correct key.
	simpleTypes := []string{"checkbox", "date", "url", "email", "phone_number", "created_time"}
	for _, typ := range simpleTypes {
		t.Run(typ, func(t *testing.T) {
			p, err := Get(typ)
			if err != nil {
				t.Fatalf("Get(%q): %v", typ, err)
			}
			if p.Type() != typ {
				t.Fatalf("expected Type() = %q, got %q", typ, p.Type())
			}
			cfg, err := p.ToNotion(nil)
			if err != nil {
				t.Fatalf("ToNotion(nil): %v", err)
			}
			if _, ok := cfg[typ]; !ok {
				t.Errorf("expected %q key in config, got %v", typ, cfg)
			}
			if diff := p.DiffSummary(map[string]interface{}{}, map[string]interface{}{}); diff != "" {
				t.Errorf("expected empty diff, got %q", diff)
			}
		})
	}
}

func TestStatusToNotion_WithOptions(t *testing.T) {
	p, _ := Get("status")
	cfg, err := p.ToNotion(map[string]interface{}{
		"options": []interface{}{
			map[string]interface{}{"name": "Not Started", "color": "default"},
			map[string]interface{}{"name": "In Progress", "color": "blue"},
			map[string]interface{}{"name": "Done", "color": "green"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	statusCfg := cfg["status"].(map[string]interface{})
	options := statusCfg["options"].([]map[string]interface{})
	if len(options) != 3 {
		t.Errorf("expected 3 options, got %d", len(options))
	}
}

func TestStatusToNotion_WithGroups(t *testing.T) {
	p, _ := Get("status")
	cfg, err := p.ToNotion(map[string]interface{}{
		"options": []interface{}{
			map[string]interface{}{"name": "Not Started", "color": "default"},
			map[string]interface{}{"name": "Done", "color": "green"},
		},
		"groups": []interface{}{
			map[string]interface{}{
				"name":       "To-do",
				"option_ids": []interface{}{"Not Started"},
			},
			map[string]interface{}{
				"name":       "Complete",
				"option_ids": []interface{}{"Done"},
			},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	statusCfg := cfg["status"].(map[string]interface{})
	groups := statusCfg["groups"].([]map[string]interface{})
	if len(groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(groups))
	}
}

func TestStatusToNotion_Empty(t *testing.T) {
	p, _ := Get("status")
	cfg, err := p.ToNotion(map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := cfg["status"]; !ok {
		t.Error("expected 'status' key in config")
	}
}

func TestStatusDiffSummary(t *testing.T) {
	p, _ := Get("status")
	diff := p.DiffSummary(
		map[string]interface{}{
			"options": []interface{}{
				map[string]interface{}{"name": "A"},
				map[string]interface{}{"name": "B"},
			},
		},
		map[string]interface{}{
			"options": []interface{}{
				map[string]interface{}{"name": "A"},
			},
		},
	)
	if diff == "" {
		t.Error("expected non-empty diff when options differ")
	}
}

func TestStatusDiffSummary_NoChange(t *testing.T) {
	p, _ := Get("status")
	diff := p.DiffSummary(
		map[string]interface{}{
			"options": []interface{}{
				map[string]interface{}{"name": "X"},
			},
		},
		map[string]interface{}{
			"options": []interface{}{
				map[string]interface{}{"name": "X"},
			},
		},
	)
	if diff != "" {
		t.Errorf("expected empty diff, got %q", diff)
	}
}
