package config

import (
	"strings"
	"testing"

	_ "github.com/radityajay/notionctl/internal/property"
)

func TestFormat_SortsDatabases(t *testing.T) {
	cfg := &Config{
		Version: "1",
		Databases: []Database{
			{Name: "Zebra", Properties: map[string]PropertyDef{"Name": {Type: "title"}}},
			{Name: "Alpha", Properties: map[string]PropertyDef{"Name": {Type: "title"}}},
		},
	}

	data, err := Format(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := string(data)
	alphaIdx := strings.Index(out, "Alpha")
	zebraIdx := strings.Index(out, "Zebra")
	if alphaIdx > zebraIdx {
		t.Error("expected Alpha before Zebra")
	}
}

func TestFormat_TitlePropertyFirst(t *testing.T) {
	cfg := &Config{
		Version: "1",
		Databases: []Database{
			{
				Name: "Test",
				Properties: map[string]PropertyDef{
					"Zebra": {Type: "rich_text"},
					"Name":  {Type: "title"},
					"Alpha": {Type: "checkbox"},
				},
			},
		},
	}

	data, err := Format(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := string(data)
	nameIdx := strings.Index(out, "Name:")
	alphaIdx := strings.Index(out, "Alpha:")
	zebraIdx := strings.Index(out, "Zebra:")

	if nameIdx > alphaIdx {
		t.Error("expected Name (title) before Alpha")
	}
	if alphaIdx > zebraIdx {
		t.Error("expected Alpha before Zebra (alphabetical)")
	}
}

func TestFormat_TypeFieldFirst(t *testing.T) {
	cfg := &Config{
		Version: "1",
		Databases: []Database{
			{
				Name: "Test",
				Properties: map[string]PropertyDef{
					"Name": {Type: "title"},
					"Price": {
						Type:  "number",
						Extra: map[string]interface{}{"format": "dollar"},
					},
				},
			},
		},
	}

	data, err := Format(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := string(data)
	// In the Price property, "type" should come before "format"
	priceSection := out[strings.Index(out, "Price:"):]
	typeIdx := strings.Index(priceSection, "type:")
	formatIdx := strings.Index(priceSection, "format:")
	if typeIdx > formatIdx {
		t.Error("expected 'type' before 'format' in property definition")
	}
}

func TestFormatFile_Idempotent(t *testing.T) {
	tmpFile := t.TempDir() + "/test.yaml"
	yaml := `version: "1"
databases:
    - name: Projects
      properties:
        Name:
            type: title
        Desc:
            type: rich_text
`
	if err := writeFile(tmpFile, yaml); err != nil {
		t.Fatal(err)
	}

	// First format
	changed1, err := FormatFile(tmpFile)
	if err != nil {
		t.Fatalf("first format error: %v", err)
	}

	// Second format should be no-op
	changed2, err := FormatFile(tmpFile)
	if err != nil {
		t.Fatalf("second format error: %v", err)
	}
	if changed2 {
		t.Error("expected second format to be no-op (idempotent)")
	}
	_ = changed1
}
