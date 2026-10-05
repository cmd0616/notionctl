package property

func init() { Register(&PhoneNumber{}) }

// PhoneNumber represents the Notion "phone_number" property type.
//
// YAML example:
//
//	phone:
//	  type: phone_number
type PhoneNumber struct{}

func (p *PhoneNumber) Type() string { return "phone_number" }

func (p *PhoneNumber) ToNotion(_ map[string]interface{}) (NotionPropertyConfig, error) {
	return NotionPropertyConfig{
		"phone_number": map[string]interface{}{},
	}, nil
}

func (p *PhoneNumber) DiffSummary(desired, current map[string]interface{}) string {
	return ""
}
