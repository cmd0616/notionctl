// Package config handles parsing and validating notionctl YAML files.
package config

import (
	"fmt"
	"os"

	"github.com/radityajay/notionctl/internal/property"
	"gopkg.in/yaml.v3"
)

// Config is the top-level YAML schema for notionctl.
type Config struct {
	// Version of the config schema. Currently "1".
	Version string `yaml:"version"`

	// Databases to manage in the Notion workspace.
	Databases []Database `yaml:"databases"`
}

// Database represents a single Notion database declaration.
type Database struct {
	// Name is the symbolic name used for references (e.g., in relations).
	// Also used as the database title in Notion if not already created.
	Name string `yaml:"name"`

	// ParentPageID is the Notion page ID where this database will be created.
	// Required for new databases, ignored for existing ones (tracked in state).
	ParentPageID string `yaml:"parent_page_id,omitempty"`

	// Properties defines the database schema.
	// Key is the property name, value contains type and type-specific config.
	Properties map[string]PropertyDef `yaml:"properties"`
}

// PropertyDef represents a property definition in YAML.
type PropertyDef struct {
	Type string `yaml:"type"`

	// Remaining fields are type-specific and passed to the Property handler.
	// We use inline yaml to capture all extra fields.
	Extra map[string]interface{} `yaml:",inline"`
}

// Load reads and parses a notionctl YAML config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	return Parse(data)
}

// Parse parses raw YAML bytes into a Config.
func Parse(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate checks the config for logical errors.
func (c *Config) Validate() error {
	if c.Version == "" {
		return fmt.Errorf("missing 'version' field")
	}
	if c.Version != "1" {
		return fmt.Errorf("unsupported version %q (supported: 1)", c.Version)
	}
	if len(c.Databases) == 0 {
		return fmt.Errorf("at least one database must be defined")
	}

	names := map[string]bool{}
	for _, db := range c.Databases {
		if db.Name == "" {
			return fmt.Errorf("database name cannot be empty")
		}
		if names[db.Name] {
			return fmt.Errorf("duplicate database name: %q", db.Name)
		}
		names[db.Name] = true

		if len(db.Properties) == 0 {
			return fmt.Errorf("database %q must have at least one property", db.Name)
		}

		hasTitle := false
		titleCount := 0
		for propName, prop := range db.Properties {
			if prop.Type == "" {
				return fmt.Errorf("database %q, property %q: missing 'type'", db.Name, propName)
			}
			if prop.Type == "title" {
				titleCount++
				hasTitle = true
			}

			// Validate property type is registered
			if _, err := property.Get(prop.Type); err != nil {
				supported := property.SupportedTypes()
				return fmt.Errorf("database %q, property %q: unknown type %q (supported: %v)", db.Name, propName, prop.Type, supported)
			}

			// Validate type-specific config
			if err := validatePropertyConfig(db.Name, propName, prop); err != nil {
				return err
			}
		}
		if !hasTitle {
			return fmt.Errorf("database %q: must have exactly one 'title' property", db.Name)
		}
		if titleCount > 1 {
			return fmt.Errorf("database %q: must have exactly one 'title' property, found %d", db.Name, titleCount)
		}
	}

	// Validate relation references
	for _, db := range c.Databases {
		for propName, prop := range db.Properties {
			if prop.Type == "relation" {
				ref, ok := prop.Extra["relation"]
				if !ok {
					return fmt.Errorf("database %q, property %q: relation requires 'relation' field", db.Name, propName)
				}
				refName := fmt.Sprintf("%v", ref)
				if !names[refName] {
					return fmt.Errorf("database %q, property %q: relation target %q not found in config", db.Name, propName, refName)
				}
			}
		}
	}

	return nil
}

// DatabaseNames returns all declared database names.
func (c *Config) DatabaseNames() []string {
	names := make([]string, len(c.Databases))
	for i, db := range c.Databases {
		names[i] = db.Name
	}
	return names
}

// validNumberFormats lists all valid Notion number formats.
var validNumberFormats = map[string]bool{
	"number": true, "number_with_commas": true, "percent": true,
	"dollar": true, "canadian_dollar": true, "euro": true, "pound": true,
	"yen": true, "ruble": true, "rupee": true, "won": true, "yuan": true,
	"real": true, "lira": true, "rupiah": true, "franc": true,
	"hong_kong_dollar": true, "new_zealand_dollar": true, "krona": true,
	"norwegian_krone": true, "mexican_peso": true, "rand": true,
	"new_taiwan_dollar": true, "danish_krone": true, "zloty": true,
	"baht": true, "forint": true, "koruna": true, "shekel": true,
	"chilean_peso": true, "philippine_peso": true, "dirham": true,
	"colombian_peso": true, "riyal": true, "ringgit": true, "leu": true,
	"argentine_peso": true,
}

// validatePropertyConfig checks type-specific configuration for common mistakes.
func validatePropertyConfig(dbName, propName string, prop PropertyDef) error {
	switch prop.Type {
	case "number":
		if f, ok := prop.Extra["format"]; ok {
			format := fmt.Sprintf("%v", f)
			if !validNumberFormats[format] {
				return fmt.Errorf("database %q, property %q: unknown number format %q", dbName, propName, format)
			}
		}

	case "select", "multi_select":
		if opts, ok := prop.Extra["options"]; ok {
			optSlice, ok := opts.([]interface{})
			if !ok {
				return fmt.Errorf("database %q, property %q: 'options' must be a list", dbName, propName)
			}
			for i, item := range optSlice {
				m, ok := item.(map[string]interface{})
				if !ok {
					return fmt.Errorf("database %q, property %q: option %d must be a map with 'name'", dbName, propName, i+1)
				}
				if _, ok := m["name"]; !ok {
					return fmt.Errorf("database %q, property %q: option %d missing 'name'", dbName, propName, i+1)
				}
			}
		}

	case "status":
		if opts, ok := prop.Extra["options"]; ok {
			optSlice, ok := opts.([]interface{})
			if !ok {
				return fmt.Errorf("database %q, property %q: 'options' must be a list", dbName, propName)
			}
			for i, item := range optSlice {
				m, ok := item.(map[string]interface{})
				if !ok {
					return fmt.Errorf("database %q, property %q: option %d must be a map with 'name'", dbName, propName, i+1)
				}
				if _, ok := m["name"]; !ok {
					return fmt.Errorf("database %q, property %q: option %d missing 'name'", dbName, propName, i+1)
				}
			}
		}
		if groups, ok := prop.Extra["groups"]; ok {
			groupSlice, ok := groups.([]interface{})
			if !ok {
				return fmt.Errorf("database %q, property %q: 'groups' must be a list", dbName, propName)
			}
			for i, item := range groupSlice {
				m, ok := item.(map[string]interface{})
				if !ok {
					return fmt.Errorf("database %q, property %q: group %d must be a map with 'name'", dbName, propName, i+1)
				}
				if _, ok := m["name"]; !ok {
					return fmt.Errorf("database %q, property %q: group %d missing 'name'", dbName, propName, i+1)
				}
			}
		}
	}
	return nil
}
