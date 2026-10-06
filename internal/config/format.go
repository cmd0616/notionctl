package config

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Format takes a parsed Config and produces clean, consistently formatted YAML.
// Databases are sorted by name. Properties are sorted with title first, then alphabetically.
func Format(cfg *Config) ([]byte, error) {
	type yamlDB struct {
		Name         string                 `yaml:"name"`
		ParentPageID string                 `yaml:"parent_page_id,omitempty"`
		Properties   yaml.Node              `yaml:"properties"`
	}

	type yamlConfig struct {
		Version   string   `yaml:"version"`
		Databases []yamlDB `yaml:"databases"`
	}

	// Sort databases by name
	sorted := make([]Database, len(cfg.Databases))
	copy(sorted, cfg.Databases)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Name < sorted[j].Name
	})

	out := yamlConfig{Version: cfg.Version}

	for _, db := range sorted {
		propsNode := buildPropertiesNode(db.Properties)
		out.Databases = append(out.Databases, yamlDB{
			Name:         db.Name,
			ParentPageID: db.ParentPageID,
			Properties:   propsNode,
		})
	}

	data, err := yaml.Marshal(out)
	if err != nil {
		return nil, fmt.Errorf("formatting YAML: %w", err)
	}

	return data, nil
}

// buildPropertiesNode creates a yaml.Node with sorted property keys.
// Title property comes first, then alphabetical order.
func buildPropertiesNode(props map[string]PropertyDef) yaml.Node {
	// Separate title from rest, sort rest alphabetically
	var titleName string
	var otherNames []string
	for name, prop := range props {
		if prop.Type == "title" {
			titleName = name
		} else {
			otherNames = append(otherNames, name)
		}
	}
	sort.Strings(otherNames)

	// Build ordered list: title first, then rest
	var ordered []string
	if titleName != "" {
		ordered = append(ordered, titleName)
	}
	ordered = append(ordered, otherNames...)

	// Build yaml.Node mapping
	node := yaml.Node{
		Kind: yaml.MappingNode,
	}
	for _, name := range ordered {
		prop := props[name]

		// Key node
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: name,
		}

		// Value node — build as mapping
		valueNode := buildPropertyValueNode(prop)

		node.Content = append(node.Content, keyNode, valueNode)
	}

	return node
}

// buildPropertyValueNode creates a yaml.Node for a single property definition.
func buildPropertyValueNode(prop PropertyDef) *yaml.Node {
	// Build a map: type first, then extra fields sorted
	propMap := map[string]interface{}{
		"type": prop.Type,
	}
	for k, v := range prop.Extra {
		propMap[k] = v
	}

	// Sort keys: type first, then alphabetical
	var keys []string
	for k := range propMap {
		if k != "type" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	orderedKeys := append([]string{"type"}, keys...)

	node := &yaml.Node{
		Kind: yaml.MappingNode,
	}
	for _, k := range orderedKeys {
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: k,
		}

		v := propMap[k]
		var valNode *yaml.Node

		// Marshal value to yaml.Node
		valBytes, _ := yaml.Marshal(v)
		var tmp yaml.Node
		if err := yaml.Unmarshal(valBytes, &tmp); err == nil && len(tmp.Content) > 0 {
			valNode = tmp.Content[0]
		} else {
			valNode = &yaml.Node{
				Kind:  yaml.ScalarNode,
				Value: fmt.Sprintf("%v", v),
			}
		}

		node.Content = append(node.Content, keyNode, valNode)
	}

	return node
}

// FormatFile reads, parses, formats, and writes back a config file.
// Returns true if the file was changed.
func FormatFile(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("reading config: %w", err)
	}

	cfg, err := Parse(data)
	if err != nil {
		return false, err
	}

	formatted, err := Format(cfg)
	if err != nil {
		return false, err
	}

	// Compare
	original := strings.TrimSpace(string(data))
	result := strings.TrimSpace(string(formatted))

	if original == result {
		return false, nil
	}

	if err := os.WriteFile(path, formatted, 0o644); err != nil {
		return false, fmt.Errorf("writing config: %w", err)
	}

	return true, nil
}
