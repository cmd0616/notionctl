package property

func init() { Register(&People{}) }

// People represents the Notion "people" property type.
// Stores references to Notion users (workspace members).
//
// YAML example:
//
//	assignee:
//	  type: people
type People struct{}

func (p *People) Type() string { return "people" }

func (p *People) ToNotion(_ map[string]interface{}) (NotionPropertyConfig, error) {
	return NotionPropertyConfig{
		"people": map[string]interface{}{},
	}, nil
}

func (p *People) DiffSummary(desired, current map[string]interface{}) string {
	return ""
}
