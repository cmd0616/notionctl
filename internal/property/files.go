package property

func init() { Register(&Files{}) }

// Files represents the Notion "files" property type.
// Allows attaching files and media to a database entry.
//
// YAML example:
//
//	attachments:
//	  type: files
type Files struct{}

func (f *Files) Type() string { return "files" }

func (f *Files) ToNotion(_ map[string]interface{}) (NotionPropertyConfig, error) {
	return NotionPropertyConfig{
		"files": map[string]interface{}{},
	}, nil
}

func (f *Files) DiffSummary(desired, current map[string]interface{}) string {
	return ""
}
