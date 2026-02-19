package jsonpath

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errExpectedOneArg = errors.New("expected 1 arg")

// testFunc is a minimal Function implementation for testing.
type testFunc struct {
	name       string
	resultType FuncType
	validateFn func([]ArgType) error
	callFn     func([]any) any
}

func (f *testFunc) Name() string              { return f.name }
func (f *testFunc) ResultType() FuncType       { return f.resultType }
func (f *testFunc) Validate(args []ArgType) error { return f.validateFn(args) }
func (f *testFunc) Call(args []any) any        { return f.callFn(args) }

func newTestFunc(name string, rt FuncType) *testFunc {
	return &testFunc{
		name:       name,
		resultType: rt,
		validateFn: func([]ArgType) error { return nil },
		callFn:     func([]any) any { return nil },
	}
}

func TestNewParser_NoOptions(t *testing.T) {
	p := NewParser()
	require.NotNil(t, p)
	assert.Empty(t, p.opts.functions)
}

func TestNewParser_WithFunctions(t *testing.T) {
	fn1 := newTestFunc("myfunc", FuncValue)
	fn2 := newTestFunc("other", FuncLogical)

	p := NewParser(WithFunctions(fn1, fn2))
	require.NotNil(t, p)
	assert.Len(t, p.opts.functions, 2)
	assert.Equal(t, fn1, p.opts.functions["myfunc"])
	assert.Equal(t, fn2, p.opts.functions["other"])
}

func TestWithFunctions_LastWins(t *testing.T) {
	fn1 := newTestFunc("dup", FuncValue)
	fn2 := newTestFunc("dup", FuncLogical)

	p := NewParser(WithFunctions(fn1, fn2))
	assert.Len(t, p.opts.functions, 1)
	assert.Equal(t, fn2, p.opts.functions["dup"])
}

func TestWithFunctions_MultipleOptions(t *testing.T) {
	fn1 := newTestFunc("a", FuncValue)
	fn2 := newTestFunc("b", FuncNodes)

	p := NewParser(WithFunctions(fn1), WithFunctions(fn2))
	assert.Len(t, p.opts.functions, 2)
	assert.Equal(t, fn1, p.opts.functions["a"])
	assert.Equal(t, fn2, p.opts.functions["b"])
}

func TestParserParse_ReturnsErrPathParse(t *testing.T) {
	p := NewParser()
	_, err := p.Parse("invalid")
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrPathParse))
}

func TestParserMustParse_Panics(t *testing.T) {
	p := NewParser()
	assert.Panics(t, func() {
		p.MustParse("invalid")
	})
}

func TestFuncType_Constants(t *testing.T) {
	assert.Equal(t, FuncType(0), FuncLogical)
	assert.Equal(t, FuncType(1), FuncValue)
	assert.Equal(t, FuncType(2), FuncNodes)
}

func TestArgType_Constants(t *testing.T) {
	assert.Equal(t, ArgType(0), ArgLiteral)
	assert.Equal(t, ArgType(1), ArgSingularQuery)
	assert.Equal(t, ArgType(2), ArgFilterQuery)
	assert.Equal(t, ArgType(3), ArgLogicalExpr)
	assert.Equal(t, ArgType(4), ArgFunctionExpr)
}

func TestFunction_Interface(t *testing.T) {
	fn := newTestFunc("length", FuncValue)
	fn.validateFn = func(args []ArgType) error {
		if len(args) != 1 {
			return fmt.Errorf("%w", errExpectedOneArg)
		}
		return nil
	}
	fn.callFn = func(args []any) any {
		return 42
	}

	assert.Equal(t, "length", fn.Name())
	assert.Equal(t, FuncValue, fn.ResultType())
	assert.NoError(t, fn.Validate([]ArgType{ArgLiteral}))
	assert.Error(t, fn.Validate([]ArgType{ArgLiteral, ArgLiteral}))
	assert.Equal(t, 42, fn.Call([]any{"hello"}))
}
