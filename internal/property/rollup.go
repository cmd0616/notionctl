package property

import (
	"fmt"
	"strings"
)

func init() { Register(&Rollup{}) }

// Rollup represents the Notion "rollup" property type.
//
// YAML example:
//
//	total_estimate:
//	  type: rollup
//	  relation: tasks           # name of a relation property in this database
//	  rollup_property: estimate  # name of a property in the target database
//	  function: sum              # aggregation function
//
// The relation field refers to a relation property by name in the same
// database. The rollup_property refers to a property in the target database
// that the relation points to. At apply time, these are sent as
// relation_property_name and rollup_property_name to the Notion API.
type Rollup struct{}

// ValidRollupFunctions lists all aggregation functions supported by the Notion API.
var ValidRollupFunctions = map[string]bool{
	"count":             true,
	"count_values":      true,
	"empty":             true,
	"not_empty":         true,
	"unique":            true,
	"show_unique":       true,
	"percent_empty":     true,
	"percent_not_empty": true,
	"sum":               true,
	"average":           true,
	"median":            true,
	"min":               true,
	"max":               true,
	"range":             true,
	"earliest_date":     true,
	"latest_date":       true,
	"date_range":        true,
	"checked":           true,
	"unchecked":         true,
	"percent_checked":   true,
	"percent_unchecked": true,
	"count_per_group":   true,
	"percent_per_group": true,
	"show_original":     true,
}

func (r *Rollup) Type() string { return "rollup" }

func (r *Rollup) ToNotion(config map[string]interface{}) (NotionPropertyConfig, error) {
	rel, ok := config["relation"]
	if !ok {
		return nil, fmt.Errorf("rollup property requires 'relation' field (name of a relation property in this database)")
	}

	rollupProp, ok := config["rollup_property"]
	if !ok {
		return nil, fmt.Errorf("rollup property requires 'rollup_property' field (name of a property in the target database)")
	}

	fn, ok := config["function"]
	if !ok {
		return nil, fmt.Errorf("rollup property requires 'function' field")
	}

	fnStr := fmt.Sprintf("%v", fn)
	if !ValidRollupFunctions[fnStr] {
		valid := make([]string, 0, len(ValidRollupFunctions))
		for k := range ValidRollupFunctions {
			valid = append(valid, k)
		}
		return nil, fmt.Errorf("rollup: unknown function %q (supported: %s)", fnStr, strings.Join(valid, ", "))
	}

	return NotionPropertyConfig{
		"rollup": map[string]interface{}{
			"relation_property_name": fmt.Sprintf("%v", rel),
			"rollup_property_name":   fmt.Sprintf("%v", rollupProp),
			"function":               fnStr,
		},
	}, nil
}

func (r *Rollup) DiffSummary(desired, current map[string]interface{}) string {
	var diffs []string

	desiredRel := fmt.Sprintf("%v", desired["relation"])
	currentRel := fmt.Sprintf("%v", current["relation"])
	if desiredRel != currentRel {
		diffs = append(diffs, fmt.Sprintf("relation: %s → %s", currentRel, desiredRel))
	}

	desiredProp := fmt.Sprintf("%v", desired["rollup_property"])
	currentProp := fmt.Sprintf("%v", current["rollup_property"])
	if desiredProp != currentProp {
		diffs = append(diffs, fmt.Sprintf("rollup_property: %s → %s", currentProp, desiredProp))
	}

	desiredFn := fmt.Sprintf("%v", desired["function"])
	currentFn := fmt.Sprintf("%v", current["function"])
	if desiredFn != currentFn {
		diffs = append(diffs, fmt.Sprintf("function: %s → %s", currentFn, desiredFn))
	}

	return strings.Join(diffs, ", ")
}
