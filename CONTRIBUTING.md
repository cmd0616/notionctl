# Contributing to notionctl

Thanks for your interest! This project is designed to make contributing easy.

## Quick Start

```bash
git clone https://github.com/radityajayantara/notionctl.git
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

Check the [Notion API docs](https://developers.notion.com/reference/property-object) for the full list. Some good ones:

| Type | Difficulty | Notes |
|------|-----------|-------|
| `checkbox` | Easy | No config needed |
| `date` | Easy | No config needed |
| `url` | Easy | No config needed |
| `email` | Easy | No config needed |
| `phone_number` | Easy | No config needed |
| `created_time` | Easy | Read-only in Notion |
| `last_edited_time` | Easy | Read-only in Notion |
| `formula` | Medium | Needs `expression` config |
| `rollup` | Hard | Needs relation + property + function config |

## Add a YAML Template (No Go Required!)

Create a YAML file in `templates/` with a common use case:

```
templates/
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
