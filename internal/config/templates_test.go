package config_test

import (
	"os"
	"testing"

	"github.com/radityajay/notionctl/internal/config"
	_ "github.com/radityajay/notionctl/internal/property"
)

func TestTemplatesValid(t *testing.T) {
	templates := []string{
		"../../templates/project-tracker.yaml",
		"../../templates/crm.yaml",
		"../../templates/inventory.yaml",
	}
	for _, tmpl := range templates {
		t.Run(tmpl, func(t *testing.T) {
			data, err := os.ReadFile(tmpl)
			if err != nil { t.Fatalf("read: %v", err) }
			_, err = config.Parse(data)
			if err != nil { t.Fatalf("parse: %v", err) }
		})
	}
}
