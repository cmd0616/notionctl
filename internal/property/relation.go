package property

import "fmt"

func init() { Register(&Relation{}) }

// Relation represents the Notion "relation" property type.
// This is the key differentiator of notionctl — relations are specified
// by symbolic database name, not by raw Notion database ID.
//
// YAML example:
//
//	tasks:
//	  type: relation
//	  relation: Tasks          # symbolic name, resolved at apply time
//	  synced_property: project  # optional: creates a two-way relation
//
// At apply time, the engine resolves "Tasks" to the actual Notion database ID
// from the state file.
type Relation struct{}

func (r *Relation) Type() string { return "relation" }

func (r *Relation) ToNotion(config map[string]interface{}) (NotionPropertyConfig, error) {
	dbRef, ok := config["relation"]
	if !ok {
		return nil, fmt.Errorf("relation property requires 'relation' field with target database name")
	}

	// At this stage we store the symbolic name; the engine will resolve it
	// to an actual database_id before sending to the Notion API.
	result := NotionPropertyConfig{
		"relation": map[string]interface{}{
			"database_id":    fmt.Sprintf("{{resolve:%v}}", dbRef),
			"type":           "single_property",
			"single_property": map[string]interface{}{},
		},
	}

	// Two-way relation support
	if synced, ok := config["synced_property"]; ok {
		result["relation"].(map[string]interface{})["type"] = "dual_property"
		result["relation"].(map[string]interface{})["dual_property"] = map[string]interface{}{
			"synced_property_name": fmt.Sprintf("%v", synced),
		}
		delete(result["relation"].(map[string]interface{}), "single_property")
	}

	return result, nil
}

func (r *Relation) DiffSummary(desired, current map[string]interface{}) string {
	desiredRef := fmt.Sprintf("%v", desired["relation"])
	currentRef := fmt.Sprintf("%v", current["relation"])

	if desiredRef != currentRef {
		return fmt.Sprintf("target: %s → %s", currentRef, desiredRef)
	}
	return ""
}
