package functions

import (
	"regexp"
	"sync"
	"testing"

	"github.com/agentable/jsonpath/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltins(t *testing.T) {
	t.Parallel()
	fns := Builtins()
	require.Len(t, fns, 5)

	names := make([]string, len(fns))
	for i, fn := range fns {
		names[i] = fn.Name()
	}
	assert.Equal(t, []string{"length", "count", "match", "search", "value"}, names)
}

func TestRegisterBuiltins(t *testing.T) {
	t.Parallel()
	r := ast.NewRegistry()

	// Stubs exist; register real implementations.
	RegisterBuiltins(r)
	assert.Equal(t, 5, r.Len())

	// Verify all are accessible and have correct result types.
	fn, ok := r.Lookup("length")
	require.True(t, ok)
	assert.Equal(t, ast.Value, fn.ResultType())

	fn, ok = r.Lookup("count")
	require.True(t, ok)
	assert.Equal(t, ast.Value, fn.ResultType())

	fn, ok = r.Lookup("match")
	require.True(t, ok)
	assert.Equal(t, ast.Logical, fn.ResultType())

	fn, ok = r.Lookup("search")
	require.True(t, ok)
	assert.Equal(t, ast.Logical, fn.ResultType())

	fn, ok = r.Lookup("value")
	require.True(t, ok)
	assert.Equal(t, ast.Value, fn.ResultType())
}

func TestLengthFunc(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		args []any
		want any
	}{
		{name: "empty_string", args: []any{""}, want: 0},
		{name: "ascii_string", args: []any{"abc def"}, want: 7},
		{name: "unicode_string", args: []any{"foö"}, want: 3},
		{name: "emoji_string", args: []any{"Hi 👋🏻"}, want: 5},
		{name: "empty_array", args: []any{[]any{}}, want: 0},
		{name: "array", args: []any{[]any{1, 2, 3, 4, 5}}, want: 5},
		{name: "nested_array", args: []any{[]any{1, 2, []any{3, 4}}}, want: 3},
		{name: "empty_object", args: []any{map[string]any{}}, want: 0},
		{name: "object", args: []any{map[string]any{"x": 1, "y": 2, "z": 3}}, want: 3},
		{name: "integer", args: []any{42}, want: nil},
		{name: "float", args: []any{3.14}, want: nil},
		{name: "bool", args: []any{true}, want: nil},
		{name: "nil_arg", args: []any{nil}, want: nil},
		{name: "no_args", args: []any{}, want: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, LengthFunc{}.Call(tc.args))
		})
	}
}

func TestLengthFuncValidate(t *testing.T) {
	t.Parallel()

	fn := LengthFunc{}

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, fn.Validate([]ast.ArgType{ast.Literal}))
		assert.NoError(t, fn.Validate([]ast.ArgType{ast.QueryArg}))
	})

	t.Run("wrong_count", func(t *testing.T) {
		t.Parallel()
		err := fn.Validate([]ast.ArgType{})
		assert.ErrorIs(t, err, ast.ErrArgCount)

		err = fn.Validate([]ast.ArgType{ast.Literal, ast.Literal})
		assert.ErrorIs(t, err, ast.ErrArgCount)
	})

	t.Run("wrong_type", func(t *testing.T) {
		t.Parallel()
		err := fn.Validate([]ast.ArgType{ast.FilterArg})
		assert.ErrorIs(t, err, ErrArgType)

		err = fn.Validate([]ast.ArgType{ast.LogicalArg})
		assert.ErrorIs(t, err, ErrArgType)
	})
}

func TestCountFunc(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		args []any
		want any
	}{
		{name: "empty_nodes", args: []any{[]any{}}, want: 0},
		{name: "one_node", args: []any{[]any{1}}, want: 1},
		{name: "three_nodes", args: []any{[]any{1, true, nil}}, want: 3},
		{name: "nil_arg", args: []any{nil}, want: 0},
		{name: "not_slice", args: []any{"hello"}, want: 0},
		{name: "no_args", args: []any{}, want: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, CountFunc{}.Call(tc.args))
		})
	}
}

func TestCountFuncValidate(t *testing.T) {
	t.Parallel()

	fn := CountFunc{}

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, fn.Validate([]ast.ArgType{ast.FilterArg}))
		assert.NoError(t, fn.Validate([]ast.ArgType{ast.QueryArg}))
	})

	t.Run("wrong_count", func(t *testing.T) {
		t.Parallel()
		err := fn.Validate([]ast.ArgType{})
		assert.ErrorIs(t, err, ast.ErrArgCount)
	})

	t.Run("wrong_type", func(t *testing.T) {
		t.Parallel()
		err := fn.Validate([]ast.ArgType{ast.Literal})
		assert.ErrorIs(t, err, ErrArgType)
	})
}

func TestMatchFunc(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		args []any
		want any
	}{
		{name: "full_match", args: []any{"foo", "foo"}, want: true},
		{name: "dot_star", args: []any{"foo", ".*"}, want: true},
		{name: "dot_single", args: []any{"x", "."}, want: true},
		{name: "dot_two_chars", args: []any{"xx", "."}, want: false},
		{name: "no_match", args: []any{"foo", "bar"}, want: false},
		{name: "partial_not_full", args: []any{"foobar", "foo"}, want: false},
		{name: "multiline_newline", args: []any{"xx\nyz", ".*"}, want: false},
		{name: "multiline_crlf", args: []any{"xx\r\nyz", ".*"}, want: false},
		{name: "not_string_input", args: []any{42, "."}, want: false},
		{name: "not_string_pattern", args: []any{"x", 42}, want: false},
		{name: "invalid_regex", args: []any{"x", ".["}, want: false},
		{name: "no_args", args: []any{}, want: false},
		{name: "one_arg", args: []any{"foo"}, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, MatchFunc{}.Call(tc.args))
		})
	}
}

func TestMatchFuncValidate(t *testing.T) {
	t.Parallel()

	fn := MatchFunc{}

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, fn.Validate([]ast.ArgType{ast.Literal, ast.Literal}))
		assert.NoError(t, fn.Validate([]ast.ArgType{ast.QueryArg, ast.Literal}))
	})

	t.Run("wrong_count", func(t *testing.T) {
		t.Parallel()
		err := fn.Validate([]ast.ArgType{ast.Literal})
		assert.ErrorIs(t, err, ast.ErrArgCount)

		err = fn.Validate([]ast.ArgType{ast.Literal, ast.Literal, ast.Literal})
		assert.ErrorIs(t, err, ast.ErrArgCount)
	})

	t.Run("wrong_type", func(t *testing.T) {
		t.Parallel()
		err := fn.Validate([]ast.ArgType{ast.LogicalArg, ast.Literal})
		assert.ErrorIs(t, err, ErrArgType)
		assert.Contains(t, err.Error(), "argument 1")

		err = fn.Validate([]ast.ArgType{ast.Literal, ast.LogicalArg})
		assert.ErrorIs(t, err, ErrArgType)
		assert.Contains(t, err.Error(), "argument 2")
	})
}

func TestSearchFunc(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		args []any
		want any
	}{
		{name: "found", args: []any{"foobar", "bar"}, want: true},
		{name: "dot", args: []any{"x", "."}, want: true},
		{name: "dot_in_longer", args: []any{"xx", "."}, want: true},
		{name: "no_match", args: []any{"foo", "baz"}, want: false},
		{name: "multiline_partial", args: []any{"xx\nyz", "xx"}, want: true},
		{name: "multiline_dot_star", args: []any{"xx\nyz", ".*"}, want: true},
		{name: "not_string_input", args: []any{42, "."}, want: false},
		{name: "not_string_pattern", args: []any{"x", 42}, want: false},
		{name: "invalid_regex", args: []any{"x", ".["}, want: false},
		{name: "no_args", args: []any{}, want: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, SearchFunc{}.Call(tc.args))
		})
	}
}

func TestSearchFuncValidate(t *testing.T) {
	t.Parallel()

	fn := SearchFunc{}

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, fn.Validate([]ast.ArgType{ast.Literal, ast.Literal}))
	})

	t.Run("wrong_count", func(t *testing.T) {
		t.Parallel()
		err := fn.Validate([]ast.ArgType{ast.Literal})
		assert.ErrorIs(t, err, ast.ErrArgCount)
	})

	t.Run("wrong_type", func(t *testing.T) {
		t.Parallel()
		err := fn.Validate([]ast.ArgType{ast.LogicalArg, ast.Literal})
		assert.ErrorIs(t, err, ErrArgType)
	})
}

func TestValueFunc(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		args []any
		want any
	}{
		{name: "single_node", args: []any{[]any{42}}, want: 42},
		{name: "single_string", args: []any{[]any{"hello"}}, want: "hello"},
		{name: "single_nil", args: []any{[]any{nil}}, want: nil},
		{name: "empty_nodes", args: []any{[]any{}}, want: nil},
		{name: "multiple_nodes", args: []any{[]any{1, 2, 3}}, want: nil},
		{name: "nil_arg", args: []any{nil}, want: nil},
		{name: "not_slice", args: []any{"hello"}, want: nil},
		{name: "no_args", args: []any{}, want: nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, ValueFunc{}.Call(tc.args))
		})
	}
}

func TestValueFuncValidate(t *testing.T) {
	t.Parallel()

	fn := ValueFunc{}

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		assert.NoError(t, fn.Validate([]ast.ArgType{ast.FilterArg}))
		assert.NoError(t, fn.Validate([]ast.ArgType{ast.QueryArg}))
	})

	t.Run("wrong_count", func(t *testing.T) {
		t.Parallel()
		err := fn.Validate([]ast.ArgType{})
		assert.ErrorIs(t, err, ast.ErrArgCount)
	})

	t.Run("wrong_type", func(t *testing.T) {
		t.Parallel()
		err := fn.Validate([]ast.ArgType{ast.Literal})
		assert.ErrorIs(t, err, ErrArgType)
	})
}

func TestCompileIRegexpCache(t *testing.T) {
	t.Parallel()

	// Clear cache for this test.
	clearRegexCache()

	re1 := compileIRegexp("abc")
	require.NotNil(t, re1)

	// Second call should return cached value.
	re2 := compileIRegexp("abc")
	require.NotNil(t, re2)
	assert.Equal(t, re1, re2)

	// Invalid pattern returns nil and is not cached.
	reInvalid := compileIRegexp(".[")
	assert.Nil(t, reInvalid)
	_, loaded := reCache.Load(".[")
	assert.False(t, loaded)
}

func TestReplaceDotIRegexp(t *testing.T) {
	t.Parallel()

	// "." should NOT match \n or \r per RFC 9485.
	re := compileIRegexp(`\A.\z`)
	require.NotNil(t, re)
	assert.True(t, re.MatchString("x"))
	assert.False(t, re.MatchString("\n"))
	assert.False(t, re.MatchString("\r"))
}

func TestCompileIRegexpConcurrent(t *testing.T) {
	t.Parallel()

	clearRegexCache()

	var wg sync.WaitGroup
	patterns := []string{"a+", "b+", "c+", "d+", "e+"}
	for _, p := range patterns {
		wg.Add(1)
		go func(pattern string) {
			defer wg.Done()
			re := compileIRegexp(pattern)
			assert.NotNil(t, re)
			assert.IsType(t, &regexp.Regexp{}, re)
		}(p)
	}
	wg.Wait()
}

func TestRegexCacheBehavior(t *testing.T) {
	t.Parallel()

	t.Run("cache_stores_compiled_regex", func(t *testing.T) {
		clearRegexCache()
		pattern := "test.*pattern"

		// First compilation
		re1 := compileIRegexp(pattern)
		require.NotNil(t, re1)

		// Verify it's in cache
		cached, ok := reCache.Load(pattern)
		require.True(t, ok)
		assert.Same(t, re1, cached)
	})

	t.Run("cache_returns_same_instance", func(t *testing.T) {
		clearRegexCache()
		pattern := "same.*instance"

		re1 := compileIRegexp(pattern)
		re2 := compileIRegexp(pattern)
		re3 := compileIRegexp(pattern)

		require.NotNil(t, re1)
		require.NotNil(t, re2)
		require.NotNil(t, re3)

		// All should be the exact same pointer
		assert.Same(t, re1, re2)
		assert.Same(t, re1, re3)
	})

	t.Run("cache_handles_different_patterns", func(t *testing.T) {
		clearRegexCache()

		re1 := compileIRegexp("pattern1")
		re2 := compileIRegexp("pattern2")
		re3 := compileIRegexp("pattern3")

		require.NotNil(t, re1)
		require.NotNil(t, re2)
		require.NotNil(t, re3)

		// All should be different instances
		assert.NotSame(t, re1, re2)
		assert.NotSame(t, re1, re3)
		assert.NotSame(t, re2, re3)
	})

	t.Run("invalid_pattern_not_cached", func(t *testing.T) {
		clearRegexCache()
		invalidPattern := "["

		re := compileIRegexp(invalidPattern)
		assert.Nil(t, re)

		// Should not be in cache
		_, ok := reCache.Load(invalidPattern)
		assert.False(t, ok)

		// Second call should also return nil
		re2 := compileIRegexp(invalidPattern)
		assert.Nil(t, re2)
	})

	t.Run("match_function_uses_cache", func(t *testing.T) {
		clearRegexCache()
		fn := MatchFunc{}

		pattern := "hello"
		anchoredPattern := `\A` + pattern + `\z`

		// First call compiles and caches
		result1 := fn.Call([]any{"hello", pattern})
		assert.Equal(t, true, result1)

		// Verify anchored pattern is cached
		cached, ok := reCache.Load(anchoredPattern)
		require.True(t, ok)
		require.NotNil(t, cached)

		// Second call uses cache
		result2 := fn.Call([]any{"hello", pattern})
		assert.Equal(t, true, result2)

		// Third call with different input but same pattern
		result3 := fn.Call([]any{"world", pattern})
		assert.Equal(t, false, result3)
	})

	t.Run("search_function_uses_cache", func(t *testing.T) {
		clearRegexCache()
		fn := SearchFunc{}

		pattern := "world"

		// First call compiles and caches
		result1 := fn.Call([]any{"hello world", pattern})
		assert.Equal(t, true, result1)

		// Verify pattern is cached
		cached, ok := reCache.Load(pattern)
		require.True(t, ok)
		require.NotNil(t, cached)

		// Second call uses cache
		result2 := fn.Call([]any{"world hello", pattern})
		assert.Equal(t, true, result2)
	})

	t.Run("concurrent_cache_access_same_pattern", func(t *testing.T) {
		clearRegexCache()
		pattern := "concurrent.*test"

		var wg sync.WaitGroup
		results := make([]*regexp.Regexp, 20)

		// Launch 20 goroutines compiling the same pattern
		for i := range 20 {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				results[idx] = compileIRegexp(pattern)
			}(i)
		}
		wg.Wait()

		// All should be non-nil
		for i, re := range results {
			require.NotNil(t, re, "result %d should not be nil", i)
		}

		// Due to race conditions, multiple goroutines may compile before caching,
		// but all results should be functionally equivalent
		for _, re := range results {
			assert.True(t, re.MatchString("concurrent test"))
			assert.True(t, re.MatchString("concurrent xyz test"))
		}

		// Verify pattern is in cache
		cached, ok := reCache.Load(pattern)
		require.True(t, ok)
		require.NotNil(t, cached)
	})

	t.Run("concurrent_cache_access_different_patterns", func(t *testing.T) {
		clearRegexCache()
		patterns := []string{
			"pattern1", "pattern2", "pattern3", "pattern4", "pattern5",
			"pattern6", "pattern7", "pattern8", "pattern9", "pattern10",
		}

		var wg sync.WaitGroup
		results := make(map[string]*regexp.Regexp)
		var mu sync.Mutex

		for _, p := range patterns {
			wg.Add(1)
			go func(pattern string) {
				defer wg.Done()
				re := compileIRegexp(pattern)
				mu.Lock()
				results[pattern] = re
				mu.Unlock()
			}(p)
		}
		wg.Wait()

		// All patterns should be compiled
		assert.Equal(t, len(patterns), len(results))

		// All should be non-nil
		for pattern, re := range results {
			require.NotNil(t, re, "pattern %s should compile", pattern)
		}

		// Verify all are in cache
		for _, pattern := range patterns {
			cached, ok := reCache.Load(pattern)
			require.True(t, ok, "pattern %s should be cached", pattern)
			require.NotNil(t, cached)
		}
	})
}

func TestIRegexpCompliance(t *testing.T) {
	t.Parallel()

	t.Run("dot_does_not_match_newline", func(t *testing.T) {
		re := compileIRegexp("a.b")
		require.NotNil(t, re)

		assert.True(t, re.MatchString("axb"))
		assert.True(t, re.MatchString("a b"))
		assert.True(t, re.MatchString("a\tb"))
		assert.False(t, re.MatchString("a\nb"))
		assert.False(t, re.MatchString("a\rb"))
	})

	t.Run("dot_star_does_not_match_across_newlines", func(t *testing.T) {
		re := compileIRegexp("a.*b")
		require.NotNil(t, re)

		assert.True(t, re.MatchString("ab"))
		assert.True(t, re.MatchString("axyzb"))
		assert.False(t, re.MatchString("a\nb"))
		assert.False(t, re.MatchString("a\r\nb"))
		assert.False(t, re.MatchString("axy\nzb"))
	})

	t.Run("multiple_dots", func(t *testing.T) {
		re := compileIRegexp("^...$")
		require.NotNil(t, re)

		assert.True(t, re.MatchString("abc"))
		assert.True(t, re.MatchString("xyz"))
		assert.False(t, re.MatchString("ab\n"))
		assert.False(t, re.MatchString("a\nbc"))
		assert.False(t, re.MatchString("\nabc"))
	})

	t.Run("case_insensitive_flag", func(t *testing.T) {
		re := compileIRegexp("(?i)hello")
		require.NotNil(t, re)

		assert.True(t, re.MatchString("hello"))
		assert.True(t, re.MatchString("HELLO"))
		assert.True(t, re.MatchString("Hello"))
		assert.True(t, re.MatchString("HeLLo"))
	})

	t.Run("character_classes_unaffected", func(t *testing.T) {
		re := compileIRegexp("[abc]")
		require.NotNil(t, re)

		assert.True(t, re.MatchString("a"))
		assert.True(t, re.MatchString("b"))
		assert.True(t, re.MatchString("c"))
		assert.False(t, re.MatchString("d"))
	})

	t.Run("negated_character_classes", func(t *testing.T) {
		re := compileIRegexp("[^abc]")
		require.NotNil(t, re)

		assert.False(t, re.MatchString("a"))
		assert.False(t, re.MatchString("b"))
		assert.True(t, re.MatchString("d"))
		assert.True(t, re.MatchString("x"))
	})

	t.Run("anchors", func(t *testing.T) {
		re := compileIRegexp("^hello$")
		require.NotNil(t, re)

		assert.True(t, re.MatchString("hello"))
		assert.False(t, re.MatchString("hello world"))
		assert.False(t, re.MatchString("say hello"))
	})

	t.Run("quantifiers", func(t *testing.T) {
		tests := []struct {
			pattern string
			input   string
			want    bool
		}{
			{"^a+$", "a", true},
			{"^a+$", "aaa", true},
			{"^a+$", "", false},
			{"^a*$", "", true},
			{"^a*$", "aaa", true},
			{"^a?$", "", true},
			{"^a?$", "a", true},
			{"^a?$", "aa", false},
			{"^a{2}$", "aa", true},
			{"^a{2}$", "a", false},
			{"^a{2,4}$", "aa", true},
			{"^a{2,4}$", "aaa", true},
			{"^a{2,4}$", "aaaa", true},
			{"^a{2,4}$", "a", false},
			{"^a{2,4}$", "aaaaa", false},
		}

		for _, tt := range tests {
			re := compileIRegexp(tt.pattern)
			require.NotNil(t, re, "pattern %s should compile", tt.pattern)
			got := re.MatchString(tt.input)
			assert.Equal(t, tt.want, got, "pattern %s with input %q", tt.pattern, tt.input)
		}
	})

	t.Run("alternation", func(t *testing.T) {
		re := compileIRegexp("cat|dog")
		require.NotNil(t, re)

		assert.True(t, re.MatchString("cat"))
		assert.True(t, re.MatchString("dog"))
		assert.False(t, re.MatchString("bird"))
	})

	t.Run("unicode_support", func(t *testing.T) {
		re := compileIRegexp("世界")
		require.NotNil(t, re)

		assert.True(t, re.MatchString("世界"))
		assert.False(t, re.MatchString("世"))
		assert.False(t, re.MatchString("界"))
	})

	t.Run("unicode_with_dot", func(t *testing.T) {
		re := compileIRegexp("世.界")
		require.NotNil(t, re)

		assert.True(t, re.MatchString("世x界"))
		assert.True(t, re.MatchString("世 界"))
		assert.False(t, re.MatchString("世\n界"))
	})
}

func TestMatchFuncAnchoring(t *testing.T) {
	t.Parallel()

	fn := MatchFunc{}

	t.Run("full_match_required", func(t *testing.T) {
		// match() implicitly anchors with \A and \z
		assert.Equal(t, true, fn.Call([]any{"hello", "hello"}))
		assert.Equal(t, false, fn.Call([]any{"hello world", "hello"}))
		assert.Equal(t, false, fn.Call([]any{"say hello", "hello"}))
		assert.Equal(t, false, fn.Call([]any{"say hello world", "hello"}))
	})

	t.Run("pattern_must_match_entire_string", func(t *testing.T) {
		assert.Equal(t, true, fn.Call([]any{"abc", "abc"}))
		assert.Equal(t, true, fn.Call([]any{"abc", "a.c"}))
		assert.Equal(t, true, fn.Call([]any{"abc", ".*"}))
		assert.Equal(t, false, fn.Call([]any{"abcd", "abc"}))
		assert.Equal(t, false, fn.Call([]any{"xabc", "abc"}))
	})
}

func TestSearchFuncNoAnchoring(t *testing.T) {
	t.Parallel()

	fn := SearchFunc{}

	t.Run("substring_match_allowed", func(t *testing.T) {
		// search() does not anchor
		assert.Equal(t, true, fn.Call([]any{"hello", "hello"}))
		assert.Equal(t, true, fn.Call([]any{"hello world", "hello"}))
		assert.Equal(t, true, fn.Call([]any{"say hello", "hello"}))
		assert.Equal(t, true, fn.Call([]any{"say hello world", "hello"}))
	})

	t.Run("pattern_can_match_anywhere", func(t *testing.T) {
		assert.Equal(t, true, fn.Call([]any{"abc", "abc"}))
		assert.Equal(t, true, fn.Call([]any{"abc", "a"}))
		assert.Equal(t, true, fn.Call([]any{"abc", "b"}))
		assert.Equal(t, true, fn.Call([]any{"abc", "c"}))
		assert.Equal(t, true, fn.Call([]any{"xabcy", "abc"}))
	})
}

func BenchmarkLengthFunc(b *testing.B) {
	fn := LengthFunc{}

	b.Run("string", func(b *testing.B) {
		args := []any{"hello world"}
		for b.Loop() {
			fn.Call(args)
		}
	})

	b.Run("array", func(b *testing.B) {
		args := []any{[]any{1, 2, 3, 4, 5}}
		for b.Loop() {
			fn.Call(args)
		}
	})

	b.Run("object", func(b *testing.B) {
		args := []any{map[string]any{"a": 1, "b": 2, "c": 3}}
		for b.Loop() {
			fn.Call(args)
		}
	})
}

func BenchmarkCountFunc(b *testing.B) {
	fn := CountFunc{}
	args := []any{[]any{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}}

	for b.Loop() {
		fn.Call(args)
	}
}

func BenchmarkMatchFunc(b *testing.B) {
	fn := MatchFunc{}

	b.Run("simple_match", func(b *testing.B) {
		args := []any{"hello", "hello"}
		for b.Loop() {
			fn.Call(args)
		}
	})

	b.Run("pattern_match", func(b *testing.B) {
		args := []any{"hello world", "hello.*"}
		for b.Loop() {
			fn.Call(args)
		}
	})

	b.Run("no_match", func(b *testing.B) {
		args := []any{"hello", "world"}
		for b.Loop() {
			fn.Call(args)
		}
	})
}

func BenchmarkSearchFunc(b *testing.B) {
	fn := SearchFunc{}

	b.Run("found", func(b *testing.B) {
		args := []any{"hello world", "world"}
		for b.Loop() {
			fn.Call(args)
		}
	})

	b.Run("not_found", func(b *testing.B) {
		args := []any{"hello world", "xyz"}
		for b.Loop() {
			fn.Call(args)
		}
	})
}

func BenchmarkValueFunc(b *testing.B) {
	fn := ValueFunc{}

	b.Run("single_node", func(b *testing.B) {
		args := []any{[]any{42}}
		for b.Loop() {
			fn.Call(args)
		}
	})

	b.Run("empty_nodes", func(b *testing.B) {
		args := []any{[]any{}}
		for b.Loop() {
			fn.Call(args)
		}
	})

	b.Run("multiple_nodes", func(b *testing.B) {
		args := []any{[]any{1, 2, 3}}
		for b.Loop() {
			fn.Call(args)
		}
	})
}

func BenchmarkRegexCache(b *testing.B) {
	b.Run("cache_hit", func(b *testing.B) {
		clearRegexCache()
		pattern := "test.*pattern"

		// Prime the cache
		compileIRegexp(pattern)

		b.ResetTimer()
		for b.Loop() {
			compileIRegexp(pattern)
		}
	})

	b.Run("cache_miss", func(b *testing.B) {
		for b.Loop() {
			clearRegexCache()
			compileIRegexp("test.*pattern")
		}
	})

	b.Run("concurrent_cache_hit", func(b *testing.B) {
		clearRegexCache()
		pattern := "concurrent.*test"

		// Prime the cache
		compileIRegexp(pattern)

		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				compileIRegexp(pattern)
			}
		})
	})
}
