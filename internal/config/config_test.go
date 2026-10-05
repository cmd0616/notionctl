package config

import (
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
