package property

func init() { Register(&Email{}) }

// Email represents the Notion "email" property type.
//
// YAML example:
//
//	contact_email:
//	  type: email
type Email struct{}

func (e *Email) Type() string { return "email" }

func (e *Email) ToNotion(_ map[string]interface{}) (NotionPropertyConfig, error) {
	return NotionPropertyConfig{
		"email": map[string]interface{}{},
	}, nil
}

func (e *Email) DiffSummary(desired, current map[string]interface{}) string {
	return ""
}
