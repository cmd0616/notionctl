package property

func init() { Register(&Date{}) }

// Date represents the Notion "date" property type.
//
// YAML example:
//
//	due_date:
//	  type: date
type Date struct{}

func (d *Date) Type() string { return "date" }

func (d *Date) ToNotion(_ map[string]interface{}) (NotionPropertyConfig, error) {
	return NotionPropertyConfig{
		"date": map[string]interface{}{},
	}, nil
}

func (d *Date) DiffSummary(desired, current map[string]interface{}) string {
	return ""
}
