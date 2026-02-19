package ast

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPathQuery(t *testing.T) {
	t.Parallel()

	t.Run("root_no_segments", func(t *testing.T) {
		t.Parallel()
		q := NewPathQuery(true)
		assert.True(t, q.IsRoot())
		assert.Empty(t, q.Segments())
	})

	t.Run("relative_no_segments", func(t *testing.T) {
		t.Parallel()
		q := NewPathQuery(false)
		assert.False(t, q.IsRoot())
		assert.Empty(t, q.Segments())
	})

	t.Run("root_with_segments", func(t *testing.T) {
		t.Parallel()
		segs := []Segment{Child(NameSelector("x")), Child(IndexSelector(0))}
		q := NewPathQuery(true, segs...)
		assert.True(t, q.IsRoot())
		require.Len(t, q.Segments(), 2)
		assert.Equal(t, Name, q.Segments()[0].Selectors()[0].Kind)
		assert.Equal(t, Index, q.Segments()[1].Selectors()[0].Kind)
	})
}

func TestPathQueryString(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		q    *PathQuery
		want string
	}{
		{
			name: "root_empty",
			q:    NewPathQuery(true),
			want: "$",
		},
		{
			name: "relative_empty",
			q:    NewPathQuery(false),
			want: "@",
		},
		{
			name: "root_single_name",
			q:    NewPathQuery(true, Child(NameSelector("foo"))),
			want: `$["foo"]`,
		},
		{
			name: "relative_single_name",
			q:    NewPathQuery(false, Child(NameSelector("bar"))),
			want: `@["bar"]`,
		},
		{
			name: "root_name_then_index",
			q:    NewPathQuery(true, Child(NameSelector("a")), Child(IndexSelector(0))),
			want: `$["a"][0]`,
		},
		{
			name: "descendant_name",
			q:    NewPathQuery(true, Descendant(NameSelector("x"))),
			want: `$..["x"]`,
		},
		{
			name: "wildcard",
			q:    NewPathQuery(true, Child(WildcardSelector())),
			want: `$[*]`,
		},
		{
			name: "multiple_selectors",
			q:    NewPathQuery(true, Child(NameSelector("a"), NameSelector("b"))),
			want: `$["a","b"]`,
		},
		{
			name: "slice_full",
			q: NewPathQuery(true, Child(SliceSelector(SliceArgs{
				Start: 1, End: 5, Step: 2,
				HasStart: true, HasEnd: true, HasStep: true,
			}))),
			want: `$[1:5:2]`,
		},
		{
			name: "slice_no_start",
			q: NewPathQuery(true, Child(SliceSelector(SliceArgs{
				End: 3, HasEnd: true,
			}))),
			want: `$[:3]`,
		},
		{
			name: "mixed_segments",
			q: NewPathQuery(true,
				Child(NameSelector("store")),
				Descendant(WildcardSelector()),
				Child(IndexSelector(0)),
			),
			want: `$["store"]..[*][0]`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.q.String())
		})
	}
}

func TestPathQueryIsSingular(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name     string
		q        *PathQuery
		singular bool
	}{
		{
			name:     "empty_query",
			q:        NewPathQuery(true),
			singular: true,
		},
		{
			name:     "single_name",
			q:        NewPathQuery(true, Child(NameSelector("x"))),
			singular: true,
		},
		{
			name:     "single_index",
			q:        NewPathQuery(true, Child(IndexSelector(0))),
			singular: true,
		},
		{
			name:     "name_then_index",
			q:        NewPathQuery(true, Child(NameSelector("a")), Child(IndexSelector(0))),
			singular: true,
		},
		{
			name:     "descendant_not_singular",
			q:        NewPathQuery(true, Descendant(NameSelector("x"))),
			singular: false,
		},
		{
			name:     "wildcard_not_singular",
			q:        NewPathQuery(true, Child(WildcardSelector())),
			singular: false,
		},
		{
			name:     "slice_not_singular",
			q:        NewPathQuery(true, Child(SliceSelector(SliceArgs{HasStart: true, Start: 0}))),
			singular: false,
		},
		{
			name:     "filter_not_singular",
			q:        NewPathQuery(true, Child(FilterSelector(&FilterExpr{}))),
			singular: false,
		},
		{
			name:     "multiple_selectors_not_singular",
			q:        NewPathQuery(true, Child(NameSelector("a"), NameSelector("b"))),
			singular: false,
		},
		{
			name: "singular_then_non_singular",
			q: NewPathQuery(true,
				Child(NameSelector("a")),
				Child(WildcardSelector()),
			),
			singular: false,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.singular, tc.q.IsSingular())
		})
	}
}

func TestPathQuerySingular(t *testing.T) {
	t.Parallel()

	t.Run("returns_nil_for_non_singular", func(t *testing.T) {
		t.Parallel()
		q := NewPathQuery(true, Child(WildcardSelector()))
		assert.Nil(t, q.Singular())
	})

	t.Run("returns_singular_for_root_name", func(t *testing.T) {
		t.Parallel()
		q := NewPathQuery(true, Child(NameSelector("x")))
		sq := q.Singular()
		require.NotNil(t, sq)
		assert.False(t, sq.IsRelative())
		require.Len(t, sq.Selectors(), 1)
		assert.Equal(t, Name, sq.Selectors()[0].Kind)
		assert.Equal(t, "x", sq.Selectors()[0].Name)
	})

	t.Run("returns_singular_for_relative", func(t *testing.T) {
		t.Parallel()
		q := NewPathQuery(false, Child(NameSelector("a")), Child(IndexSelector(1)))
		sq := q.Singular()
		require.NotNil(t, sq)
		assert.True(t, sq.IsRelative())
		require.Len(t, sq.Selectors(), 2)
		assert.Equal(t, "a", sq.Selectors()[0].Name)
		assert.Equal(t, int64(1), sq.Selectors()[1].Index)
	})
}

func TestSingularQueryString(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		sq   *SingularQuery
		want string
	}{
		{
			name: "root_name",
			sq:   NewSingularQuery(false, NameSelector("x")),
			want: `$["x"]`,
		},
		{
			name: "relative_name",
			sq:   NewSingularQuery(true, NameSelector("x")),
			want: `@["x"]`,
		},
		{
			name: "root_name_index",
			sq:   NewSingularQuery(false, NameSelector("a"), IndexSelector(0)),
			want: `$["a"][0]`,
		},
		{
			name: "relative_index",
			sq:   NewSingularQuery(true, IndexSelector(3)),
			want: `@[3]`,
		},
		{
			name: "empty",
			sq:   NewSingularQuery(false),
			want: `$`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, tc.sq.String())
		})
	}
}

func TestSingularQueryFromPathQuery(t *testing.T) {
	t.Parallel()

	// Verify round-trip: PathQuery → Singular → String matches expected.
	q := NewPathQuery(true, Child(NameSelector("store")), Child(IndexSelector(0)))
	sq := q.Singular()
	require.NotNil(t, sq)
	assert.Equal(t, `$["store"][0]`, sq.String())
	assert.False(t, sq.IsRelative())
}
