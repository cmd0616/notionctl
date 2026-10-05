package property

import (
	"fmt"
	"sort"
	"strings"
)

func init() { Register(&Status{}) }

// Status represents the Notion "status" property type.
//
// Status has options grouped into three fixed groups: To-do, In progress, Complete.
// Unlike select, Notion manages these groups — you declare options and assign them to groups.
//
// YAML example:
//
//	status:
//	  type: status
//	  options:
//	    - name: Not Started
//	      color: default
//	    - name: In Progress
//	      color: blue
//	    - name: Done
//	      color: green
//	  groups:
//	    - name: To-do
//	      option_ids:
//	        - Not Started
//	    - name: In progress
//	      option_ids:
//	        - In Progress
//	    - name: Complete
//	      option_ids:
//	        - Done
type Status struct{}

func (s *Status) Type() string { return "status" }

func (s *Status) ToNotion(config map[string]interface{}) (NotionPropertyConfig, error) {
	result := map[string]interface{}{}

	// Parse options
	if rawOpts, ok := config["options"]; ok {
		rawSlice, ok := rawOpts.([]interface{})
		if !ok {
			return nil, fmt.Errorf("status: options must be a list")
		}
		options := make([]map[string]interface{}, 0, len(rawSlice))
		for _, item := range rawSlice {
			m, ok := item.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("status: each option must be a map with 'name' and optional 'color'")
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
		result["options"] = options
	}

	// Parse groups
	if rawGroups, ok := config["groups"]; ok {
		rawSlice, ok := rawGroups.([]interface{})
		if !ok {
			return nil, fmt.Errorf("status: groups must be a list")
		}
		groups := make([]map[string]interface{}, 0, len(rawSlice))
		for _, item := range rawSlice {
			m, ok := item.(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("status: each group must be a map with 'name' and 'option_ids'")
			}
			group := map[string]interface{}{}
			if name, ok := m["name"]; ok {
				group["name"] = fmt.Sprintf("%v", name)
			}
			if ids, ok := m["option_ids"]; ok {
				group["option_ids"] = ids
			}
			groups = append(groups, group)
		}
		result["groups"] = groups
	}

	return NotionPropertyConfig{
		"status": result,
	}, nil
}

func (s *Status) DiffSummary(desired, current map[string]interface{}) string {
	desiredNames := statusOptionNames(desired)
	currentNames := statusOptionNames(current)

	sort.Strings(desiredNames)
	sort.Strings(currentNames)

	if strings.Join(desiredNames, ",") != strings.Join(currentNames, ",") {
		return fmt.Sprintf("options: [%s] → [%s]",
			strings.Join(currentNames, ", "),
			strings.Join(desiredNames, ", "))
	}
	return ""
}

func statusOptionNames(config map[string]interface{}) []string {
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
