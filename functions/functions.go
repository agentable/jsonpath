// Package functions provides the RFC 9535 §2.4 built-in function
// implementations for JSONPath filter expressions.
package functions

import (
	"errors"
	"fmt"
	"regexp"
	"regexp/syntax"
	"sync"
	"unicode/utf8"

	"github.com/agentable/jsonpath/internal/ast"
)

// reCache caches compiled regular expressions keyed by pattern string.
var reCache sync.Map

// clearRegexCache clears the regex cache. Only used for testing.
func clearRegexCache() {
	reCache.Range(func(key, value any) bool {
		reCache.Delete(key)
		return true
	})
}

// ErrArgType indicates a function argument has an incompatible type.
var ErrArgType = errors.New("incompatible argument type")

// Builtins returns the five RFC 9535 §2.4 built-in function implementations.
func Builtins() []ast.Function {
	return []ast.Function{
		&LengthFunc{},
		&CountFunc{},
		&MatchFunc{},
		&SearchFunc{},
		&ValueFunc{},
	}
}

// RegisterBuiltins registers the RFC 9535 built-in functions into r,
// replacing any existing stub registrations.
func RegisterBuiltins(r *ast.Registry) {
	for _, fn := range Builtins() {
		r.Register(fn)
	}
}

// LengthFunc implements the RFC 9535 §2.4.4 length() function.
//
// Parameters: 1 ValueType
// Result: ValueType (int for string/array/object, nil otherwise)
type LengthFunc struct{}

func (LengthFunc) Name() string             { return "length" }
func (LengthFunc) ResultType() ast.FuncType { return ast.Value }

func (LengthFunc) Validate(args []ast.ArgType) error {
	if len(args) != 1 {
		return fmt.Errorf("expected 1, got %d: %w", len(args), ast.ErrArgCount)
	}
	if !ast.ArgConvertsTo(args[0], ast.Value) {
		return fmt.Errorf("cannot convert argument to ValueType: %w", ErrArgType)
	}
	return nil
}

// Call returns the length of the argument:
//   - string: number of Unicode scalar values
//   - []any: number of elements
//   - map[string]any: number of members
//   - nil or other: nil
func (LengthFunc) Call(args []any) any {
	if len(args) == 0 || args[0] == nil {
		return nil
	}
	switch v := args[0].(type) {
	case string:
		return utf8.RuneCountInString(v)
	case []any:
		return len(v)
	case map[string]any:
		return len(v)
	default:
		return nil
	}
}

// CountFunc implements the RFC 9535 §2.4.6 count() function.
//
// Parameters: 1 NodesType
// Result: ValueType (int)
type CountFunc struct{}

func (CountFunc) Name() string             { return "count" }
func (CountFunc) ResultType() ast.FuncType { return ast.Value }

func (CountFunc) Validate(args []ast.ArgType) error {
	if len(args) != 1 {
		return fmt.Errorf("expected 1, got %d: %w", len(args), ast.ErrArgCount)
	}
	if !ast.ArgConvertsTo(args[0], ast.Nodes) {
		return fmt.Errorf("cannot convert argument to NodesType: %w", ErrArgType)
	}
	return nil
}

// Call returns the number of nodes in the node list argument.
func (CountFunc) Call(args []any) any {
	if len(args) == 0 {
		return 0
	}
	if nodes, ok := args[0].([]any); ok {
		return len(nodes)
	}
	return 0
}

// MatchFunc implements the RFC 9535 §2.4.7 match() function.
//
// match() tests whether the string argument fully matches the regex pattern
// (implicitly anchored with \A and \z).
//
// Parameters: 2 ValueType (string, regex pattern)
// Result: LogicalType (bool)
type MatchFunc struct{}

func (MatchFunc) Name() string             { return "match" }
func (MatchFunc) ResultType() ast.FuncType { return ast.Logical }

func (MatchFunc) Validate(args []ast.ArgType) error {
	return validateTwoValueArgs(args)
}

// Call returns true if the string argument fully matches the regex pattern.
// Returns false if either argument is not a string or the regex is invalid.
func (MatchFunc) Call(args []any) any {
	if len(args) < 2 {
		return false
	}
	str, ok1 := args[0].(string)
	pattern, ok2 := args[1].(string)
	if !ok1 || !ok2 {
		return false
	}
	re := compileIRegexp(`\A` + pattern + `\z`)
	if re == nil {
		return false
	}
	return re.MatchString(str)
}

// SearchFunc implements the RFC 9535 §2.4.7 search() function.
//
// search() tests whether the string argument contains a substring matching
// the regex pattern (not anchored).
//
// Parameters: 2 ValueType (string, regex pattern)
// Result: LogicalType (bool)
type SearchFunc struct{}

func (SearchFunc) Name() string             { return "search" }
func (SearchFunc) ResultType() ast.FuncType { return ast.Logical }

func (SearchFunc) Validate(args []ast.ArgType) error {
	return validateTwoValueArgs(args)
}

// Call returns true if the string argument contains a match for the regex pattern.
// Returns false if either argument is not a string or the regex is invalid.
func (SearchFunc) Call(args []any) any {
	if len(args) < 2 {
		return false
	}
	str, ok1 := args[0].(string)
	pattern, ok2 := args[1].(string)
	if !ok1 || !ok2 {
		return false
	}
	re := compileIRegexp(pattern)
	if re == nil {
		return false
	}
	return re.MatchString(str)
}

// ValueFunc implements the RFC 9535 §2.4.8 value() function.
//
// If the node list contains exactly one node, value() returns that node's value.
// Otherwise it returns nil (Nothing).
//
// Parameters: 1 NodesType
// Result: ValueType
type ValueFunc struct{}

func (ValueFunc) Name() string             { return "value" }
func (ValueFunc) ResultType() ast.FuncType { return ast.Value }

func (ValueFunc) Validate(args []ast.ArgType) error {
	if len(args) != 1 {
		return fmt.Errorf("expected 1, got %d: %w", len(args), ast.ErrArgCount)
	}
	if !ast.ArgConvertsTo(args[0], ast.Nodes) {
		return fmt.Errorf("cannot convert argument to NodesType: %w", ErrArgType)
	}
	return nil
}

// Call returns the value of the single node in the node list, or nil if
// the list is empty or contains more than one node.
func (ValueFunc) Call(args []any) any {
	if len(args) == 0 || args[0] == nil {
		return nil
	}
	nodes, ok := args[0].([]any)
	if !ok || len(nodes) != 1 {
		return nil
	}
	return nodes[0]
}

// compileIRegexp compiles an I-Regexp pattern (RFC 9485) into a Go *regexp.Regexp.
// It replaces "." (OpAnyChar) with "[^\n\r]" per RFC 9485 §5.
// Results are cached via sync.Map for concurrent safety.
// Returns nil if the pattern is invalid.
func compileIRegexp(pattern string) *regexp.Regexp {
	if v, ok := reCache.Load(pattern); ok {
		return v.(*regexp.Regexp)
	}
	re, err := compileIRegexpUncached(pattern)
	if err != nil {
		return nil
	}
	reCache.Store(pattern, re)
	return re
}

// crlf is the pre-compiled replacement for "." in I-Regexp patterns.
var crlf = mustParseSyntax(`[^\n\r]`, syntax.Perl)

// mustParseSyntax parses a constant regex pattern or panics.
func mustParseSyntax(pattern string, flags syntax.Flags) *syntax.Regexp {
	re, err := syntax.Parse(pattern, flags)
	if err != nil {
		panic("functions: bad constant pattern: " + err.Error())
	}
	return re
}

// compileIRegexpUncached compiles an I-Regexp pattern without caching.
func compileIRegexpUncached(pattern string) (*regexp.Regexp, error) {
	parsed, err := syntax.Parse(pattern, syntax.Perl|syntax.DotNL)
	if err != nil {
		return nil, err
	}
	replaceDot(parsed)
	return regexp.Compile(parsed.String())
}

// validateTwoValueArgs validates that exactly 2 ValueType arguments are provided.
func validateTwoValueArgs(args []ast.ArgType) error {
	if len(args) != 2 {
		return fmt.Errorf("expected 2, got %d: %w", len(args), ast.ErrArgCount)
	}
	for i, arg := range args {
		if !ast.ArgConvertsTo(arg, ast.Value) {
			return fmt.Errorf("cannot convert argument %d to ValueType: %w", i+1, ErrArgType)
		}
	}
	return nil
}

// replaceDot recursively replaces all OpAnyChar nodes with [^\n\r] nodes
// to comply with RFC 9485 I-Regexp semantics.
func replaceDot(re *syntax.Regexp) {
	if re.Op == syntax.OpAnyChar {
		*re = *crlf
		return
	}
	for _, sub := range re.Sub {
		replaceDot(sub)
	}
}
