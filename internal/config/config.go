// Package config handles parsing and validating notionctl YAML files.
package config

import (
	"fmt"
	"os"

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
		for propName, prop := range db.Properties {
			if prop.Type == "" {
				return fmt.Errorf("database %q, property %q: missing 'type'", db.Name, propName)
			}
			if prop.Type == "title" {
				hasTitle = true
			}
		}
		if !hasTitle {
			return fmt.Errorf("database %q: must have exactly one 'title' property", db.Name)
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
