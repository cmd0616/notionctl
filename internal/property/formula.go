package property

import "fmt"

func init() { Register(&Formula{}) }

// Formula represents the Notion "formula" property type.
//
// YAML example:
//
//	double_estimate:
//	  type: formula
//	  expression: 'prop("Estimate") * 2'
//
// The expression uses Notion's formula syntax. The computed page value
// is read-only — notionctl only manages the schema (expression definition).
type Formula struct{}

func (f *Formula) Type() string { return "formula" }

func (f *Formula) ToNotion(config map[string]interface{}) (NotionPropertyConfig, error) {
	expr, ok := config["expression"]
	if !ok {
		return nil, fmt.Errorf("formula property requires 'expression' field")
	}

	return NotionPropertyConfig{
		"formula": map[string]interface{}{
			"expression": fmt.Sprintf("%v", expr),
		},
	}, nil
}

func (f *Formula) DiffSummary(desired, current map[string]interface{}) string {
	desiredExpr := fmt.Sprintf("%v", desired["expression"])
	currentExpr := fmt.Sprintf("%v", current["expression"])

	if desiredExpr != currentExpr {
		return fmt.Sprintf("expression: %q → %q", currentExpr, desiredExpr)
	}
	return ""
}
