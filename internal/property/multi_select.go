package property

import "fmt"

func init() { Register(&MultiSelect{}) }

// MultiSelect represents the Notion "multi_select" property type.
//
// YAML example:
//
//	tags:
//	  type: multi_select
//	  options:
//	    - name: frontend
//	      color: blue
//	    - name: backend
//	      color: green
//	    - name: urgent
//	      color: red
type MultiSelect struct{}

func (m *MultiSelect) Type() string { return "multi_select" }

func (m *MultiSelect) ToNotion(config map[string]interface{}) (NotionPropertyConfig, error) {
	options, err := parseOptions(config)
	if err != nil {
		return nil, fmt.Errorf("multi_select: %w", err)
	}

	return NotionPropertyConfig{
		"multi_select": map[string]interface{}{
			"options": options,
		},
	}, nil
}

func (m *MultiSelect) DiffSummary(desired, current map[string]interface{}) string {
	return diffOptions(desired, current)
}
