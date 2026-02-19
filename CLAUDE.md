# JSONPath Library

RFC 9535 compliant JSONPath implementation for Go with zero-allocation hot paths and native Go 1.26 idioms.

## Project Overview

**Module**: `github.com/agentable/jsonpath`
**Go Version**: 1.26+
**License**: MIT

High-performance JSONPath query engine validated against the official RFC 9535 compliance test suite. Operates on standard Go `any` values (`map[string]any`, `[]any`, primitives) with first-class support for `github.com/go-json-experiment/json`.

## Commands

```bash
# Development
task test              # Run tests with race detector
task test-coverage     # Generate coverage report
task bench             # Run benchmarks
task lint              # Run golangci-lint
task fmt               # Format code
task vet               # Run go vet
task verify            # Full verification (deps, fmt, vet, lint, test)

# Utilities
task deps              # Download and tidy dependencies
task clean             # Clean build artifacts and caches
```

## Architecture

```
jsonpath/
├── jsonpath.go          # Public API: Parse, Select, QueryJSON
├── types.go             # NodeList, LocatedNodeList, NormalizedPath
├── options.go           # Parser, WithFunctions
├── internal/lexer/      # Zero-copy lexer (token offsets, no string copies)
├── internal/parser/     # Recursive descent parser
├── internal/ast/        # PathQuery, Segment, Selector (tagged union)
├── functions/           # RFC 9535 built-ins (length, count, match, search, value)
└── compliance/          # RFC 9535 CTS validation
```

### Key Types

- `Path`: Compiled JSONPath query, safe for concurrent use
- `NodeList`: Query results with `iter.Seq[any]` iterator
- `LocatedNodeList`: Results with normalized paths (RFC 9535 §2.7) and JSON Pointers (RFC 6901)
- `NormalizedPath`: Sequence of `NameElement` (string) or `IndexElement` (int)
- `Parser`: Configurable parser with `WithFunctions` for custom filter functions

## Coding Rules

### Performance Requirements

- **Zero-allocation hot paths**: Lexer uses byte offsets, not string copies. Pre-allocate slices with capacity hints.
- **Flat data structures**: Use tagged unions (struct with `kind` field) instead of interfaces for cache efficiency.
- **Early returns**: Check `if len(nodes) == 0 { return nodes }` before allocating.
- **Regex caching**: Cache compiled regexes in `sync.Map` keyed by pattern string.

### Go 1.26 Idioms

Use modern Go features consistently:

- `for i := range n` for integer iteration (Go 1.22+)
- `clear(map)` for efficient map clearing (Go 1.21+)
- `slices.SortFunc`, `slices.Clone`, `slices.Clip` for slice operations
- `for b.Loop()` in benchmarks (Go 1.24+), not `for i := 0; i < b.N; i++`
- `errors.Join()` for combining errors (Go 1.20+)
- `iter.Seq[T]` for iterators (Go 1.23+)

### Error Handling

- Use sentinel errors: `ErrPathParse`, `ErrFunction`, `ErrUnmarshal`
- Wrap errors with `%w` at end: `fmt.Errorf("context: %w", err)`
- Combine nil checks: `if len(args) == 0 || args[0] == nil { return nil }`

### Naming

- No `new()` for composites: use `&strings.Builder{}` or `var buf strings.Builder`
- Consistent receiver names: `l` for Lexer, `p` for Parser, `n` for NameElement
- No redundant naming: avoid repeating package/type in names

### Code Simplification

- Keep implementations minimal and focused on actual requirements
- Iterate directly over map/array values instead of building intermediate slices
- Remove comments that restate what the code does

## Testing

- Use `testify/require` for assertions
- Table-driven tests with subtests: `t.Run(tt.name, func(t *testing.T) { ... })`
- Use `b.Loop()` in benchmarks (Go 1.24+), not `for i := 0; i < b.N; i++`
- Compliance tests validate against RFC 9535 CTS embedded via `//go:embed`
- Run tests with race detector: `task test` or `go test -race ./...`

## Dependencies

Runtime dependencies:
- `github.com/go-json-experiment/json` - JSON unmarshaling for `QueryJSON` helpers

Test dependencies:
- `github.com/stretchr/testify` - Test assertions

## Agent Skills

Available skills in `.claude/skills/`:

- **agent-md-creating**: Generate CLAUDE.md for Go projects
- **code-simplifying**: Refine recently written Go code for clarity
- **committing**: Create conventional commits for Go packages
- **dependency-selecting**: Select Go dependencies from agentable ecosystem
- **github-actions**: Configure GitHub Actions CI/CD for Go packages
- **go-best-practices**: Google Go coding best practices and style guide
- **golang-taskfile**: Create and manage Taskfiles for Go projects
- **linting**: Set up and run golangci-lint v2 for Go projects
- **modernizing**: Go 1.20-1.26 modernization guide
- **ralphy-initializing**: Initialize Ralphy AI coding loop configuration
- **ralphy-todo-creating**: Create Ralphy TODO.yaml task files
- **readme-creating**: Generate README.md for Go libraries
- **reference-submodule-curation**: Find and vendor GitHub references as submodules
- **releasing**: Guide release process for Go packages
- **research-contract-planning**: Generate contract-only PLAN.md and TODO.yaml
- **testing**: Write Go tests following best practices
