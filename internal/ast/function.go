package ast

import (
	"errors"
	"fmt"
	"strings"
)

// FuncType describes the return type of a function expression per RFC 9535 §2.4.1.
type FuncType uint8

const (
	// Logical indicates the function returns a logical (bool) value.
	Logical FuncType = iota
	// Value indicates the function returns a single JSON value.
	Value
	// Nodes indicates the function returns a node list.
	Nodes
)

// String returns the string representation of ft.
func (ft FuncType) String() string {
	switch ft {
	case Logical:
		return "Logical"
	case Value:
		return "Value"
	case Nodes:
		return "Nodes"
	default:
		return fmt.Sprintf("FuncType(%d)", ft)
	}
}

// ArgType describes the type of a function argument expression for
// parse-time validation per RFC 9535 §2.4.
type ArgType uint8

const (
	// Literal is a literal JSON value argument.
	Literal ArgType = iota
	// QueryArg is a singular query argument (e.g. @.name or $.name).
	QueryArg
	// FilterArg is a filter query argument producing a node list.
	FilterArg
	// LogicalArg is a logical expression argument.
	LogicalArg
	// FunctionArg is a nested function call argument.
	FunctionArg
)

// ArgConvertsTo reports whether an argument of type arg can be used where a
// parameter of type target is expected per RFC 9535 §2.4.1 type conversion rules.
func ArgConvertsTo(arg ArgType, target FuncType) bool {
	switch arg {
	case Literal:
		return target == Value
	case QueryArg:
		return target == Value || target == Nodes
	case FilterArg:
		return target == Nodes
	case LogicalArg:
		return target == Logical
	case FunctionArg:
		// Requires deeper validation using the function's ResultType;
		// accepted here and validated separately by the parser.
		return true
	default:
		return false
	}
}

// Function defines a function that can be called in filter expressions.
// Implementations must be safe for concurrent use.
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

// FuncExpr represents a function call in a filter expression per RFC 9535 §2.4.
type FuncExpr struct {
	name     string    // function name
	fn       Function  // resolved function definition
	args     []any     // argument expressions; typed when filter support is complete
	argTypes []ArgType // argument types determined at parse time
}

// NewFuncExpr creates a [FuncExpr] for the given function and arguments.
func NewFuncExpr(fn Function, argTypes []ArgType, args ...any) *FuncExpr {
	return &FuncExpr{name: fn.Name(), fn: fn, args: args, argTypes: argTypes}
}

// Name returns the function name.
func (fe *FuncExpr) Name() string { return fe.name }

// Func returns the resolved [Function].
func (fe *FuncExpr) Func() Function { return fe.fn }

// Args returns the argument expressions.
func (fe *FuncExpr) Args() []any { return fe.args }

// ResultType returns the return type of the underlying function.
func (fe *FuncExpr) ResultType() FuncType { return fe.fn.ResultType() }

// Call evaluates the function with the given current and root nodes.
// It evaluates argument expressions and passes the results to the underlying function.
func (fe *FuncExpr) Call(current, root any) any {
	// Evaluate argument expressions
	evalArgs := make([]any, len(fe.args))
	for i, arg := range fe.args {
		switch a := arg.(type) {
		case *PathQuery:
			nodes := a.Select(current, root)
			switch {
			case i < len(fe.argTypes) && fe.argTypes[i] == FilterArg:
				// Function parameter expects NodesType, pass the node list
				evalArgs[i] = nodes
			case a.IsSingular():
				// For singular queries used as ValueType, extract the single value
				if len(nodes) == 1 {
					evalArgs[i] = nodes[0]
				} else {
					// Singular query returned no nodes - this is "nothing"
					evalArgs[i] = nil
				}
			default:
				evalArgs[i] = nodes
			}
		case *FuncExpr:
			evalArgs[i] = a.Call(current, root)
		case CompValue:
			evalArgs[i] = a.Value(current, root)
		default:
			evalArgs[i] = arg
		}
	}
	return fe.fn.Call(evalArgs)
}

// Eval implements BasicExpr for logical functions.
// Returns false if the function is not a logical function.
func (fe *FuncExpr) Eval(current, root any) bool {
	if fe.fn.ResultType() != Logical {
		return false
	}
	result := fe.Call(current, root)
	if b, ok := result.(bool); ok {
		return b
	}
	return false
}

// writeTo writes the canonical string representation of fe to buf.
func (fe *FuncExpr) writeTo(buf *strings.Builder) {
	buf.WriteString(fe.name)
	buf.WriteByte('(')
	buf.WriteByte(')')
}

// String returns the canonical string representation of fe.
func (fe *FuncExpr) String() string {
	var buf strings.Builder
	fe.writeTo(&buf)
	return buf.String()
}

// Registry holds named [Function] definitions for use during parsing and
// evaluation. A Registry is safe for concurrent reads after construction.
type Registry struct {
	funcs map[string]Function
}

// NewRegistry creates a [Registry] pre-populated with the RFC 9535 §2.4
// built-in function signatures.
func NewRegistry() *Registry {
	r := &Registry{funcs: make(map[string]Function, 8)}
	r.registerBuiltins()
	return r
}

// Register adds fn to the registry. If a function with the same name
// already exists, it is replaced.
func (r *Registry) Register(fn Function) {
	r.funcs[fn.Name()] = fn
}

// Lookup returns the [Function] with the given name and true, or nil and
// false if not found.
func (r *Registry) Lookup(name string) (Function, bool) {
	fn, ok := r.funcs[name]
	return fn, ok
}

// Len returns the number of registered functions.
func (r *Registry) Len() int { return len(r.funcs) }

// builtinFunc is a [Function] implementation for RFC 9535 built-in function
// signatures. It provides parse-time validation; actual evaluation logic is
// provided by the functions package.
type builtinFunc struct {
	name       string
	resultType FuncType
	validate   func([]ArgType) error
}

func (f *builtinFunc) Name() string                 { return f.name }
func (f *builtinFunc) ResultType() FuncType          { return f.resultType }
func (f *builtinFunc) Validate(args []ArgType) error { return f.validate(args) }

// Call is a placeholder that returns nil. Actual built-in evaluation logic
// is provided by the functions package.
func (f *builtinFunc) Call([]any) any { return nil }

// registerBuiltins registers the five RFC 9535 §2.4 built-in function
// signatures: length, count, match, search, value.
func (r *Registry) registerBuiltins() {
	r.Register(&builtinFunc{name: "length", resultType: Value, validate: validateNArgs(1)})
	r.Register(&builtinFunc{name: "count", resultType: Value, validate: validateNArgs(1)})
	r.Register(&builtinFunc{name: "match", resultType: Logical, validate: validateNArgs(2)})
	r.Register(&builtinFunc{name: "search", resultType: Logical, validate: validateNArgs(2)})
	r.Register(&builtinFunc{name: "value", resultType: Value, validate: validateNArgs(1)})
}

// ErrArgCount indicates a function received the wrong number of arguments.
var ErrArgCount = errors.New("wrong number of arguments")

// validateNArgs returns a validation function that checks for exactly n arguments.
func validateNArgs(n int) func([]ArgType) error {
	return func(args []ArgType) error {
		if len(args) != n {
			return fmt.Errorf("%w: expected %d, got %d", ErrArgCount, n, len(args))
		}
		return nil
	}
}
