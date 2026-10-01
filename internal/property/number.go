package property

import "fmt"

func init() { Register(&Number{}) }

// Number represents the Notion "number" property type.
//
// YAML example:
//
//	price:
//	  type: number
//	  format: dollar  # optional: number, number_with_commas, percent, dollar, canadian_dollar, euro, pound, yen, ruble, rupee, won, yuan, real, lira, rupiah, franc, hong_kong_dollar, new_zealand_dollar, krona, norwegian_krone, mexican_peso, rand, new_taiwan_dollar, danish_krone, zloty, baht, forint, koruna, shekel, chilean_peso, philippine_peso, dirham, colombian_peso, riyal, ringgit, leu, argentine_peso
type Number struct{}

func (n *Number) Type() string { return "number" }

func (n *Number) ToNotion(config map[string]interface{}) (NotionPropertyConfig, error) {
	format := "number" // default
	if f, ok := config["format"]; ok {
		format = fmt.Sprintf("%v", f)
	}
	return NotionPropertyConfig{
		"number": map[string]interface{}{
			"format": format,
		},
	}, nil
}

func (n *Number) DiffSummary(desired, current map[string]interface{}) string {
	desiredFmt := "number"
	if f, ok := desired["format"]; ok {
		desiredFmt = fmt.Sprintf("%v", f)
	}

	currentFmt := ""
	if f, ok := current["format"]; ok {
		currentFmt = fmt.Sprintf("%v", f)
	}

	if desiredFmt != currentFmt {
		return fmt.Sprintf("format: %s → %s", currentFmt, desiredFmt)
	}
	return ""
}
