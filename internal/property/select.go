package property

import (
	"fmt"
	"sort"
	"strings"
)

func init() { Register(&Select{}) }

// Select represents the Notion "select" property type.
//
// YAML example:
//
//	status:
//	  type: select
//	  options:
//	    - name: To Do
//	      color: red
//	    - name: In Progress
//	      color: yellow
//	    - name: Done
//	      color: green
type Select struct{}

func (s *Select) Type() string { return "select" }

func (s *Select) ToNotion(config map[string]interface{}) (NotionPropertyConfig, error) {
	options, err := parseOptions(config)
	if err != nil {
		return nil, fmt.Errorf("select: %w", err)
	}

	return NotionPropertyConfig{
		"select": map[string]interface{}{
			"options": options,
		},
	}, nil
}

func (s *Select) DiffSummary(desired, current map[string]interface{}) string {
	return diffOptions(desired, current)
}

// parseOptions extracts []map[string]interface{} from the "options" key.
// Shared by Select and MultiSelect.
func parseOptions(config map[string]interface{}) ([]map[string]interface{}, error) {
	raw, ok := config["options"]
	if !ok {
		return []map[string]interface{}{}, nil
	}

	rawSlice, ok := raw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("options must be a list")
	}

	options := make([]map[string]interface{}, 0, len(rawSlice))
	for _, item := range rawSlice {
		m, ok := item.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("each option must be a map with 'name' and optional 'color'")
		}
		opt := map[string]interface{}{}
		if name, ok := m["name"]; ok {
			opt["name"] = fmt.Sprintf("%v", name)
		}
		if color, ok := m["color"]; ok {
			opt["color"] = fmt.Sprintf("%v", color)
		}
		options = append(options, opt)
	}
	return options, nil
}

// diffOptions compares option lists and returns a human-readable diff.
// Shared by Select and MultiSelect.
func diffOptions(desired, current map[string]interface{}) string {
	desiredNames := optionNames(desired)
	currentNames := optionNames(current)

	sort.Strings(desiredNames)
	sort.Strings(currentNames)

	if strings.Join(desiredNames, ",") != strings.Join(currentNames, ",") {
		return fmt.Sprintf("options: [%s] → [%s]",
			strings.Join(currentNames, ", "),
			strings.Join(desiredNames, ", "))
	}
	return ""
}

func optionNames(config map[string]interface{}) []string {
	raw, ok := config["options"]
	if !ok {
		return nil
	}
	rawSlice, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	names := make([]string, 0, len(rawSlice))
	for _, item := range rawSlice {
		if m, ok := item.(map[string]interface{}); ok {
			if name, ok := m["name"]; ok {
				names = append(names, fmt.Sprintf("%v", name))
			}
		}
	}
	return names
}
