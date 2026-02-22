package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFuncTypeString(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		ft   FuncType
		want string
	}{
		{name: "logical", ft: Logical, want: "Logical"},
		{name: "value", ft: Value, want: "Value"},
		{name: "nodes", ft: Nodes, want: "Nodes"},
		{name: "unknown", ft: FuncType(99), want: "FuncType(99)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.ft.String())
		})
	}
}

func TestFuncTypeValues(t *testing.T) {
	t.Parallel()

	// Verify enum values are distinct and ordered as expected.
	assert.Equal(t, FuncType(0), Logical)
	assert.Equal(t, FuncType(1), Value)
	assert.Equal(t, FuncType(2), Nodes)
}

func TestArgTypeValues(t *testing.T) {
	t.Parallel()

	assert.Equal(t, ArgType(0), Literal)
	assert.Equal(t, ArgType(1), QueryArg)
	assert.Equal(t, ArgType(2), FilterArg)
	assert.Equal(t, ArgType(3), LogicalArg)
	assert.Equal(t, ArgType(4), FunctionArg)
}

func TestArgConvertsTo(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name   string
		arg    ArgType
		target FuncType
		want   bool
	}{
		// Literal converts only to Value.
		{name: "literal_to_value", arg: Literal, target: Value, want: true},
		{name: "literal_to_logical", arg: Literal, target: Logical, want: false},
		{name: "literal_to_nodes", arg: Literal, target: Nodes, want: false},

		// QueryArg converts to Value or Nodes.
		{name: "singular_to_value", arg: QueryArg, target: Value, want: true},
		{name: "singular_to_nodes", arg: QueryArg, target: Nodes, want: true},
		{name: "singular_to_logical", arg: QueryArg, target: Logical, want: false},

		// FilterArg converts only to Nodes.
		{name: "filter_to_nodes", arg: FilterArg, target: Nodes, want: true},
		{name: "filter_to_value", arg: FilterArg, target: Value, want: false},
		{name: "filter_to_logical", arg: FilterArg, target: Logical, want: false},

		// LogicalArg converts only to Logical.
		{name: "logical_to_logical", arg: LogicalArg, target: Logical, want: true},
		{name: "logical_to_value", arg: LogicalArg, target: Value, want: false},
		{name: "logical_to_nodes", arg: LogicalArg, target: Nodes, want: false},

		// FunctionArg accepts all (validated separately).
		{name: "func_to_value", arg: FunctionArg, target: Value, want: true},
		{name: "func_to_logical", arg: FunctionArg, target: Logical, want: true},
		{name: "func_to_nodes", arg: FunctionArg, target: Nodes, want: true},

		// Unknown ArgType.
		{name: "unknown_to_value", arg: ArgType(99), target: Value, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, ArgConvertsTo(tc.arg, tc.target))
		})
	}
}

// mockFunc implements Function for testing.
type mockFunc struct {
	name       string
	resultType FuncType
	validateFn func([]ArgType) error
	callFn     func([]any) any
}

func (f *mockFunc) Name() string                  { return f.name }
func (f *mockFunc) ResultType() FuncType          { return f.resultType }
func (f *mockFunc) Validate(args []ArgType) error { return f.validateFn(args) }
func (f *mockFunc) Call(args []any) any           { return f.callFn(args) }

func TestFuncExpr(t *testing.T) {
	t.Parallel()

	fn := &mockFunc{
		name:       "testfn",
		resultType: Value,
		validateFn: func(args []ArgType) error { return nil },
		callFn:     func(args []any) any { return 42 },
	}

	t.Run("constructor", func(t *testing.T) {
		t.Parallel()
		fe := NewFuncExpr(fn, []ArgType{Literal, Literal}, "arg1", 2)
		assert.Equal(t, "testfn", fe.Name())
		assert.Same(t, fn, fe.Func())
		require.Len(t, fe.Args(), 2)
		assert.Equal(t, "arg1", fe.Args()[0])
		assert.Equal(t, 2, fe.Args()[1])
	})

	t.Run("result_type", func(t *testing.T) {
		t.Parallel()
		fe := NewFuncExpr(fn, nil)
		assert.Equal(t, Value, fe.ResultType())
	})

	t.Run("string", func(t *testing.T) {
		t.Parallel()
		fe := NewFuncExpr(fn, []ArgType{Literal}, "a")
		assert.Equal(t, "testfn()", fe.String())
	})

	t.Run("no_args", func(t *testing.T) {
		t.Parallel()
		fe := NewFuncExpr(fn, nil)
		assert.Empty(t, fe.Args())
		assert.Equal(t, "testfn()", fe.String())
	})
}

func TestRegistry(t *testing.T) {
	t.Parallel()

	t.Run("new_registry_has_builtins", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		assert.Equal(t, 5, r.Len())

		for _, name := range []string{"length", "count", "match", "search", "value"} {
			fn, ok := r.Lookup(name)
			assert.True(t, ok, "built-in %q should exist", name)
			assert.Equal(t, name, fn.Name())
		}
	})

	t.Run("builtin_result_types", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()

		fn, _ := r.Lookup("length")
		assert.Equal(t, Value, fn.ResultType())

		fn, _ = r.Lookup("count")
		assert.Equal(t, Value, fn.ResultType())

		fn, _ = r.Lookup("match")
		assert.Equal(t, Logical, fn.ResultType())

		fn, _ = r.Lookup("search")
		assert.Equal(t, Logical, fn.ResultType())

		fn, _ = r.Lookup("value")
		assert.Equal(t, Value, fn.ResultType())
	})

	t.Run("lookup_missing", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		fn, ok := r.Lookup("nonexistent")
		assert.False(t, ok)
		assert.Nil(t, fn)
	})

	t.Run("register_custom", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		custom := &mockFunc{
			name:       "custom",
			resultType: Nodes,
			validateFn: func(args []ArgType) error { return nil },
			callFn:     func(args []any) any { return nil },
		}
		r.Register(custom)
		assert.Equal(t, 6, r.Len())

		fn, ok := r.Lookup("custom")
		assert.True(t, ok)
		assert.Same(t, custom, fn)
	})

	t.Run("register_overwrites", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		replacement := &mockFunc{
			name:       "length",
			resultType: Nodes,
			validateFn: func(args []ArgType) error { return nil },
			callFn:     func(args []any) any { return nil },
		}
		r.Register(replacement)
		assert.Equal(t, 5, r.Len()) // count unchanged

		fn, ok := r.Lookup("length")
		assert.True(t, ok)
		assert.Same(t, replacement, fn)
		assert.Equal(t, Nodes, fn.ResultType())
	})
}

func TestBuiltinValidation(t *testing.T) {
	t.Parallel()

	r := NewRegistry()

	t.Run("length_valid", func(t *testing.T) {
		t.Parallel()
		fn, _ := r.Lookup("length")
		assert.NoError(t, fn.Validate([]ArgType{Literal}))
	})

	t.Run("length_too_many_args", func(t *testing.T) {
		t.Parallel()
		fn, _ := r.Lookup("length")
		err := fn.Validate([]ArgType{Literal, Literal})
		assert.ErrorIs(t, err, ErrArgCount)
		assert.Contains(t, err.Error(), "expected 1, got 2")
	})

	t.Run("length_no_args", func(t *testing.T) {
		t.Parallel()
		fn, _ := r.Lookup("length")
		err := fn.Validate([]ArgType{})
		assert.ErrorIs(t, err, ErrArgCount)
		assert.Contains(t, err.Error(), "expected 1, got 0")
	})

	t.Run("match_valid", func(t *testing.T) {
		t.Parallel()
		fn, _ := r.Lookup("match")
		assert.NoError(t, fn.Validate([]ArgType{Literal, Literal}))
	})

	t.Run("match_wrong_count", func(t *testing.T) {
		t.Parallel()
		fn, _ := r.Lookup("match")
		err := fn.Validate([]ArgType{Literal})
		assert.ErrorIs(t, err, ErrArgCount)
		assert.Contains(t, err.Error(), "expected 2, got 1")
	})

	t.Run("search_valid", func(t *testing.T) {
		t.Parallel()
		fn, _ := r.Lookup("search")
		assert.NoError(t, fn.Validate([]ArgType{Literal, Literal}))
	})

	t.Run("count_valid", func(t *testing.T) {
		t.Parallel()
		fn, _ := r.Lookup("count")
		assert.NoError(t, fn.Validate([]ArgType{FilterArg}))
	})

	t.Run("value_valid", func(t *testing.T) {
		t.Parallel()
		fn, _ := r.Lookup("value")
		assert.NoError(t, fn.Validate([]ArgType{FilterArg}))
	})
}

func TestBuiltinCallReturnsNil(t *testing.T) {
	t.Parallel()

	// Built-in stubs return nil; actual evaluation is in the functions package.
	r := NewRegistry()
	for _, name := range []string{"length", "count", "match", "search", "value"} {
		fn, _ := r.Lookup(name)
		assert.Nil(t, fn.Call(nil))
	}
}
