package property

import (
	"testing"
)

func TestAllCoreTypesRegistered(t *testing.T) {
	expected := []string{"title", "rich_text", "number", "select", "multi_select", "relation"}
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
	if len(types) < 6 {
		t.Errorf("expected at least 6 types, got %d: %v", len(types), types)
	}
}
