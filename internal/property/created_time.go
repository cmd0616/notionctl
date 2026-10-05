package property

func init() { Register(&CreatedTime{}) }

// CreatedTime represents the Notion "created_time" property type.
// This is a read-only property — Notion sets the value automatically.
//
// YAML example:
//
//	created:
//	  type: created_time
type CreatedTime struct{}

func (c *CreatedTime) Type() string { return "created_time" }

func (c *CreatedTime) ToNotion(_ map[string]interface{}) (NotionPropertyConfig, error) {
	return NotionPropertyConfig{
		"created_time": map[string]interface{}{},
	}, nil
}

func (c *CreatedTime) DiffSummary(desired, current map[string]interface{}) string {
	return ""
}
