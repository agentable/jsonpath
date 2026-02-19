package ast

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChild(t *testing.T) {
	t.Parallel()

	t.Run("no_selectors", func(t *testing.T) {
		t.Parallel()
		s := Child()
		assert.Empty(t, s.Selectors())
		assert.False(t, s.IsDescendant())
	})

	t.Run("single_selector", func(t *testing.T) {
		t.Parallel()
		s := Child(NameSelector("foo"))
		assert.Len(t, s.Selectors(), 1)
		assert.Equal(t, Name, s.Selectors()[0].Kind)
		assert.False(t, s.IsDescendant())
	})

	t.Run("multiple_selectors", func(t *testing.T) {
		t.Parallel()
		s := Child(NameSelector("a"), IndexSelector(1), WildcardSelector())
		assert.Len(t, s.Selectors(), 3)
		assert.False(t, s.IsDescendant())
	})
}

func TestDescendant(t *testing.T) {
	t.Parallel()

	t.Run("no_selectors", func(t *testing.T) {
		t.Parallel()
		s := Descendant()
		assert.Empty(t, s.Selectors())
		assert.True(t, s.IsDescendant())
	})

	t.Run("single_selector", func(t *testing.T) {
		t.Parallel()
		s := Descendant(NameSelector("x"))
		assert.Len(t, s.Selectors(), 1)
		assert.True(t, s.IsDescendant())
	})
}

func TestSegmentIsSingular(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		seg      Segment
		singular bool
	}{
		{
			name:     "child_single_name",
			seg:      Child(NameSelector("a")),
			singular: true,
		},
		{
			name:     "child_single_index",
			seg:      Child(IndexSelector(0)),
			singular: true,
		},
		{
			name:     "child_wildcard",
			seg:      Child(WildcardSelector()),
			singular: false,
		},
		{
			name:     "child_slice",
			seg:      Child(SliceSelector(SliceArgs{HasStart: true, Start: 0})),
			singular: false,
		},
		{
			name:     "child_filter",
			seg:      Child(FilterSelector(&FilterExpr{})),
			singular: false,
		},
		{
			name:     "child_multiple_selectors",
			seg:      Child(NameSelector("a"), NameSelector("b")),
			singular: false,
		},
		{
			name:     "child_no_selectors",
			seg:      Child(),
			singular: false,
		},
		{
			name:     "descendant_single_name",
			seg:      Descendant(NameSelector("a")),
			singular: false,
		},
		{
			name:     "descendant_single_index",
			seg:      Descendant(IndexSelector(0)),
			singular: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.singular, tc.seg.IsSingular())
		})
	}
}

func TestSegmentString(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		seg  Segment
		want string
	}{
		{
			name: "child_single_name",
			seg:  Child(NameSelector("foo")),
			want: `["foo"]`,
		},
		{
			name: "child_single_index",
			seg:  Child(IndexSelector(3)),
			want: `[3]`,
		},
		{
			name: "child_wildcard",
			seg:  Child(WildcardSelector()),
			want: `[*]`,
		},
		{
			name: "child_multiple_selectors",
			seg:  Child(NameSelector("a"), NameSelector("b"), IndexSelector(0)),
			want: `["a","b",0]`,
		},
		{
			name: "child_slice",
			seg:  Child(SliceSelector(SliceArgs{Start: 1, End: 5, Step: 2, HasStart: true, HasEnd: true, HasStep: true})),
			want: `[1:5:2]`,
		},
		{
			name: "descendant_single_name",
			seg:  Descendant(NameSelector("x")),
			want: `..["x"]`,
		},
		{
			name: "descendant_wildcard",
			seg:  Descendant(WildcardSelector()),
			want: `..[*]`,
		},
		{
			name: "descendant_multiple",
			seg:  Descendant(NameSelector("a"), IndexSelector(1)),
			want: `..["a",1]`,
		},
		{
			name: "child_empty",
			seg:  Child(),
			want: `[]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.seg.String())
		})
	}
}

func TestSegmentWriteTo(t *testing.T) {
	t.Parallel()

	// Verify writeTo produces the same result as String.
	seg := Child(NameSelector("test"), IndexSelector(2))
	var buf strings.Builder
	seg.writeTo(&buf)
	assert.Equal(t, seg.String(), buf.String())
}
