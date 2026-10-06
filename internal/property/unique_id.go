package property

import "fmt"

func init() { Register(&UniqueID{}) }

// UniqueID represents the Notion "unique_id" property type.
// Auto-generated sequential IDs managed by Notion.
//
// YAML example:
//
//	ticket_id:
//	  type: unique_id
//	  prefix: TICKET  # optional prefix for display (e.g., TICKET-1, TICKET-2)
type UniqueID struct{}

func (u *UniqueID) Type() string { return "unique_id" }

func (u *UniqueID) ToNotion(config map[string]interface{}) (NotionPropertyConfig, error) {
	innerCfg := map[string]interface{}{}
	if prefix, ok := config["prefix"]; ok {
		innerCfg["prefix"] = fmt.Sprintf("%v", prefix)
	}
	return NotionPropertyConfig{
		"unique_id": innerCfg,
	}, nil
}

func (u *UniqueID) DiffSummary(desired, current map[string]interface{}) string {
	desiredPrefix := ""
	if p, ok := desired["prefix"]; ok {
		desiredPrefix = fmt.Sprintf("%v", p)
	}

	currentPrefix := ""
	if p, ok := current["prefix"]; ok {
		currentPrefix = fmt.Sprintf("%v", p)
	}

	if desiredPrefix != currentPrefix {
		return fmt.Sprintf("prefix: %q → %q", currentPrefix, desiredPrefix)
	}
	return ""
}
