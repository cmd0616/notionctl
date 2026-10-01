// Package property defines the interface for Notion property types.
//
// Adding a new property type:
//  1. Create a new file (e.g., checkbox.go)
//  2. Implement the Property interface
//  3. Register it in an init() function using Register()
//
// See title.go for a minimal example.
package property

import "fmt"

// NotionPropertyConfig represents the Notion API payload for a property.
type NotionPropertyConfig map[string]interface{}

// Property is the interface that all Notion property types must implement.
// Each property type knows how to convert between YAML config and Notion API format.
type Property interface {
	// Type returns the Notion property type string (e.g., "title", "rich_text").
	Type() string

	// ToNotion converts YAML config into a Notion API property configuration.
	// The config parameter contains the user-defined options from the YAML file.
	ToNotion(config map[string]interface{}) (NotionPropertyConfig, error)

	// DiffSummary compares desired (from YAML) and current (from Notion API) configs
	// and returns a human-readable summary of changes. Empty string means no change.
	DiffSummary(desired, current map[string]interface{}) string
}

// registry holds all registered property types.
var registry = map[string]Property{}

// Register adds a property type to the global registry.
// Called from init() in each property type file.
func Register(p Property) {
	if _, exists := registry[p.Type()]; exists {
		panic(fmt.Sprintf("property type %q already registered", p.Type()))
	}
	registry[p.Type()] = p
}

// Get returns the property handler for the given type, or an error if unknown.
func Get(typeName string) (Property, error) {
	p, ok := registry[typeName]
	if !ok {
		return nil, fmt.Errorf("unknown property type: %q (is it registered?)", typeName)
	}
	return p, nil
}

// SupportedTypes returns all registered property type names.
func SupportedTypes() []string {
	types := make([]string, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	return types
}
