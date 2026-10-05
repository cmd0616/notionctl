package property

func init() { Register(&URL{}) }

// URL represents the Notion "url" property type.
//
// YAML example:
//
//	website:
//	  type: url
type URL struct{}

func (u *URL) Type() string { return "url" }

func (u *URL) ToNotion(_ map[string]interface{}) (NotionPropertyConfig, error) {
	return NotionPropertyConfig{
		"url": map[string]interface{}{},
	}, nil
}

func (u *URL) DiffSummary(desired, current map[string]interface{}) string {
	return ""
}
