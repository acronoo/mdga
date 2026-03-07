# AGENTS.md - Guidelines for AI Agents

## Project Overview

**mdga** (MAKE DEVELOP GREAT AGAIN) is a Go CLI tool for local Docker Compose development.
It provides a TUI (Text User Interface) using Charm's Bubble Tea framework to selectively
exclude services from Docker Compose and run them locally instead.

## Build Commands

```bash
# Build the binary
go build -o mdga

# Build with version info
go build -ldflags="-s -w" -o mdga

# Run the application
go run main.go

# Clean build artifacts
rm -f mdga tmp.compose.json tmp.env
```

## Lint/Format Commands

```bash
# Format code (REQUIRED before committing)
go fmt ./...

# Vet code for issues
go vet ./...

# Run linter (if golangci-lint is installed)
golangci-lint run
```

## Test Commands

```bash
# Run all tests (none currently exist)
go test ./...

# Run specific test (when tests are added)
go test -run TestFunctionName ./...

# Run with coverage
go test -cover ./...
```

## Code Style Guidelines

### Formatting
- **Always run `go fmt ./...`** before committing
- Use tabs for indentation (Go standard)
- Line length: no strict limit, but keep readable
- Group imports: standard library first, then third-party packages

### Naming Conventions
- **Types**: PascalCase (e.g., `BuildMethod`, `ExecResult`)
- **Functions**: camelCase, exported if needed (e.g., `getWorkDir`, `ModifyCompose`)
- **Variables**: camelCase (e.g., `servicesLocal`, `hostsLine`)
- **Constants**: Use `iota` pattern for enums (see `step`, `buildMethod`)
- **Package-level vars**: snake_case for style definitions (e.g., `titleStyle`)
- **Receivers**: Single letter matching type (e.g., `m` for `model`)

### Error Handling
- Always wrap errors with `fmt.Errorf` and `%w` verb:
  ```go
  return fmt.Errorf("failed to get working directory: %w", err)
  ```
- Check errors immediately after assignment
- Return early on errors when possible

### Functions and Methods
- Keep functions focused and single-purpose
- Methods on struct should use pointer receivers for state modification
- Methods on struct should use value receivers for read-only operations
- Document exported functions with comments starting with the function name

### Comments
- Use `//` for single-line comments
- Use `/* */` for multi-line comments sparingly
- Comments should explain "why" not "what" (code shows what)

### Types and Structs
- Use typed constants with `iota` for state enums
- Struct tags not needed unless serializing
- Keep struct fields exported only if necessary

### Imports
Group order (separated by blank line):
1. Standard library
2. Third-party packages

Example:
```go
import (
    "encoding/json"
    "fmt"
    "os"

    "github.com/charmbracelet/bubbles/spinner"
    tea "github.com/charmbracelet/bubbletea"
)
```

### Project Structure
- Single-file application: `main.go`
- Configuration: `go.mod`, `go.sum`
- Docker: `compose.yml`, `.env`
- Binary: `mdga` (gitignored, but exists for distribution)

### Dependencies
Key libraries:
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/bubbles` - UI components
- `github.com/charmbracelet/lipgloss` - Styling

### Bubble Tea Patterns
- Model struct holds all application state
- Use typed messages for commands (e.g., `type execResult struct`)
- Commands return `tea.Cmd` for async operations
- Update method handles all state transitions
- View method renders the UI using `strings.Builder`

### Special Notes
- UI strings may contain Russian text (this is intentional)
- Application modifies `tmp.compose.json` and `tmp.env` files
- Requires Docker Compose and Git in PATH
- Application runs in alternate screen mode (`tea.WithAltScreen()`)

## Pre-Commit Checklist

1. Run `go fmt ./...`
2. Run `go vet ./...`
3. Build with `go build -o mdga`
4. Test binary works: `./mdga`

## Git Guidelines

- Do NOT commit the `mdga` binary or temporary files (`tmp.compose.json`, `tmp.env`)
- Use descriptive commit messages
- The binary is gitignored but kept for distribution
- Temporary files are created at runtime and should not be tracked

## No Cursor/Copilot Rules Found

No `.cursorrules`, `.cursor/rules/`, or `.github/copilot-instructions.md` files exist.
