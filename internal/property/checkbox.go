package property

func init() { Register(&Checkbox{}) }

// Checkbox represents the Notion "checkbox" property type.
//
// YAML example:
//
//	done:
//	  type: checkbox
type Checkbox struct{}

func (c *Checkbox) Type() string { return "checkbox" }

func (c *Checkbox) ToNotion(_ map[string]interface{}) (NotionPropertyConfig, error) {
	return NotionPropertyConfig{
		"checkbox": map[string]interface{}{},
	}, nil
}

func (c *Checkbox) DiffSummary(desired, current map[string]interface{}) string {
	return ""
}
