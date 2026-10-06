package config

import (
	"os"
	"testing"

	// Register property types for validation tests.
	_ "github.com/radityajay/notionctl/internal/property"
)

func TestParse_ValidConfig(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Projects
    parent_page_id: "abc123"
    properties:
      Name:
        type: title
      Status:
        type: select
        options:
          - name: Active
            color: green
      tasks:
        type: relation
        relation: Tasks

  - name: Tasks
    parent_page_id: "abc123"
    properties:
      Name:
        type: title
      project:
        type: relation
        relation: Projects
`
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Databases) != 2 {
		t.Fatalf("expected 2 databases, got %d", len(cfg.Databases))
	}
	if cfg.Databases[0].Name != "Projects" {
		t.Errorf("expected first db name 'Projects', got %q", cfg.Databases[0].Name)
	}
	if cfg.Databases[1].Properties["project"].Type != "relation" {
		t.Errorf("expected relation type for 'project' property")
	}
}

func TestParse_MissingVersion(t *testing.T) {
	yaml := `
databases:
  - name: Test
    properties:
      Name:
        type: title
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing version")
	}
}

func TestParse_NoTitle(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Test
    properties:
      Description:
        type: rich_text
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing title property")
	}
}

func TestParse_DuplicateName(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Same
    properties:
      Name:
        type: title
  - name: Same
    properties:
      Name:
        type: title
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for duplicate database name")
	}
}

func TestParse_InvalidRelationRef(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
      tasks:
        type: relation
        relation: NonExistent
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for invalid relation reference")
	}
}

func TestDatabaseNames(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: A
    properties:
      Name:
        type: title
  - name: B
    properties:
      Title:
        type: title
`
	cfg, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	names := cfg.DatabaseNames()
	if len(names) != 2 || names[0] != "A" || names[1] != "B" {
		t.Errorf("unexpected names: %v", names)
	}
}

func TestParse_UnknownPropertyType(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Test
    properties:
      Name:
        type: title
      Status:
        type: selec
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for unknown property type 'selec'")
	}
}

func TestParse_MultipleTitles(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Test
    properties:
      Name:
        type: title
      AnotherTitle:
        type: title
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for multiple title properties")
	}
}

func TestParse_InvalidNumberFormat(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Test
    properties:
      Name:
        type: title
      Price:
        type: number
        format: dolars
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for invalid number format 'dolars'")
	}
}

func TestParse_ValidNumberFormat(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Test
    properties:
      Name:
        type: title
      Price:
        type: number
        format: dollar
`
	_, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParse_SelectOptionsMustBeList(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Test
    properties:
      Name:
        type: title
      Status:
        type: select
        options: "not a list"
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for options not being a list")
	}
}

func TestParse_SelectOptionMissingName(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Test
    properties:
      Name:
        type: title
      Status:
        type: select
        options:
          - color: red
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for select option missing name")
	}
}

func TestParse_StatusGroupsMustBeList(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Test
    properties:
      Name:
        type: title
      Status:
        type: status
        groups: "not a list"
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for groups not being a list")
	}
}

// --- Formula config validation tests ---

func TestParse_FormulaValid(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Test
    properties:
      Name:
        type: title
      DoubleEstimate:
        type: formula
        expression: 'prop("Estimate") * 2'
`
	_, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParse_FormulaMissingExpression(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Test
    properties:
      Name:
        type: title
      Calc:
        type: formula
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for formula missing expression")
	}
}

// --- Rollup config validation tests ---

func TestParse_RollupValid(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
      tasks:
        type: relation
        relation: Tasks
      total_estimate:
        type: rollup
        relation: tasks
        rollup_property: estimate
        function: sum

  - name: Tasks
    properties:
      Name:
        type: title
      estimate:
        type: number
      project:
        type: relation
        relation: Projects
`
	_, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParse_RollupMissingRelation(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Test
    properties:
      Name:
        type: title
      total:
        type: rollup
        rollup_property: estimate
        function: sum
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for rollup missing relation")
	}
}

func TestParse_RollupMissingRollupProperty(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
      tasks:
        type: relation
        relation: Tasks
      total:
        type: rollup
        relation: tasks
        function: sum

  - name: Tasks
    properties:
      Name:
        type: title
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for rollup missing rollup_property")
	}
}

func TestParse_RollupMissingFunction(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
      tasks:
        type: relation
        relation: Tasks
      total:
        type: rollup
        relation: tasks
        rollup_property: estimate

  - name: Tasks
    properties:
      Name:
        type: title
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for rollup missing function")
	}
}

func TestParse_RollupInvalidFunction(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Projects
    properties:
      Name:
        type: title
      tasks:
        type: relation
        relation: Tasks
      total:
        type: rollup
        relation: tasks
        rollup_property: estimate
        function: not_real

  - name: Tasks
    properties:
      Name:
        type: title
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for rollup invalid function")
	}
}

func TestParse_RollupRelationNotARelationProperty(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Test
    properties:
      Name:
        type: title
      description:
        type: rich_text
      total:
        type: rollup
        relation: description
        rollup_property: estimate
        function: sum
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error when rollup relation points to non-relation property")
	}
}

func TestParse_RollupRelationPropertyNotFound(t *testing.T) {
	yaml := `
version: "1"
databases:
  - name: Test
    properties:
      Name:
        type: title
      total:
        type: rollup
        relation: nonexistent
        rollup_property: estimate
        function: sum
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error when rollup relation property does not exist")
	}
}

// --- Tests for env var substitution ---

func TestExpandEnvVars_Basic(t *testing.T) {
	t.Setenv("TEST_PAGE_ID", "abc123")
	result, missing := expandEnvVars(`parent_page_id: "${TEST_PAGE_ID}"`)
	if len(missing) > 0 {
		t.Errorf("unexpected missing vars: %v", missing)
	}
	if result != `parent_page_id: "abc123"` {
		t.Errorf("expected 'abc123', got %q", result)
	}
}

func TestExpandEnvVars_Multiple(t *testing.T) {
	t.Setenv("PAGE_A", "aaa")
	t.Setenv("PAGE_B", "bbb")
	result, missing := expandEnvVars(`a: "${PAGE_A}" b: "${PAGE_B}"`)
	if len(missing) > 0 {
		t.Errorf("unexpected missing vars: %v", missing)
	}
	if result != `a: "aaa" b: "bbb"` {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestExpandEnvVars_Missing(t *testing.T) {
	_, missing := expandEnvVars(`page: "${NONEXISTENT_VAR_12345}"`)
	if len(missing) != 1 || missing[0] != "${NONEXISTENT_VAR_12345}" {
		t.Errorf("expected 1 missing var, got: %v", missing)
	}
}

func TestExpandEnvVars_NoVars(t *testing.T) {
	input := `parent_page_id: "abc123"`
	result, missing := expandEnvVars(input)
	if len(missing) > 0 {
		t.Errorf("unexpected missing vars: %v", missing)
	}
	if result != input {
		t.Errorf("expected no change, got %q", result)
	}
}

func TestExpandEnvVars_EmptyValue(t *testing.T) {
	t.Setenv("EMPTY_VAR", "")
	result, missing := expandEnvVars(`page: "${EMPTY_VAR}"`)
	if len(missing) > 0 {
		t.Errorf("unexpected missing vars: %v", missing)
	}
	if result != `page: ""` {
		t.Errorf("expected empty value, got %q", result)
	}
}

func TestLoad_WithEnvVars(t *testing.T) {
	t.Setenv("TEST_NOTION_PAGE", "page-123")

	tmpFile := t.TempDir() + "/notionctl.yaml"
	yaml := `version: "1"
databases:
  - name: Projects
    parent_page_id: "${TEST_NOTION_PAGE}"
    properties:
      Name:
        type: title
`
	if err := writeFile(tmpFile, yaml); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Databases[0].ParentPageID != "page-123" {
		t.Errorf("expected 'page-123', got %q", cfg.Databases[0].ParentPageID)
	}
}

func TestLoad_MissingEnvVar(t *testing.T) {
	tmpFile := t.TempDir() + "/notionctl.yaml"
	yaml := `version: "1"
databases:
  - name: Projects
    parent_page_id: "${MISSING_VAR_XYZ}"
    properties:
      Name:
        type: title
`
	if err := writeFile(tmpFile, yaml); err != nil {
		t.Fatal(err)
	}

	_, err := Load(tmpFile)
	if err == nil {
		t.Fatal("expected error for missing env var")
	}
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
