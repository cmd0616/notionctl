package property

func init() { Register(&Title{}) }

// Title represents the Notion "title" property type.
// Every database must have exactly one title property.
type Title struct{}

func (t *Title) Type() string { return "title" }

func (t *Title) ToNotion(_ map[string]interface{}) (NotionPropertyConfig, error) {
	return NotionPropertyConfig{
		"title": map[string]interface{}{},
	}, nil
}

func (t *Title) DiffSummary(desired, current map[string]interface{}) string {
	// Title properties have no configurable options — if it exists, it matches.
	return ""
}
