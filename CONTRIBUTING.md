# Contributing to notionctl

Thanks for your interest! This project is designed to make contributing easy.

## Quick Start

```bash
git clone https://github.com/radityajay/notionctl.git
cd notionctl
go test ./...
go build -o notionctl .
```

## Add a New Property Type (Easiest Contribution)

Every Notion property type is a single Go file implementing one interface. This is the fastest way to contribute.

### Step 1: Create the file

```bash
touch internal/property/checkbox.go
```

### Step 2: Implement the interface

```go
package property

func init() { Register(&Checkbox{}) }

type Checkbox struct{}

func (c *Checkbox) Type() string { return "checkbox" }

func (c *Checkbox) ToNotion(config map[string]interface{}) (NotionPropertyConfig, error) {
    return NotionPropertyConfig{
        "checkbox": map[string]interface{}{},
    }, nil
}

func (c *Checkbox) DiffSummary(desired, current map[string]interface{}) string {
    return ""
}
```

### Step 3: Add a test

Add test cases to `internal/property/property_test.go` or create a dedicated test file.

### Step 4: Done!

The `init()` function auto-registers your type. No other files need to change.

### Available Property Types to Add

All core Notion property types are already implemented. Check the [Notion API docs](https://developers.notion.com/reference/property-object) for any new types that Notion may add in the future. Some types that could still be contributed:

| Type | Difficulty | Notes |
|------|-----------|-------|
| `people` | Easy | No config needed |
| `files` | Easy | No config needed |
| `unique_id` | Easy | `prefix` config |

## Add a YAML Template (No Go Required!)

Create a YAML file in `templates/` with a common use case:

```
templates/
  bug-tracker.yaml
  crm.yaml
  inventory.yaml
  project-tracker.yaml
```

Each template should be a valid `notionctl.yaml` with `parent_page_id: "YOUR_PAGE_ID"` as placeholder.

## Code Style

- Run `go vet ./...` and `go test ./...` before submitting
- Follow standard Go conventions
- Keep property implementations simple — one file, one type
- Add comments explaining any Notion API quirks

## Pull Requests

1. Fork and create a feature branch
2. Make your changes
3. Run tests: `go test ./...`
4. Submit a PR with a clear description

## Project Structure

```
notionctl/
├── cmd/                    # CLI commands (cobra)
├── internal/
│   ├── config/             # YAML parsing & validation
│   ├── engine/             # Diff & reconcile logic
│   ├── notion/             # Notion API client
│   ├── property/           # ← Property types live here
│   └── state/              # State file management
├── main.go
└── notionctl.yaml          # Example config
```

## Questions?

Open an issue — happy to help!
