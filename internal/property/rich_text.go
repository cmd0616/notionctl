package property

func init() { Register(&RichText{}) }

// RichText represents the Notion "rich_text" property type.
type RichText struct{}

func (r *RichText) Type() string { return "rich_text" }

func (r *RichText) ToNotion(_ map[string]interface{}) (NotionPropertyConfig, error) {
	return NotionPropertyConfig{
		"rich_text": map[string]interface{}{},
	}, nil
}

func (r *RichText) DiffSummary(desired, current map[string]interface{}) string {
	return ""
}
