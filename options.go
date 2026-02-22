package jsonpath

import (
	"fmt"
	"maps"

	"github.com/agentable/jsonpath/functions"
	"github.com/agentable/jsonpath/internal/ast"
	"github.com/agentable/jsonpath/internal/parser"
)

// FuncType describes the type of a function extension's return value as
// defined by RFC 9535 §2.4.1.
type FuncType uint8

const (
	// FuncLogical indicates the function returns a logical (bool) value.
	FuncLogical FuncType = iota
	// FuncValue indicates the function returns a single JSON value.
	FuncValue
	// FuncNodes indicates the function returns a node list.
	FuncNodes
)

// ArgType describes the type of a function argument expression for
// parse-time validation.
type ArgType uint8

const (
	// ArgLiteral is a literal JSON value argument.
	ArgLiteral ArgType = iota
	// ArgSingularQuery is a singular query argument (e.g. @.name or $.name).
	ArgSingularQuery
	// ArgFilterQuery is a filter query argument producing a node list.
	ArgFilterQuery
	// ArgLogicalExpr is a logical expression argument.
	ArgLogicalExpr
	// ArgFunctionExpr is a nested function call argument.
	ArgFunctionExpr
)

// Function defines an extension function that can be registered with a
// [Parser] via [WithFunctions]. Implementations must be safe for concurrent
// use if the [Parser] is used concurrently.
type Function interface {
	// Name returns the function name as used in JSONPath expressions.
	Name() string
	// ResultType returns the FuncType of the function's return value.
	ResultType() FuncType
	// Validate checks argument types at parse time. It returns an error
	// if the argument types are incompatible with this function.
	Validate(args []ArgType) error
	// Call evaluates the function at query time and returns the result.
	Call(args []any) any
}

// Option configures a [Parser].
type Option func(*parserOptions)

// parserOptions holds configuration for a [Parser].
type parserOptions struct {
	functions map[string]Function
}

// WithFunctions registers additional filter functions beyond the RFC 9535
// built-ins. If multiple functions share the same name, the last one wins.
func WithFunctions(fns ...Function) Option {
	return func(o *parserOptions) {
		for _, fn := range fns {
			o.functions[fn.Name()] = fn
		}
	}
}

// Parser parses JSONPath expressions into [Path] values, optionally
// configured with extension functions.
type Parser struct {
	opts parserOptions
}

// NewParser creates a new [Parser] configured by opts.
func NewParser(opts ...Option) *Parser {
	p := &Parser{
		opts: parserOptions{
			functions: make(map[string]Function),
		},
	}
	for _, o := range opts {
		o(&p.opts)
	}
	return p
}

// Parse compiles a JSONPath expression. Returns [ErrPathParse] on failure.
func (p *Parser) Parse(expr string) (*Path, error) {
	// Convert function map to map[string]any for internal parser
	// Start with built-in functions
	funcs := make(map[string]any, 5+len(p.opts.functions))

	// Register built-in functions from the functions package
	registry := newBuiltinRegistry()
	maps.Copy(funcs, registry)

	// Add user-provided functions (can override built-ins)
	for name, fn := range p.opts.functions {
		funcs[name] = fn
	}

	internalParser, err := parser.New(expr, funcs)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrPathParse, err)
	}

	query, err := internalParser.Parse()
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrPathParse, err)
	}

	return &Path{query: query}, nil
}

// newBuiltinRegistry creates a registry with RFC 9535 built-in functions.
func newBuiltinRegistry() map[string]any {
	builtins := []ast.Function{
		&functions.LengthFunc{},
		&functions.CountFunc{},
		&functions.MatchFunc{},
		&functions.SearchFunc{},
		&functions.ValueFunc{},
	}

	registry := make(map[string]any, len(builtins))
	for _, fn := range builtins {
		registry[fn.Name()] = fn
	}
	return registry
}

// MustParse compiles a JSONPath expression. Panics on failure.
func (p *Parser) MustParse(expr string) *Path {
	path, err := p.Parse(expr)
	if err != nil {
		panic(err)
	}
	return path
}
