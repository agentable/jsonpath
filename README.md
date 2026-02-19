# JSONPath

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.26-blue)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A high-performance RFC 9535 compliant JSONPath implementation for Go

## Features

- **RFC 9535 compliant**: Full compliance with the JSONPath standard, validated against the official test suite
- **Zero-allocation hot paths**: Hand-written lexer and optimized evaluator minimize GC pressure
- **Native Go 1.26 idioms**: Leverages generics, iterators (`iter.Seq`), and modern stdlib (`slices`, `maps`, `cmp`)
- **First-class json experiment support**: Seamless integration with `github.com/go-json-experiment/json` alongside standard `any`-based evaluation
- **Concurrent-safe**: Compiled `Path` objects are safe for concurrent use across goroutines
- **Clean minimal API**: Simple, focused interface consistent with `jsondiff` and `jsonmerge` siblings

## Installation

```bash
go get github.com/agentable/jsonpath
```

## Quick Start

```go
package main

import (
	"fmt"
	"github.com/agentable/jsonpath"
)

func main() {
	// Parse a JSONPath expression
	path := jsonpath.MustParse("$.store.book[*].author")

	// Query JSON data
	data := map[string]any{
		"store": map[string]any{
			"book": []any{
				map[string]any{"author": "Nigel Rees", "title": "Sayings of the Century"},
				map[string]any{"author": "Evelyn Waugh", "title": "Sword of Honour"},
			},
		},
	}

	// Select matching nodes
	results := path.Select(data)
	for _, author := range results {
		fmt.Println(author)
	}
	// Output:
	// Nigel Rees
	// Evelyn Waugh
}
```

## API Overview

### Parsing

```go
// Parse compiles a JSONPath expression
path, err := jsonpath.Parse("$.store.book[0].title")
if err != nil {
	// handle error
}

// MustParse panics on parse error
path := jsonpath.MustParse("$.store.book[0].title")
```

### Querying

```go
// Select returns all matching nodes
results := path.Select(data)

// SelectLocated returns nodes with their normalized paths
located := path.SelectLocated(data)
for _, node := range located {
	fmt.Printf("%s: %v\n", node.Path, node.Value)
}

// QueryJSON unmarshals and queries in one step
results, err := jsonpath.QueryJSON(jsonBytes, path)
```

### Iterators

```go
// NodeList supports range-over-func (Go 1.23+)
for node := range results.All() {
	fmt.Println(node)
}

// LocatedNodeList provides multiple iterators
for node := range located.All() {
	fmt.Printf("%s: %v\n", node.Path, node.Value)
}

for value := range located.Values() {
	fmt.Println(value)
}

for path := range located.Paths() {
	fmt.Println(path)
}
```

### Normalized Paths

```go
located := path.SelectLocated(data)
for _, node := range located {
	// Normalized path: $['store']['book'][0]
	fmt.Println(node.Path.String())

	// JSON Pointer (RFC 6901): /store/book/0
	fmt.Println(node.Path.Pointer())
}

// Deduplicate and sort results
located = located.Deduplicate()
located.Sort()
```

## Supported Selectors

| Selector | Example | Description |
|----------|---------|-------------|
| Root | `$` | Root node |
| Child name | `$.store` or `$['store']` | Object member |
| Child index | `$[0]` | Array element |
| Wildcard | `$[*]` or `$.*` | All children |
| Slice | `$[0:5:2]` | Array slice with start:end:step |
| Descendant | `$..book` | Recursive descent |
| Filter | `$[?@.price < 10]` | Filter expression |

## Filter Expressions

```go
// Comparison operators: ==, !=, <, <=, >, >=
path := jsonpath.MustParse("$.store.book[?@.price < 10]")

// Logical operators: &&, ||, !
path := jsonpath.MustParse("$.store.book[?@.price < 10 && @.category == 'fiction']")

// Existence test
path := jsonpath.MustParse("$.store.book[?@.isbn]")

// Built-in functions
path := jsonpath.MustParse("$.store.book[?length(@.title) > 20]")
```

## Built-in Functions

| Function | Signature | Description |
|----------|-----------|-------------|
| `length` | `length(value) → int` | String length or array/object size |
| `count` | `count(nodes) → int` | Number of nodes in a node list |
| `match` | `match(str, regex) → bool` | Full string match (I-Regexp) |
| `search` | `search(str, regex) → bool` | Substring match (I-Regexp) |
| `value` | `value(nodes) → any` | Extract single value from node list |

## Custom Functions

Extend JSONPath with custom filter functions:

```go
type MinFunc struct{}

func (MinFunc) Name() string { return "min" }
func (MinFunc) ResultType() jsonpath.FuncType { return jsonpath.FuncValue }

func (MinFunc) Validate(args []jsonpath.ArgType) error {
	if len(args) != 1 {
		return fmt.Errorf("min() requires exactly 1 argument")
	}
	return nil
}

func (MinFunc) Call(args []any) any {
	nodes, ok := args[0].([]any)
	if !ok || len(nodes) == 0 {
		return nil
	}
	// Find minimum numeric value
	minVal := nodes[0]
	for _, n := range nodes[1:] {
		if nf, ok := n.(float64); ok {
			if mf, ok := minVal.(float64); ok && nf < mf {
				minVal = n
			}
		}
	}
	return minVal
}

// Register and use
parser := jsonpath.NewParser(
	jsonpath.WithFunctions(MinFunc{}),
)
path := parser.MustParse("$.prices[?@ == min(@)]")
```

## Working with Go Structs

JSONPath operates on the JSON data model (`map[string]any`, `[]any`, primitives). To query Go structs, marshal them first:

```go
type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

user := User{Name: "Alice", Email: "alice@example.com"}

// Marshal to JSON-compatible format
data, _ := json.Marshal(user)
var v any
json.Unmarshal(data, &v)

// Now query with JSONPath
path := jsonpath.MustParse("$.name")
result := path.Select(v)  // []any{"Alice"}
```

For repeated queries, marshal once and reuse the result.

## Concurrent Usage

Compiled `Path` objects are safe for concurrent use:

```go
path := jsonpath.MustParse("$.store.book[*].author")

var wg sync.WaitGroup
for _, doc := range documents {
	wg.Add(1)
	go func(d any) {
		defer wg.Done()
		results := path.Select(d)
		// process results
	}(doc)
}
wg.Wait()
```

Compile once, query many times across goroutines.

## Performance

Optimized for high-throughput scenarios:

- Zero-copy lexer using byte offsets instead of string copies
- Flat selector arrays for cache-friendly memory layout
- Pre-allocated result slices with capacity hints
- Fast-path type assertions before reflection
- Compiled regex caching with `sync.Map`

Run benchmarks:

```bash
task bench
```

## Development

```bash
# Run tests
task test

# Run tests with coverage
task test-coverage

# Run linters
task lint

# Run benchmarks
task bench

# Format code
task fmt

# Full verification (deps, format, vet, lint, test)
task verify
```

## Contributing

Contributions are welcome! Please ensure:

- All tests pass (`task test`)
- Code is formatted (`task fmt`)
- Linters pass (`task lint`)
- New features include tests and documentation

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
