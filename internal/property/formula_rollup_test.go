package property

import (
	"strings"
	"testing"
)

// --- Formula tests ---

func TestFormulaToNotion(t *testing.T) {
	p, _ := Get("formula")
	cfg, err := p.ToNotion(map[string]interface{}{
		"expression": `prop("Estimate") * 2`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	formulaCfg := cfg["formula"].(map[string]interface{})
	if formulaCfg["expression"] != `prop("Estimate") * 2` {
		t.Errorf("expected expression preserved, got %v", formulaCfg["expression"])
	}
}

func TestFormulaToNotion_MissingExpression(t *testing.T) {
	p, _ := Get("formula")
	_, err := p.ToNotion(map[string]interface{}{})
	if err == nil {
		t.Fatal("expected error for missing expression")
	}
}

func TestFormulaDiffSummary_Changed(t *testing.T) {
	p, _ := Get("formula")
	diff := p.DiffSummary(
		map[string]interface{}{"expression": `prop("Price") * 1.1`},
		map[string]interface{}{"expression": `prop("Price") * 1.0`},
	)
	if diff == "" {
		t.Error("expected non-empty diff when expression changes")
	}
	if !strings.Contains(diff, "expression:") {
		t.Errorf("diff should mention 'expression:', got %q", diff)
	}
}

func TestFormulaDiffSummary_NoChange(t *testing.T) {
	p, _ := Get("formula")
	diff := p.DiffSummary(
		map[string]interface{}{"expression": `prop("X") + 1`},
		map[string]interface{}{"expression": `prop("X") + 1`},
	)
	if diff != "" {
		t.Errorf("expected empty diff, got %q", diff)
	}
}

// --- Rollup tests ---

func TestRollupToNotion(t *testing.T) {
	p, _ := Get("rollup")
	cfg, err := p.ToNotion(map[string]interface{}{
		"relation":        "tasks",
		"rollup_property": "estimate",
		"function":        "sum",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	rollupCfg := cfg["rollup"].(map[string]interface{})
	if rollupCfg["relation_property_name"] != "tasks" {
		t.Errorf("expected relation_property_name 'tasks', got %v", rollupCfg["relation_property_name"])
	}
	if rollupCfg["rollup_property_name"] != "estimate" {
		t.Errorf("expected rollup_property_name 'estimate', got %v", rollupCfg["rollup_property_name"])
	}
	if rollupCfg["function"] != "sum" {
		t.Errorf("expected function 'sum', got %v", rollupCfg["function"])
	}
}

func TestRollupToNotion_MissingRelation(t *testing.T) {
	p, _ := Get("rollup")
	_, err := p.ToNotion(map[string]interface{}{
		"rollup_property": "estimate",
		"function":        "sum",
	})
	if err == nil {
		t.Fatal("expected error for missing relation")
	}
}

func TestRollupToNotion_MissingRollupProperty(t *testing.T) {
	p, _ := Get("rollup")
	_, err := p.ToNotion(map[string]interface{}{
		"relation": "tasks",
		"function": "sum",
	})
	if err == nil {
		t.Fatal("expected error for missing rollup_property")
	}
}

func TestRollupToNotion_MissingFunction(t *testing.T) {
	p, _ := Get("rollup")
	_, err := p.ToNotion(map[string]interface{}{
		"relation":        "tasks",
		"rollup_property": "estimate",
	})
	if err == nil {
		t.Fatal("expected error for missing function")
	}
}

func TestRollupToNotion_InvalidFunction(t *testing.T) {
	p, _ := Get("rollup")
	_, err := p.ToNotion(map[string]interface{}{
		"relation":        "tasks",
		"rollup_property": "estimate",
		"function":        "not_a_function",
	})
	if err == nil {
		t.Fatal("expected error for invalid function")
	}
	if !strings.Contains(err.Error(), "not_a_function") {
		t.Errorf("error should mention the invalid function, got %v", err)
	}
}

func TestRollupToNotion_AllFunctions(t *testing.T) {
	p, _ := Get("rollup")
	for fn := range ValidRollupFunctions {
		t.Run(fn, func(t *testing.T) {
			_, err := p.ToNotion(map[string]interface{}{
				"relation":        "tasks",
				"rollup_property": "estimate",
				"function":        fn,
			})
			if err != nil {
				t.Errorf("function %q should be valid: %v", fn, err)
			}
		})
	}
}

func TestRollupDiffSummary_FunctionChanged(t *testing.T) {
	p, _ := Get("rollup")
	diff := p.DiffSummary(
		map[string]interface{}{"relation": "tasks", "rollup_property": "estimate", "function": "average"},
		map[string]interface{}{"relation": "tasks", "rollup_property": "estimate", "function": "sum"},
	)
	if !strings.Contains(diff, "function:") {
		t.Errorf("expected function diff, got %q", diff)
	}
}

func TestRollupDiffSummary_MultipleChanges(t *testing.T) {
	p, _ := Get("rollup")
	diff := p.DiffSummary(
		map[string]interface{}{"relation": "projects", "rollup_property": "budget", "function": "average"},
		map[string]interface{}{"relation": "tasks", "rollup_property": "estimate", "function": "sum"},
	)
	if !strings.Contains(diff, "relation:") {
		t.Errorf("expected relation diff, got %q", diff)
	}
	if !strings.Contains(diff, "rollup_property:") {
		t.Errorf("expected rollup_property diff, got %q", diff)
	}
	if !strings.Contains(diff, "function:") {
		t.Errorf("expected function diff, got %q", diff)
	}
}

func TestRollupDiffSummary_NoChange(t *testing.T) {
	p, _ := Get("rollup")
	diff := p.DiffSummary(
		map[string]interface{}{"relation": "tasks", "rollup_property": "estimate", "function": "sum"},
		map[string]interface{}{"relation": "tasks", "rollup_property": "estimate", "function": "sum"},
	)
	if diff != "" {
		t.Errorf("expected empty diff, got %q", diff)
	}
}
