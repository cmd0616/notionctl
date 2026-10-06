## What
Brief description of the change.

## Why
What problem does this solve? Link to issue if applicable.

Closes #

## How
Key implementation details or approach taken.

## Checklist
- [ ] `go vet ./...` passes
- [ ] `go test ./...` passes (use `CGO_ENABLED=0` on macOS if needed)
- [ ] New tests added for new functionality
- [ ] README updated (if adding commands/features)
- [ ] Templates pass validation (`go test ./internal/config/... -run TestTemplatesValid`)
