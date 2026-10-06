package engine

import (
	"fmt"
	"sort"
	"strings"

	"github.com/radityajay/notionctl/internal/config"
)

// DiffResult represents the diff between remote Notion state and local config.
type DiffResult struct {
	DatabaseName string
	Status       string   // "in_sync", "drifted", "not_deployed"
	Differences  []string // human-readable list of differences
}

// Diff compares each configured database against its remote Notion state.
// Requires a valid API client and databases to be deployed (have state IDs).
func (e *Engine) Diff() ([]DiffResult, error) {
	var results []DiffResult

	for _, db := range e.config.Databases {
		dbID, exists := e.state.ResolveDatabase(db.Name)
		if !exists {
			results = append(results, DiffResult{
				DatabaseName: db.Name,
				Status:       "not_deployed",
				Differences:  []string{"database not yet applied (no state ID)"},
			})
			continue
		}

		remote, err := e.client.GetDatabase(dbID)
		if err != nil {
			results = append(results, DiffResult{
				DatabaseName: db.Name,
				Status:       "error",
				Differences:  []string{fmt.Sprintf("failed to fetch: %v", err)},
			})
			continue
		}

		diffs := compareDatabase(db, remote)
		status := "in_sync"
		if len(diffs) > 0 {
			status = "drifted"
		}

		results = append(results, DiffResult{
			DatabaseName: db.Name,
			Status:       status,
			Differences:  diffs,
		})
	}

	return results, nil
}

// compareDatabase compares a config database against the remote Notion response.
func compareDatabase(db config.Database, remote map[string]interface{}) []string {
	var diffs []string

	remoteProps, _ := remote["properties"].(map[string]interface{})

	// Check each config property against remote
	for name, prop := range db.Properties {
		remoteProp, exists := remoteProps[name]
		if !exists {
			diffs = append(diffs, fmt.Sprintf("+ property %q (%s): in config but not in Notion", name, prop.Type))
			continue
		}

		propMap, ok := remoteProp.(map[string]interface{})
		if !ok {
			continue
		}

		remoteType, _ := propMap["type"].(string)
		if remoteType != prop.Type {
			diffs = append(diffs, fmt.Sprintf("~ property %q: type config=%s remote=%s", name, prop.Type, remoteType))
			continue
		}

		// Type-specific comparison
		typeDiffs := comparePropertyConfig(name, prop, propMap)
		diffs = append(diffs, typeDiffs...)
	}

	// Check for properties in remote but not in config
	for name := range remoteProps {
		if _, exists := db.Properties[name]; !exists {
			propMap, ok := remoteProps[name].(map[string]interface{})
			remoteType := "unknown"
			if ok {
				if t, ok := propMap["type"].(string); ok {
					remoteType = t
				}
			}
			diffs = append(diffs, fmt.Sprintf("- property %q (%s): in Notion but not in config", name, remoteType))
		}
	}

	sort.Strings(diffs)
	return diffs
}

// comparePropertyConfig compares type-specific config for a property.
func comparePropertyConfig(name string, desired config.PropertyDef, remote map[string]interface{}) []string {
	var diffs []string

	switch desired.Type {
	case "number":
		desiredFmt := "number"
		if f, ok := desired.Extra["format"]; ok {
			desiredFmt = fmt.Sprintf("%v", f)
		}
		remoteFmt := "number"
		if numCfg, ok := remote["number"].(map[string]interface{}); ok {
			if f, ok := numCfg["format"].(string); ok {
				remoteFmt = f
			}
		}
		if desiredFmt != remoteFmt {
			diffs = append(diffs, fmt.Sprintf("~ property %q: format config=%s remote=%s", name, desiredFmt, remoteFmt))
		}

	case "select", "multi_select":
		desiredOpts := extractOptionNames(desired.Extra)
		remoteOpts := extractRemoteOptionNames(remote, desired.Type)
		if diff := compareStringSlices(desiredOpts, remoteOpts); diff != "" {
			diffs = append(diffs, fmt.Sprintf("~ property %q: options %s", name, diff))
		}

	case "status":
		desiredOpts := extractOptionNames(desired.Extra)
		remoteOpts := extractRemoteOptionNames(remote, "status")
		if diff := compareStringSlices(desiredOpts, remoteOpts); diff != "" {
			diffs = append(diffs, fmt.Sprintf("~ property %q: options %s", name, diff))
		}

	case "formula":
		desiredExpr, _ := desired.Extra["expression"].(string)
		remoteExpr := ""
		if fCfg, ok := remote["formula"].(map[string]interface{}); ok {
			if expr, ok := fCfg["expression"].(string); ok {
				remoteExpr = expr
			}
		}
		if desiredExpr != remoteExpr {
			diffs = append(diffs, fmt.Sprintf("~ property %q: expression differs", name))
		}

	case "unique_id":
		desiredPrefix, _ := desired.Extra["prefix"].(string)
		remotePrefix := ""
		if uidCfg, ok := remote["unique_id"].(map[string]interface{}); ok {
			if p, ok := uidCfg["prefix"].(string); ok {
				remotePrefix = p
			}
		}
		if desiredPrefix != remotePrefix {
			diffs = append(diffs, fmt.Sprintf("~ property %q: prefix config=%q remote=%q", name, desiredPrefix, remotePrefix))
		}
	}

	return diffs
}

func extractOptionNames(extra map[string]interface{}) []string {
	opts, ok := extra["options"].([]interface{})
	if !ok {
		return nil
	}
	var names []string
	for _, o := range opts {
		if m, ok := o.(map[string]interface{}); ok {
			if n, ok := m["name"].(string); ok {
				names = append(names, n)
			}
		}
	}
	sort.Strings(names)
	return names
}

func extractRemoteOptionNames(remote map[string]interface{}, propType string) []string {
	typeCfg, ok := remote[propType].(map[string]interface{})
	if !ok {
		return nil
	}
	opts, ok := typeCfg["options"].([]interface{})
	if !ok {
		return nil
	}
	var names []string
	for _, o := range opts {
		if m, ok := o.(map[string]interface{}); ok {
			if n, ok := m["name"].(string); ok {
				names = append(names, n)
			}
		}
	}
	sort.Strings(names)
	return names
}

func compareStringSlices(a, b []string) string {
	aSet := map[string]bool{}
	bSet := map[string]bool{}
	for _, s := range a {
		aSet[s] = true
	}
	for _, s := range b {
		bSet[s] = true
	}

	var added, removed []string
	for _, s := range a {
		if !bSet[s] {
			added = append(added, s)
		}
	}
	for _, s := range b {
		if !aSet[s] {
			removed = append(removed, s)
		}
	}

	if len(added) == 0 && len(removed) == 0 {
		return ""
	}

	var parts []string
	if len(added) > 0 {
		parts = append(parts, fmt.Sprintf("in config but not remote: [%s]", strings.Join(added, ", ")))
	}
	if len(removed) > 0 {
		parts = append(parts, fmt.Sprintf("in remote but not config: [%s]", strings.Join(removed, ", ")))
	}
	return strings.Join(parts, "; ")
}

// FormatDiff returns a human-readable diff output.
func FormatDiff(results []DiffResult) string {
	if len(results) == 0 {
		return "No databases to compare."
	}

	allInSync := true
	for _, r := range results {
		if r.Status != "in_sync" {
			allInSync = false
			break
		}
	}
	if allInSync {
		return "All databases in sync. Notion matches your config."
	}

	var b strings.Builder
	for _, r := range results {
		switch r.Status {
		case "in_sync":
			b.WriteString(fmt.Sprintf("✓ %q — in sync\n", r.DatabaseName))
		case "drifted":
			b.WriteString(fmt.Sprintf("⚠ %q — drifted\n", r.DatabaseName))
			for _, d := range r.Differences {
				b.WriteString(fmt.Sprintf("    %s\n", d))
			}
		case "not_deployed":
			b.WriteString(fmt.Sprintf("○ %q — not deployed\n", r.DatabaseName))
		case "error":
			b.WriteString(fmt.Sprintf("✗ %q — error\n", r.DatabaseName))
			for _, d := range r.Differences {
				b.WriteString(fmt.Sprintf("    %s\n", d))
			}
		}
		b.WriteString("\n")
	}
	return b.String()
}
