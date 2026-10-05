package property

func init() { Register(&LastEditedTime{}) }

// LastEditedTime represents a timestamp maintained by Notion, not a writable page value.
type LastEditedTime struct{}

func (p *LastEditedTime) Type() string { return "last_edited_time" }

func (p *LastEditedTime) ToNotion(_ map[string]interface{}) (NotionPropertyConfig, error) {
	return NotionPropertyConfig{"last_edited_time": map[string]interface{}{}}, nil
}

func (p *LastEditedTime) DiffSummary(_, _ map[string]interface{}) string { return "" }
