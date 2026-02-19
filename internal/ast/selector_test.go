package ast

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSelectorConstructors(t *testing.T) {
	t.Parallel()

	t.Run("name_selector", func(t *testing.T) {
		t.Parallel()
		s := NameSelector("foo")
		assert.Equal(t, Name, s.Kind)
		assert.Equal(t, "foo", s.Name)
	})

	t.Run("index_selector", func(t *testing.T) {
		t.Parallel()
		s := IndexSelector(42)
		assert.Equal(t, Index, s.Kind)
		assert.Equal(t, int64(42), s.Index)
	})

	t.Run("index_selector_negative", func(t *testing.T) {
		t.Parallel()
		s := IndexSelector(-1)
		assert.Equal(t, Index, s.Kind)
		assert.Equal(t, int64(-1), s.Index)
	})

	t.Run("slice_selector", func(t *testing.T) {
		t.Parallel()
		args := SliceArgs{Start: 1, End: 5, Step: 2, HasStart: true, HasEnd: true, HasStep: true}
		s := SliceSelector(args)
		assert.Equal(t, Slice, s.Kind)
		assert.Equal(t, args, s.Slice)
	})

	t.Run("wildcard_selector", func(t *testing.T) {
		t.Parallel()
		s := WildcardSelector()
		assert.Equal(t, Wildcard, s.Kind)
	})

	t.Run("filter_selector", func(t *testing.T) {
		t.Parallel()
		expr := &FilterExpr{}
		s := FilterSelector(expr)
		assert.Equal(t, Filter, s.Kind)
		assert.Same(t, expr, s.Filter)
	})
}

func TestSelectorIsSingular(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		sel      Selector
		singular bool
	}{
		{
			name:     "name_is_singular",
			sel:      NameSelector("x"),
			singular: true,
		},
		{
			name:     "index_is_singular",
			sel:      IndexSelector(0),
			singular: true,
		},
		{
			name:     "slice_not_singular",
			sel:      SliceSelector(SliceArgs{HasStart: true, Start: 0}),
			singular: false,
		},
		{
			name:     "wildcard_not_singular",
			sel:      WildcardSelector(),
			singular: false,
		},
		{
			name:     "filter_not_singular",
			sel:      FilterSelector(&FilterExpr{}),
			singular: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.singular, tc.sel.IsSingular())
		})
	}
}

func TestSelectorString(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		sel  Selector
		want string
	}{
		{
			name: "name_simple",
			sel:  NameSelector("foo"),
			want: `"foo"`,
		},
		{
			name: "name_with_space",
			sel:  NameSelector("hello world"),
			want: `"hello world"`,
		},
		{
			name: "name_with_quote",
			sel:  NameSelector(`say "hi"`),
			want: `"say \"hi\""`,
		},
		{
			name: "name_unicode",
			sel:  NameSelector("日本語"),
			want: `"日本語"`,
		},
		{
			name: "name_empty",
			sel:  NameSelector(""),
			want: `""`,
		},
		{
			name: "index_zero",
			sel:  IndexSelector(0),
			want: "0",
		},
		{
			name: "index_positive",
			sel:  IndexSelector(42),
			want: "42",
		},
		{
			name: "index_negative",
			sel:  IndexSelector(-1),
			want: "-1",
		},
		{
			name: "index_large",
			sel:  IndexSelector(9007199254740992),
			want: "9007199254740992",
		},
		{
			name: "wildcard",
			sel:  WildcardSelector(),
			want: "*",
		},
		{
			name: "filter",
			sel:  FilterSelector(&FilterExpr{}),
			want: "?",
		},
		{
			name: "slice_full",
			sel:  SliceSelector(SliceArgs{Start: 1, End: 5, Step: 2, HasStart: true, HasEnd: true, HasStep: true}),
			want: "1:5:2",
		},
		{
			name: "slice_start_only",
			sel:  SliceSelector(SliceArgs{Start: 3, HasStart: true}),
			want: "3:",
		},
		{
			name: "slice_end_only",
			sel:  SliceSelector(SliceArgs{End: 5, HasEnd: true}),
			want: ":5",
		},
		{
			name: "slice_step_only",
			sel:  SliceSelector(SliceArgs{Step: 2, HasStep: true}),
			want: "::2",
		},
		{
			name: "slice_start_end",
			sel:  SliceSelector(SliceArgs{Start: 1, End: 3, HasStart: true, HasEnd: true}),
			want: "1:3",
		},
		{
			name: "slice_start_step",
			sel:  SliceSelector(SliceArgs{Start: 1, Step: 2, HasStart: true, HasStep: true}),
			want: "1::2",
		},
		{
			name: "slice_end_step",
			sel:  SliceSelector(SliceArgs{End: 5, Step: 2, HasEnd: true, HasStep: true}),
			want: ":5:2",
		},
		{
			name: "slice_no_args",
			sel:  SliceSelector(SliceArgs{}),
			want: ":",
		},
		{
			name: "slice_negative_step",
			sel:  SliceSelector(SliceArgs{Start: 5, End: 1, Step: -1, HasStart: true, HasEnd: true, HasStep: true}),
			want: "5:1:-1",
		},
		{
			name: "slice_negative_start",
			sel:  SliceSelector(SliceArgs{Start: -3, HasStart: true}),
			want: "-3:",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.sel.String())
		})
	}
}

func TestSelectorWriteTo(t *testing.T) {
	t.Parallel()

	// Verify writeTo produces the same result as String for each kind.
	selectors := []Selector{
		NameSelector("test"),
		IndexSelector(7),
		SliceSelector(SliceArgs{Start: 1, End: 5, HasStart: true, HasEnd: true}),
		WildcardSelector(),
		FilterSelector(&FilterExpr{}),
	}
	for _, sel := range selectors {
		var buf strings.Builder
		sel.writeTo(&buf)
		assert.Equal(t, sel.String(), buf.String())
	}
}

func TestSliceArgsWriteTo(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		args SliceArgs
		want string
	}{
		{
			name: "all_set",
			args: SliceArgs{Start: 0, End: 10, Step: 2, HasStart: true, HasEnd: true, HasStep: true},
			want: "0:10:2",
		},
		{
			name: "none_set",
			args: SliceArgs{},
			want: ":",
		},
		{
			name: "only_start",
			args: SliceArgs{Start: 5, HasStart: true},
			want: "5:",
		},
		{
			name: "only_end",
			args: SliceArgs{End: 5, HasEnd: true},
			want: ":5",
		},
		{
			name: "only_step",
			args: SliceArgs{Step: 3, HasStep: true},
			want: "::3",
		},
		{
			name: "start_and_end",
			args: SliceArgs{Start: 1, End: 4, HasStart: true, HasEnd: true},
			want: "1:4",
		},
		{
			name: "negative_values",
			args: SliceArgs{Start: -5, End: -1, Step: -1, HasStart: true, HasEnd: true, HasStep: true},
			want: "-5:-1:-1",
		},
		{
			name: "zero_start_set",
			args: SliceArgs{Start: 0, End: 3, HasStart: true, HasEnd: true},
			want: "0:3",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf strings.Builder
			tc.args.writeTo(&buf)
			assert.Equal(t, tc.want, buf.String())
		})
	}
}

func TestSelectorKindValues(t *testing.T) {
	t.Parallel()

	// Verify enum values are distinct and ordered as expected.
	assert.Equal(t, SelectorKind(0), Name)
	assert.Equal(t, SelectorKind(1), Index)
	assert.Equal(t, SelectorKind(2), Slice)
	assert.Equal(t, SelectorKind(3), Wildcard)
	assert.Equal(t, SelectorKind(4), Filter)
}
