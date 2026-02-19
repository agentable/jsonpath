package jsonpath

import (
	"os"
	"strings"
	"testing"

	"github.com/agentable/jsonpath/internal/ast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModuleInitialized(t *testing.T) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal("go.mod not found:", err)
	}
	content := string(data)

	if !strings.Contains(content, "module github.com/agentable/jsonpath") {
		t.Error("go.mod missing correct module path")
	}
	if !strings.Contains(content, "go 1.26") {
		t.Error("go.mod missing go 1.26 directive")
	}
}

func TestPath_Select_NameSelector(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		selector ast.Selector
		want     []any
	}{
		{
			name:     "select existing key",
			input:    map[string]any{"a": 1, "b": 2},
			selector: ast.NameSelector("a"),
			want:     []any{1},
		},
		{
			name:     "select missing key",
			input:    map[string]any{"a": 1},
			selector: ast.NameSelector("b"),
			want:     []any{},
		},
		{
			name:     "select from non-object",
			input:    []any{1, 2, 3},
			selector: ast.NameSelector("a"),
			want:     []any{},
		},
		{
			name:     "select nested object",
			input:    map[string]any{"a": map[string]any{"b": 42}},
			selector: ast.NameSelector("a"),
			want:     []any{map[string]any{"b": 42}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg := ast.Child(tt.selector)
			query := ast.NewPathQuery(true, seg)
			path := &Path{query: query}
			got := path.Select(tt.input)
			assert.Equal(t, tt.want, []any(got))
		})
	}
}

func TestPath_Select_IndexSelector(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		selector ast.Selector
		want     []any
	}{
		{
			name:     "select positive index",
			input:    []any{10, 20, 30},
			selector: ast.IndexSelector(1),
			want:     []any{20},
		},
		{
			name:     "select negative index",
			input:    []any{10, 20, 30},
			selector: ast.IndexSelector(-1),
			want:     []any{30},
		},
		{
			name:     "select negative index -2",
			input:    []any{10, 20, 30},
			selector: ast.IndexSelector(-2),
			want:     []any{20},
		},
		{
			name:     "select out of bounds positive",
			input:    []any{10, 20},
			selector: ast.IndexSelector(5),
			want:     []any{},
		},
		{
			name:     "select out of bounds negative",
			input:    []any{10, 20},
			selector: ast.IndexSelector(-5),
			want:     []any{},
		},
		{
			name:     "select from non-array",
			input:    map[string]any{"a": 1},
			selector: ast.IndexSelector(0),
			want:     []any{},
		},
		{
			name:     "select from empty array",
			input:    []any{},
			selector: ast.IndexSelector(0),
			want:     []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg := ast.Child(tt.selector)
			query := ast.NewPathQuery(true, seg)
			path := &Path{query: query}
			got := path.Select(tt.input)
			assert.Equal(t, tt.want, []any(got))
		})
	}
}

func TestPath_Select_SliceSelector(t *testing.T) {
	tests := []struct {
		name  string
		input any
		slice ast.SliceArgs
		want  []any
	}{
		{
			name:  "slice with start and end",
			input: []any{0, 1, 2, 3, 4},
			slice: ast.SliceArgs{Start: 1, End: 3, HasStart: true, HasEnd: true},
			want:  []any{1, 2},
		},
		{
			name:  "slice with only start",
			input: []any{0, 1, 2, 3, 4},
			slice: ast.SliceArgs{Start: 2, HasStart: true},
			want:  []any{2, 3, 4},
		},
		{
			name:  "slice with only end",
			input: []any{0, 1, 2, 3, 4},
			slice: ast.SliceArgs{End: 3, HasEnd: true},
			want:  []any{0, 1, 2},
		},
		{
			name:  "slice with step",
			input: []any{0, 1, 2, 3, 4, 5},
			slice: ast.SliceArgs{Start: 0, End: 6, Step: 2, HasStart: true, HasEnd: true, HasStep: true},
			want:  []any{0, 2, 4},
		},
		{
			name:  "slice with negative start",
			input: []any{0, 1, 2, 3, 4},
			slice: ast.SliceArgs{Start: -2, HasStart: true},
			want:  []any{3, 4},
		},
		{
			name:  "slice with negative end",
			input: []any{0, 1, 2, 3, 4},
			slice: ast.SliceArgs{End: -1, HasEnd: true},
			want:  []any{0, 1, 2, 3},
		},
		{
			name:  "slice with negative step",
			input: []any{0, 1, 2, 3, 4},
			slice: ast.SliceArgs{Start: 4, End: 0, Step: -1, HasStart: true, HasEnd: true, HasStep: true},
			want:  []any{4, 3, 2, 1},
		},
		{
			name:  "slice with step 0 returns empty",
			input: []any{0, 1, 2, 3, 4},
			slice: ast.SliceArgs{Step: 0, HasStep: true},
			want:  []any{},
		},
		{
			name:  "slice from empty array",
			input: []any{},
			slice: ast.SliceArgs{Start: 0, End: 5, HasStart: true, HasEnd: true},
			want:  []any{},
		},
		{
			name:  "slice from non-array",
			input: map[string]any{"a": 1},
			slice: ast.SliceArgs{Start: 0, End: 5, HasStart: true, HasEnd: true},
			want:  []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg := ast.Child(ast.SliceSelector(tt.slice))
			query := ast.NewPathQuery(true, seg)
			path := &Path{query: query}
			got := path.Select(tt.input)
			assert.Equal(t, tt.want, []any(got))
		})
	}
}

func TestPath_Select_WildcardSelector(t *testing.T) {
	tests := []struct {
		name  string
		input any
		want  []any
	}{
		{
			name:  "wildcard on object",
			input: map[string]any{"a": 1, "b": 2, "c": 3},
			want:  []any{1, 2, 3},
		},
		{
			name:  "wildcard on array",
			input: []any{10, 20, 30},
			want:  []any{10, 20, 30},
		},
		{
			name:  "wildcard on empty object",
			input: map[string]any{},
			want:  []any{},
		},
		{
			name:  "wildcard on empty array",
			input: []any{},
			want:  []any{},
		},
		{
			name:  "wildcard on primitive",
			input: 42,
			want:  []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg := ast.Child(ast.WildcardSelector())
			query := ast.NewPathQuery(true, seg)
			path := &Path{query: query}
			got := path.Select(tt.input)
			// Wildcard on maps returns values in undefined order, so we check length and contents
			assert.Len(t, got, len(tt.want))
			if len(tt.want) > 0 {
				assert.ElementsMatch(t, tt.want, []any(got))
			}
		})
	}
}

func TestPath_Select_MultipleSelectors(t *testing.T) {
	input := map[string]any{
		"a": 1,
		"b": 2,
		"c": 3,
	}

	seg := ast.Child(ast.NameSelector("a"), ast.NameSelector("c"))
	query := ast.NewPathQuery(true, seg)
	path := &Path{query: query}
	got := path.Select(input)

	assert.Equal(t, []any{1, 3}, []any(got))
}

func TestPath_Select_MultipleSegments(t *testing.T) {
	input := map[string]any{
		"store": map[string]any{
			"book": []any{
				map[string]any{"title": "Book 1", "price": 10},
				map[string]any{"title": "Book 2", "price": 20},
			},
		},
	}

	seg1 := ast.Child(ast.NameSelector("store"))
	seg2 := ast.Child(ast.NameSelector("book"))
	seg3 := ast.Child(ast.IndexSelector(0))
	seg4 := ast.Child(ast.NameSelector("title"))

	query := ast.NewPathQuery(true, seg1, seg2, seg3, seg4)
	path := &Path{query: query}
	got := path.Select(input)

	assert.Equal(t, []any{"Book 1"}, []any(got))
}

func TestPath_Select_DescendantSelector(t *testing.T) {
	input := map[string]any{
		"a": 1,
		"b": map[string]any{
			"a": 2,
			"c": map[string]any{
				"a": 3,
			},
		},
		"d": []any{
			map[string]any{"a": 4},
			map[string]any{"b": 5},
		},
	}

	seg := ast.Descendant(ast.NameSelector("a"))
	query := ast.NewPathQuery(true, seg)
	path := &Path{query: query}
	got := path.Select(input)

	assert.ElementsMatch(t, []any{1, 2, 3, 4}, []any(got))
}

func TestPath_Select_DescendantWildcard(t *testing.T) {
	input := map[string]any{
		"a": 1,
		"b": map[string]any{
			"c": 2,
			"d": 3,
		},
		"e": []any{4, 5},
	}

	seg := ast.Descendant(ast.WildcardSelector())
	query := ast.NewPathQuery(true, seg)
	path := &Path{query: query}
	got := path.Select(input)

	// Should select all values recursively
	assert.ElementsMatch(t, []any{
		1,
		map[string]any{"c": 2, "d": 3},
		2,
		3,
		[]any{4, 5},
		4,
		5,
	}, []any(got))
}

func TestPath_Select_NilQuery(t *testing.T) {
	path := &Path{query: nil}
	got := path.Select(map[string]any{"a": 1})
	assert.Nil(t, got)
}

func TestPath_Select_ComplexPath(t *testing.T) {
	input := map[string]any{
		"store": map[string]any{
			"book": []any{
				map[string]any{"category": "reference", "author": "Nigel Rees", "title": "Sayings of the Century", "price": 8.95},
				map[string]any{"category": "fiction", "author": "Evelyn Waugh", "title": "Sword of Honour", "price": 12.99},
				map[string]any{"category": "fiction", "author": "Herman Melville", "title": "Moby Dick", "isbn": "0-553-21311-3", "price": 8.99},
			},
		},
	}

	// $['store']['book'][*]['price']
	seg1 := ast.Child(ast.NameSelector("store"))
	seg2 := ast.Child(ast.NameSelector("book"))
	seg3 := ast.Child(ast.WildcardSelector())
	seg4 := ast.Child(ast.NameSelector("price"))

	query := ast.NewPathQuery(true, seg1, seg2, seg3, seg4)
	path := &Path{query: query}
	got := path.Select(input)

	assert.Equal(t, []any{8.95, 12.99, 8.99}, []any(got))
}

func TestNormalizeIndex(t *testing.T) {
	tests := []struct {
		name   string
		idx    int64
		length int
		want   int
	}{
		{"positive in bounds", 2, 5, 2},
		{"zero", 0, 5, 0},
		{"last element", 4, 5, 4},
		{"negative -1", -1, 5, 4},
		{"negative -2", -2, 5, 3},
		{"negative all", -5, 5, 0},
		{"out of bounds positive", 10, 5, -1},
		{"out of bounds negative", -10, 5, -1},
		{"empty array", 0, 0, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeIndex(tt.idx, tt.length)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppendSlice_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		arr  []any
		args ast.SliceArgs
		want []any
	}{
		{
			name: "start beyond end with positive step",
			arr:  []any{0, 1, 2, 3, 4},
			args: ast.SliceArgs{Start: 3, End: 1, Step: 1, HasStart: true, HasEnd: true, HasStep: true},
			want: []any{},
		},
		{
			name: "negative step from start to end",
			arr:  []any{0, 1, 2, 3, 4},
			args: ast.SliceArgs{Start: 3, End: 1, Step: -1, HasStart: true, HasEnd: true, HasStep: true},
			want: []any{3, 2},
		},
		{
			name: "large negative start normalizes to 0",
			arr:  []any{0, 1, 2},
			args: ast.SliceArgs{Start: -100, HasStart: true},
			want: []any{0, 1, 2},
		},
		{
			name: "large positive end normalizes to length",
			arr:  []any{0, 1, 2},
			args: ast.SliceArgs{End: 100, HasEnd: true},
			want: []any{0, 1, 2},
		},
		{
			name: "step larger than array",
			arr:  []any{0, 1, 2, 3, 4},
			args: ast.SliceArgs{Step: 10, HasStep: true},
			want: []any{0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendSlice([]any{}, tt.arr, tt.args)
			assert.Equal(t, tt.want, got)
		})
	}
}

func BenchmarkSelect_NameSelector(b *testing.B) {
	input := map[string]any{
		"a": 1,
		"b": 2,
		"c": 3,
	}
	seg := ast.Child(ast.NameSelector("b"))
	query := ast.NewPathQuery(true, seg)
	path := &Path{query: query}

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_IndexSelector(b *testing.B) {
	input := []any{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	seg := ast.Child(ast.IndexSelector(5))
	query := ast.NewPathQuery(true, seg)
	path := &Path{query: query}

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_SliceSelector(b *testing.B) {
	input := make([]any, 100)
	for i := range input {
		input[i] = i
	}
	seg := ast.Child(ast.SliceSelector(ast.SliceArgs{
		Start: 10, End: 50, Step: 2,
		HasStart: true, HasEnd: true, HasStep: true,
	}))
	query := ast.NewPathQuery(true, seg)
	path := &Path{query: query}

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_WildcardSelector(b *testing.B) {
	input := map[string]any{
		"a": 1, "b": 2, "c": 3, "d": 4, "e": 5,
		"f": 6, "g": 7, "h": 8, "i": 9, "j": 10,
	}
	seg := ast.Child(ast.WildcardSelector())
	query := ast.NewPathQuery(true, seg)
	path := &Path{query: query}

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_DescendantSelector(b *testing.B) {
	input := map[string]any{
		"a": 1,
		"b": map[string]any{
			"a": 2,
			"c": map[string]any{
				"a": 3,
				"d": map[string]any{
					"a": 4,
				},
			},
		},
	}
	seg := ast.Descendant(ast.NameSelector("a"))
	query := ast.NewPathQuery(true, seg)
	path := &Path{query: query}

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_ComplexPath(b *testing.B) {
	input := map[string]any{
		"store": map[string]any{
			"book": []any{
				map[string]any{"title": "Book 1", "price": 10},
				map[string]any{"title": "Book 2", "price": 20},
				map[string]any{"title": "Book 3", "price": 30},
				map[string]any{"title": "Book 4", "price": 40},
				map[string]any{"title": "Book 5", "price": 50},
			},
		},
	}

	seg1 := ast.Child(ast.NameSelector("store"))
	seg2 := ast.Child(ast.NameSelector("book"))
	seg3 := ast.Child(ast.WildcardSelector())
	seg4 := ast.Child(ast.NameSelector("price"))

	query := ast.NewPathQuery(true, seg1, seg2, seg3, seg4)
	path := &Path{query: query}

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func TestPath_Select_FilterSelector(t *testing.T) {
	// Filter selectors are not yet implemented, so they should select nothing
	input := []any{
		map[string]any{"price": 10},
		map[string]any{"price": 20},
	}

	seg := ast.Child(ast.FilterSelector(&ast.FilterExpr{}))
	query := ast.NewPathQuery(true, seg)
	path := &Path{query: query}
	got := path.Select(input)

	require.Empty(t, got, "filter selectors should select nothing until implemented")
}

func TestPath_SelectLocated_NameSelector(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		selector ast.Selector
		want     []*LocatedNode
	}{
		{
			name:     "select existing key",
			input:    map[string]any{"a": 1, "b": 2},
			selector: ast.NameSelector("a"),
			want: []*LocatedNode{
				{Value: 1, Path: NormalizedPath{NameElement("a")}},
			},
		},
		{
			name:     "select missing key",
			input:    map[string]any{"a": 1},
			selector: ast.NameSelector("b"),
			want:     []*LocatedNode{},
		},
		{
			name:     "select from non-object",
			input:    []any{1, 2, 3},
			selector: ast.NameSelector("a"),
			want:     []*LocatedNode{},
		},
		{
			name:     "select nested object",
			input:    map[string]any{"a": map[string]any{"b": 42}},
			selector: ast.NameSelector("a"),
			want: []*LocatedNode{
				{Value: map[string]any{"b": 42}, Path: NormalizedPath{NameElement("a")}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg := ast.Child(tt.selector)
			query := ast.NewPathQuery(true, seg)
			path := &Path{query: query}
			got := path.SelectLocated(tt.input)
			assert.Equal(t, tt.want, []*LocatedNode(got))
		})
	}
}

func TestPath_SelectLocated_IndexSelector(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		selector ast.Selector
		want     []*LocatedNode
	}{
		{
			name:     "select positive index",
			input:    []any{10, 20, 30},
			selector: ast.IndexSelector(1),
			want: []*LocatedNode{
				{Value: 20, Path: NormalizedPath{IndexElement(1)}},
			},
		},
		{
			name:     "select negative index",
			input:    []any{10, 20, 30},
			selector: ast.IndexSelector(-1),
			want: []*LocatedNode{
				{Value: 30, Path: NormalizedPath{IndexElement(2)}},
			},
		},
		{
			name:     "select out of bounds",
			input:    []any{10, 20},
			selector: ast.IndexSelector(5),
			want:     []*LocatedNode{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seg := ast.Child(tt.selector)
			query := ast.NewPathQuery(true, seg)
			path := &Path{query: query}
			got := path.SelectLocated(tt.input)
			assert.Equal(t, tt.want, []*LocatedNode(got))
		})
	}
}

func TestPath_SelectLocated_SliceSelector(t *testing.T) {
	input := []any{0, 1, 2, 3, 4}
	slice := ast.SliceArgs{Start: 1, End: 4, Step: 2, HasStart: true, HasEnd: true, HasStep: true}

	seg := ast.Child(ast.SliceSelector(slice))
	query := ast.NewPathQuery(true, seg)
	path := &Path{query: query}
	got := path.SelectLocated(input)

	want := []*LocatedNode{
		{Value: 1, Path: NormalizedPath{IndexElement(1)}},
		{Value: 3, Path: NormalizedPath{IndexElement(3)}},
	}
	assert.Equal(t, want, []*LocatedNode(got))
}

func TestPath_SelectLocated_WildcardSelector(t *testing.T) {
	t.Run("wildcard on object", func(t *testing.T) {
		input := map[string]any{"a": 1, "b": 2}
		seg := ast.Child(ast.WildcardSelector())
		query := ast.NewPathQuery(true, seg)
		path := &Path{query: query}
		got := path.SelectLocated(input)

		assert.Len(t, got, 2)
		// Check that we have the right paths (order may vary for maps)
		paths := make(map[string]any)
		for _, node := range got {
			paths[node.Path.String()] = node.Value
		}
		assert.Equal(t, 1, paths["$['a']"])
		assert.Equal(t, 2, paths["$['b']"])
	})

	t.Run("wildcard on array", func(t *testing.T) {
		input := []any{10, 20, 30}
		seg := ast.Child(ast.WildcardSelector())
		query := ast.NewPathQuery(true, seg)
		path := &Path{query: query}
		got := path.SelectLocated(input)

		want := []*LocatedNode{
			{Value: 10, Path: NormalizedPath{IndexElement(0)}},
			{Value: 20, Path: NormalizedPath{IndexElement(1)}},
			{Value: 30, Path: NormalizedPath{IndexElement(2)}},
		}
		assert.Equal(t, want, []*LocatedNode(got))
	})
}

func TestPath_SelectLocated_MultipleSegments(t *testing.T) {
	input := map[string]any{
		"store": map[string]any{
			"book": []any{
				map[string]any{"title": "Book 1", "price": 10},
				map[string]any{"title": "Book 2", "price": 20},
			},
		},
	}

	seg1 := ast.Child(ast.NameSelector("store"))
	seg2 := ast.Child(ast.NameSelector("book"))
	seg3 := ast.Child(ast.IndexSelector(0))
	seg4 := ast.Child(ast.NameSelector("title"))

	query := ast.NewPathQuery(true, seg1, seg2, seg3, seg4)
	path := &Path{query: query}
	got := path.SelectLocated(input)

	want := []*LocatedNode{
		{
			Value: "Book 1",
			Path: NormalizedPath{
				NameElement("store"),
				NameElement("book"),
				IndexElement(0),
				NameElement("title"),
			},
		},
	}
	assert.Equal(t, want, []*LocatedNode(got))
}

func TestPath_SelectLocated_DescendantSelector(t *testing.T) {
	input := map[string]any{
		"a": 1,
		"b": map[string]any{
			"a": 2,
			"c": map[string]any{
				"a": 3,
			},
		},
		"d": []any{
			map[string]any{"a": 4},
		},
	}

	seg := ast.Descendant(ast.NameSelector("a"))
	query := ast.NewPathQuery(true, seg)
	path := &Path{query: query}
	got := path.SelectLocated(input)

	// Build a map of path -> value for easier comparison
	pathMap := make(map[string]any)
	for _, node := range got {
		pathMap[node.Path.String()] = node.Value
	}

	assert.Len(t, pathMap, 4)
	assert.Equal(t, 1, pathMap["$['a']"])
	assert.Equal(t, 2, pathMap["$['b']['a']"])
	assert.Equal(t, 3, pathMap["$['b']['c']['a']"])
	assert.Equal(t, 4, pathMap["$['d'][0]['a']"])
}

func TestPath_SelectLocated_ComplexPath(t *testing.T) {
	input := map[string]any{
		"store": map[string]any{
			"book": []any{
				map[string]any{"title": "Book 1", "price": 8.95},
				map[string]any{"title": "Book 2", "price": 12.99},
				map[string]any{"title": "Book 3", "price": 8.99},
			},
		},
	}

	// $['store']['book'][*]['price']
	seg1 := ast.Child(ast.NameSelector("store"))
	seg2 := ast.Child(ast.NameSelector("book"))
	seg3 := ast.Child(ast.WildcardSelector())
	seg4 := ast.Child(ast.NameSelector("price"))

	query := ast.NewPathQuery(true, seg1, seg2, seg3, seg4)
	path := &Path{query: query}
	got := path.SelectLocated(input)

	want := []*LocatedNode{
		{
			Value: 8.95,
			Path: NormalizedPath{
				NameElement("store"),
				NameElement("book"),
				IndexElement(0),
				NameElement("price"),
			},
		},
		{
			Value: 12.99,
			Path: NormalizedPath{
				NameElement("store"),
				NameElement("book"),
				IndexElement(1),
				NameElement("price"),
			},
		},
		{
			Value: 8.99,
			Path: NormalizedPath{
				NameElement("store"),
				NameElement("book"),
				IndexElement(2),
				NameElement("price"),
			},
		},
	}
	assert.Equal(t, want, []*LocatedNode(got))
}

func TestPath_SelectLocated_NilQuery(t *testing.T) {
	path := &Path{query: nil}
	got := path.SelectLocated(map[string]any{"a": 1})
	assert.Nil(t, got)
}

func TestLocatedNodeList_Methods(t *testing.T) {
	list := LocatedNodeList{
		{Value: 1, Path: NormalizedPath{NameElement("a")}},
		{Value: 2, Path: NormalizedPath{NameElement("b")}},
		{Value: 3, Path: NormalizedPath{IndexElement(0)}},
	}

	t.Run("Values", func(t *testing.T) {
		values := make([]any, 0, len(list))
		for v := range list.Values() {
			values = append(values, v)
		}
		assert.Equal(t, []any{1, 2, 3}, values)
	})

	t.Run("Paths", func(t *testing.T) {
		paths := make([]string, 0, len(list))
		for p := range list.Paths() {
			paths = append(paths, p.String())
		}
		assert.Equal(t, []string{"$['a']", "$['b']", "$[0]"}, paths)
	})

	t.Run("All", func(t *testing.T) {
		nodes := make([]*LocatedNode, 0, len(list))
		for n := range list.All() {
			nodes = append(nodes, n)
		}
		assert.Equal(t, []*LocatedNode(list), nodes)
	})
}

func TestLocatedNodeList_Deduplicate(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		list LocatedNodeList
		exp  LocatedNodeList
	}{
		{
			name: "empty",
			list: LocatedNodeList{},
			exp:  LocatedNodeList{},
		},
		{
			name: "single",
			list: LocatedNodeList{
				{Value: "a", Path: NormalizedPath{NameElement("x")}},
			},
			exp: LocatedNodeList{
				{Value: "a", Path: NormalizedPath{NameElement("x")}},
			},
		},
		{
			name: "no_duplicates",
			list: LocatedNodeList{
				{Value: "a", Path: NormalizedPath{NameElement("x")}},
				{Value: "b", Path: NormalizedPath{NameElement("y")}},
				{Value: "c", Path: NormalizedPath{IndexElement(0)}},
			},
			exp: LocatedNodeList{
				{Value: "a", Path: NormalizedPath{NameElement("x")}},
				{Value: "b", Path: NormalizedPath{NameElement("y")}},
				{Value: "c", Path: NormalizedPath{IndexElement(0)}},
			},
		},
		{
			name: "duplicates_same_value",
			list: LocatedNodeList{
				{Value: "a", Path: NormalizedPath{NameElement("x")}},
				{Value: "a", Path: NormalizedPath{NameElement("x")}},
				{Value: "b", Path: NormalizedPath{NameElement("y")}},
			},
			exp: LocatedNodeList{
				{Value: "a", Path: NormalizedPath{NameElement("x")}},
				{Value: "b", Path: NormalizedPath{NameElement("y")}},
			},
		},
		{
			name: "duplicates_diff_value",
			list: LocatedNodeList{
				{Value: "a", Path: NormalizedPath{NameElement("x")}},
				{Value: "different", Path: NormalizedPath{NameElement("x")}},
				{Value: "b", Path: NormalizedPath{NameElement("y")}},
			},
			exp: LocatedNodeList{
				{Value: "a", Path: NormalizedPath{NameElement("x")}},
				{Value: "b", Path: NormalizedPath{NameElement("y")}},
			},
		},
		{
			name: "multiple_duplicates",
			list: LocatedNodeList{
				{Value: "a", Path: NormalizedPath{NameElement("x")}},
				{Value: "b", Path: NormalizedPath{NameElement("y")}},
				{Value: "c", Path: NormalizedPath{NameElement("x")}},
				{Value: "d", Path: NormalizedPath{NameElement("z")}},
				{Value: "e", Path: NormalizedPath{NameElement("y")}},
			},
			exp: LocatedNodeList{
				{Value: "a", Path: NormalizedPath{NameElement("x")}},
				{Value: "b", Path: NormalizedPath{NameElement("y")}},
				{Value: "d", Path: NormalizedPath{NameElement("z")}},
			},
		},
		{
			name: "nested_paths",
			list: LocatedNodeList{
				{Value: 1, Path: NormalizedPath{NameElement("a"), IndexElement(0)}},
				{Value: 2, Path: NormalizedPath{NameElement("a"), IndexElement(1)}},
				{Value: 3, Path: NormalizedPath{NameElement("a"), IndexElement(0)}},
			},
			exp: LocatedNodeList{
				{Value: 1, Path: NormalizedPath{NameElement("a"), IndexElement(0)}},
				{Value: 2, Path: NormalizedPath{NameElement("a"), IndexElement(1)}},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			got := tc.list.Deduplicate()
			a.Equal(len(tc.exp), len(got))
			for i := range tc.exp {
				a.Equal(tc.exp[i].Value, got[i].Value)
				a.Equal(tc.exp[i].Path, got[i].Path)
			}
		})
	}
}

func TestLocatedNodeList_Sort(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		list LocatedNodeList
		exp  LocatedNodeList
	}{
		{
			name: "empty",
			list: LocatedNodeList{},
			exp:  LocatedNodeList{},
		},
		{
			name: "single",
			list: LocatedNodeList{
				{Value: "a", Path: NormalizedPath{NameElement("x")}},
			},
			exp: LocatedNodeList{
				{Value: "a", Path: NormalizedPath{NameElement("x")}},
			},
		},
		{
			name: "already_sorted",
			list: LocatedNodeList{
				{Value: "a", Path: NormalizedPath{NameElement("a")}},
				{Value: "b", Path: NormalizedPath{NameElement("b")}},
				{Value: "c", Path: NormalizedPath{NameElement("c")}},
			},
			exp: LocatedNodeList{
				{Value: "a", Path: NormalizedPath{NameElement("a")}},
				{Value: "b", Path: NormalizedPath{NameElement("b")}},
				{Value: "c", Path: NormalizedPath{NameElement("c")}},
			},
		},
		{
			name: "reverse_order",
			list: LocatedNodeList{
				{Value: "c", Path: NormalizedPath{NameElement("c")}},
				{Value: "b", Path: NormalizedPath{NameElement("b")}},
				{Value: "a", Path: NormalizedPath{NameElement("a")}},
			},
			exp: LocatedNodeList{
				{Value: "a", Path: NormalizedPath{NameElement("a")}},
				{Value: "b", Path: NormalizedPath{NameElement("b")}},
				{Value: "c", Path: NormalizedPath{NameElement("c")}},
			},
		},
		{
			name: "indexes_before_names",
			list: LocatedNodeList{
				{Value: "name", Path: NormalizedPath{NameElement("x")}},
				{Value: "index", Path: NormalizedPath{IndexElement(0)}},
			},
			exp: LocatedNodeList{
				{Value: "index", Path: NormalizedPath{IndexElement(0)}},
				{Value: "name", Path: NormalizedPath{NameElement("x")}},
			},
		},
		{
			name: "mixed_indexes_and_names",
			list: LocatedNodeList{
				{Value: "n2", Path: NormalizedPath{NameElement("z")}},
				{Value: "i2", Path: NormalizedPath{IndexElement(5)}},
				{Value: "n1", Path: NormalizedPath{NameElement("a")}},
				{Value: "i1", Path: NormalizedPath{IndexElement(0)}},
			},
			exp: LocatedNodeList{
				{Value: "i1", Path: NormalizedPath{IndexElement(0)}},
				{Value: "i2", Path: NormalizedPath{IndexElement(5)}},
				{Value: "n1", Path: NormalizedPath{NameElement("a")}},
				{Value: "n2", Path: NormalizedPath{NameElement("z")}},
			},
		},
		{
			name: "nested_paths",
			list: LocatedNodeList{
				{Value: 3, Path: NormalizedPath{NameElement("b"), IndexElement(0)}},
				{Value: 1, Path: NormalizedPath{NameElement("a"), IndexElement(0)}},
				{Value: 4, Path: NormalizedPath{NameElement("b"), IndexElement(1)}},
				{Value: 2, Path: NormalizedPath{NameElement("a"), IndexElement(1)}},
			},
			exp: LocatedNodeList{
				{Value: 1, Path: NormalizedPath{NameElement("a"), IndexElement(0)}},
				{Value: 2, Path: NormalizedPath{NameElement("a"), IndexElement(1)}},
				{Value: 3, Path: NormalizedPath{NameElement("b"), IndexElement(0)}},
				{Value: 4, Path: NormalizedPath{NameElement("b"), IndexElement(1)}},
			},
		},
		{
			name: "different_lengths",
			list: LocatedNodeList{
				{Value: "long", Path: NormalizedPath{NameElement("a"), NameElement("b"), IndexElement(0)}},
				{Value: "short", Path: NormalizedPath{NameElement("a")}},
				{Value: "medium", Path: NormalizedPath{NameElement("a"), NameElement("b")}},
			},
			exp: LocatedNodeList{
				{Value: "short", Path: NormalizedPath{NameElement("a")}},
				{Value: "medium", Path: NormalizedPath{NameElement("a"), NameElement("b")}},
				{Value: "long", Path: NormalizedPath{NameElement("a"), NameElement("b"), IndexElement(0)}},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			// Make a copy to avoid modifying the test case
			list := make(LocatedNodeList, len(tc.list))
			copy(list, tc.list)

			list.Sort()
			a.Equal(len(tc.exp), len(list))
			for i := range tc.exp {
				a.Equal(tc.exp[i].Value, list[i].Value)
				a.Equal(tc.exp[i].Path, list[i].Path)
			}
		})
	}
}

func BenchmarkSelectLocated_NameSelector(b *testing.B) {
	input := map[string]any{
		"a": 1,
		"b": 2,
		"c": 3,
	}
	seg := ast.Child(ast.NameSelector("b"))
	query := ast.NewPathQuery(true, seg)
	path := &Path{query: query}

	b.ResetTimer()
	for b.Loop() {
		_ = path.SelectLocated(input)
	}
}

func BenchmarkSelectLocated_ComplexPath(b *testing.B) {
	input := map[string]any{
		"store": map[string]any{
			"book": []any{
				map[string]any{"title": "Book 1", "price": 10},
				map[string]any{"title": "Book 2", "price": 20},
				map[string]any{"title": "Book 3", "price": 30},
				map[string]any{"title": "Book 4", "price": 40},
				map[string]any{"title": "Book 5", "price": 50},
			},
		},
	}

	seg1 := ast.Child(ast.NameSelector("store"))
	seg2 := ast.Child(ast.NameSelector("book"))
	seg3 := ast.Child(ast.WildcardSelector())
	seg4 := ast.Child(ast.NameSelector("price"))

	query := ast.NewPathQuery(true, seg1, seg2, seg3, seg4)
	path := &Path{query: query}

	b.ResetTimer()
	for b.Loop() {
		_ = path.SelectLocated(input)
	}
}

func TestQueryJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		path    string
		want    []any
		wantErr bool
	}{
		{
			name: "simple name selector",
			json: `{"a": 1, "b": 2}`,
			path: "$.a",
			want: []any{float64(1)},
		},
		{
			name: "array index selector",
			json: `[10, 20, 30]`,
			path: "$[1]",
			want: []any{float64(20)},
		},
		{
			name: "nested path",
			json: `{"store": {"book": [{"title": "Book 1", "price": 8.95}]}}`,
			path: "$.store.book[0].title",
			want: []any{"Book 1"},
		},
		{
			name: "wildcard selector",
			json: `{"a": 1, "b": 2, "c": 3}`,
			path: "$[*]",
			want: []any{float64(1), float64(2), float64(3)},
		},
		{
			name:    "invalid json",
			json:    `{invalid}`,
			path:    "$.a",
			wantErr: true,
		},
		{
			name: "empty result",
			json: `{"a": 1}`,
			path: "$.b",
			want: []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := MustParse(tt.path)
			got, err := QueryJSON([]byte(tt.json), path)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if len(tt.want) > 0 && tt.path == "$[*]" {
				// Wildcard on maps returns values in undefined order
				assert.ElementsMatch(t, tt.want, []any(got))
			} else {
				assert.Equal(t, tt.want, []any(got))
			}
		})
	}
}

func TestQueryJSONLocated(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		path    string
		want    []*LocatedNode
		wantErr bool
	}{
		{
			name: "simple name selector",
			json: `{"a": 1, "b": 2}`,
			path: "$.a",
			want: []*LocatedNode{
				{Value: float64(1), Path: NormalizedPath{NameElement("a")}},
			},
		},
		{
			name: "array index selector",
			json: `[10, 20, 30]`,
			path: "$[1]",
			want: []*LocatedNode{
				{Value: float64(20), Path: NormalizedPath{IndexElement(1)}},
			},
		},
		{
			name: "nested path",
			json: `{"store": {"book": [{"title": "Book 1"}]}}`,
			path: "$.store.book[0].title",
			want: []*LocatedNode{
				{
					Value: "Book 1",
					Path: NormalizedPath{
						NameElement("store"),
						NameElement("book"),
						IndexElement(0),
						NameElement("title"),
					},
				},
			},
		},
		{
			name: "multiple results",
			json: `{"store": {"book": [{"price": 8.95}, {"price": 12.99}]}}`,
			path: "$.store.book[*].price",
			want: []*LocatedNode{
				{
					Value: float64(8.95),
					Path: NormalizedPath{
						NameElement("store"),
						NameElement("book"),
						IndexElement(0),
						NameElement("price"),
					},
				},
				{
					Value: float64(12.99),
					Path: NormalizedPath{
						NameElement("store"),
						NameElement("book"),
						IndexElement(1),
						NameElement("price"),
					},
				},
			},
		},
		{
			name:    "invalid json",
			json:    `{invalid}`,
			path:    "$.a",
			wantErr: true,
		},
		{
			name: "empty result",
			json: `{"a": 1}`,
			path: "$.b",
			want: []*LocatedNode{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := MustParse(tt.path)
			got, err := QueryJSONLocated([]byte(tt.json), path)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, []*LocatedNode(got))
		})
	}
}

func TestQueryJSON_ComplexDocument(t *testing.T) {
	jsonDoc := `{
		"store": {
			"book": [
				{
					"category": "reference",
					"author": "Nigel Rees",
					"title": "Sayings of the Century",
					"price": 8.95
				},
				{
					"category": "fiction",
					"author": "Evelyn Waugh",
					"title": "Sword of Honour",
					"price": 12.99
				},
				{
					"category": "fiction",
					"author": "Herman Melville",
					"title": "Moby Dick",
					"isbn": "0-553-21311-3",
					"price": 8.99
				}
			],
			"bicycle": {
				"color": "red",
				"price": 19.95
			}
		}
	}`

	t.Run("all book prices", func(t *testing.T) {
		path := MustParse("$.store.book[*].price")
		got, err := QueryJSON([]byte(jsonDoc), path)
		require.NoError(t, err)
		assert.Equal(t, []any{float64(8.95), float64(12.99), float64(8.99)}, []any(got))
	})

	t.Run("all authors", func(t *testing.T) {
		path := MustParse("$.store.book[*].author")
		got, err := QueryJSON([]byte(jsonDoc), path)
		require.NoError(t, err)
		assert.Equal(t, []any{"Nigel Rees", "Evelyn Waugh", "Herman Melville"}, []any(got))
	})

	t.Run("first book", func(t *testing.T) {
		path := MustParse("$.store.book[0]")
		got, err := QueryJSON([]byte(jsonDoc), path)
		require.NoError(t, err)
		require.Len(t, got, 1)
		book := got[0].(map[string]any)
		assert.Equal(t, "Sayings of the Century", book["title"])
		assert.Equal(t, float64(8.95), book["price"])
	})
}

func BenchmarkQueryJSON(b *testing.B) {
	jsonDoc := []byte(`{"store": {"book": [{"title": "Book 1", "price": 10}, {"title": "Book 2", "price": 20}]}}`)
	path := MustParse("$.store.book[*].price")

	b.ResetTimer()
	for b.Loop() {
		_, _ = QueryJSON(jsonDoc, path)
	}
}

func BenchmarkQueryJSONLocated(b *testing.B) {
	jsonDoc := []byte(`{"store": {"book": [{"title": "Book 1", "price": 10}, {"title": "Book 2", "price": 20}]}}`)
	path := MustParse("$.store.book[*].price")

	b.ResetTimer()
	for b.Loop() {
		_, _ = QueryJSONLocated(jsonDoc, path)
	}
}

// BenchmarkSelect suite covering name, index, slice, wildcard, filter, and descendant selectors

func BenchmarkSelect_Name_SmallObject(b *testing.B) {
	input := map[string]any{
		"a": 1, "b": 2, "c": 3, "d": 4, "e": 5,
	}
	path := MustParse("$.c")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Name_LargeObject(b *testing.B) {
	input := make(map[string]any, 100)
	for i := range 100 {
		input[string(rune('a'+i%26))+string(rune('0'+i/26))] = i
	}
	path := MustParse("$.z9")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Name_NestedPath(b *testing.B) {
	input := map[string]any{
		"level1": map[string]any{
			"level2": map[string]any{
				"level3": map[string]any{
					"level4": map[string]any{
						"value": 42,
					},
				},
			},
		},
	}
	path := MustParse("$.level1.level2.level3.level4.value")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Index_SmallArray(b *testing.B) {
	input := []any{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	path := MustParse("$[5]")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Index_LargeArray(b *testing.B) {
	input := make([]any, 1000)
	for i := range input {
		input[i] = i
	}
	path := MustParse("$[500]")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Index_NegativeIndex(b *testing.B) {
	input := make([]any, 100)
	for i := range input {
		input[i] = i
	}
	path := MustParse("$[-1]")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Slice_SmallRange(b *testing.B) {
	input := make([]any, 100)
	for i := range input {
		input[i] = i
	}
	path := MustParse("$[10:20]")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Slice_LargeRange(b *testing.B) {
	input := make([]any, 1000)
	for i := range input {
		input[i] = i
	}
	path := MustParse("$[100:900]")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Slice_WithStep(b *testing.B) {
	input := make([]any, 1000)
	for i := range input {
		input[i] = i
	}
	path := MustParse("$[0:1000:10]")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Slice_NegativeStep(b *testing.B) {
	input := make([]any, 100)
	for i := range input {
		input[i] = i
	}
	path := MustParse("$[99:0:-1]")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Wildcard_SmallObject(b *testing.B) {
	input := map[string]any{
		"a": 1, "b": 2, "c": 3, "d": 4, "e": 5,
	}
	path := MustParse("$[*]")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Wildcard_LargeObject(b *testing.B) {
	input := make(map[string]any, 100)
	for i := range 100 {
		input[string(rune('a'+i%26))+string(rune('0'+i/26))] = i
	}
	path := MustParse("$[*]")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Wildcard_Array(b *testing.B) {
	input := make([]any, 100)
	for i := range input {
		input[i] = i
	}
	path := MustParse("$[*]")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Wildcard_NestedArrays(b *testing.B) {
	input := make([]any, 10)
	for i := range input {
		inner := make([]any, 10)
		for j := range inner {
			inner[j] = i*10 + j
		}
		input[i] = inner
	}
	path := MustParse("$[*][*]")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Descendant_ShallowStructure(b *testing.B) {
	input := map[string]any{
		"a": 1,
		"b": 2,
		"c": 3,
		"d": 4,
		"e": 5,
	}
	path := MustParse("$..a")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Descendant_DeepStructure(b *testing.B) {
	input := map[string]any{
		"a": 1,
		"b": map[string]any{
			"a": 2,
			"c": map[string]any{
				"a": 3,
				"d": map[string]any{
					"a": 4,
					"e": map[string]any{
						"a": 5,
					},
				},
			},
		},
	}
	path := MustParse("$..a")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Descendant_WideStructure(b *testing.B) {
	input := make(map[string]any, 20)
	for i := range 20 {
		inner := make(map[string]any, 5)
		for j := range 5 {
			inner[string(rune('a'+j))] = i*5 + j
		}
		input[string(rune('A'+i))] = inner
	}
	path := MustParse("$..a")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Descendant_Wildcard(b *testing.B) {
	input := map[string]any{
		"a": 1,
		"b": map[string]any{
			"c": 2,
			"d": 3,
		},
		"e": []any{4, 5, 6},
		"f": map[string]any{
			"g": map[string]any{
				"h": 7,
			},
		},
	}
	path := MustParse("$..[*]")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_Filter_Placeholder(b *testing.B) {
	// Filter selectors are not yet implemented, but include a placeholder benchmark
	// for when they are added
	input := []any{
		map[string]any{"price": 10},
		map[string]any{"price": 20},
		map[string]any{"price": 30},
	}
	// This will select nothing until filters are implemented
	seg := ast.Child(ast.FilterSelector(&ast.FilterExpr{}))
	query := ast.NewPathQuery(true, seg)
	path := &Path{query: query}

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_RealWorld_BookStore(b *testing.B) {
	input := map[string]any{
		"store": map[string]any{
			"book": []any{
				map[string]any{"category": "reference", "author": "Nigel Rees", "title": "Sayings of the Century", "price": 8.95},
				map[string]any{"category": "fiction", "author": "Evelyn Waugh", "title": "Sword of Honour", "price": 12.99},
				map[string]any{"category": "fiction", "author": "Herman Melville", "title": "Moby Dick", "isbn": "0-553-21311-3", "price": 8.99},
				map[string]any{"category": "fiction", "author": "J. R. R. Tolkien", "title": "The Lord of the Rings", "isbn": "0-395-19395-8", "price": 22.99},
			},
			"bicycle": map[string]any{
				"color": "red",
				"price": 19.95,
			},
		},
	}
	path := MustParse("$.store.book[*].price")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_RealWorld_AllPrices(b *testing.B) {
	input := map[string]any{
		"store": map[string]any{
			"book": []any{
				map[string]any{"category": "reference", "author": "Nigel Rees", "title": "Sayings of the Century", "price": 8.95},
				map[string]any{"category": "fiction", "author": "Evelyn Waugh", "title": "Sword of Honour", "price": 12.99},
				map[string]any{"category": "fiction", "author": "Herman Melville", "title": "Moby Dick", "isbn": "0-553-21311-3", "price": 8.99},
				map[string]any{"category": "fiction", "author": "J. R. R. Tolkien", "title": "The Lord of the Rings", "isbn": "0-395-19395-8", "price": 22.99},
			},
			"bicycle": map[string]any{
				"color": "red",
				"price": 19.95,
			},
		},
	}
	path := MustParse("$..price")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func BenchmarkSelect_RealWorld_DeepJSON(b *testing.B) {
	input := map[string]any{
		"users": []any{
			map[string]any{
				"id":   1,
				"name": "Alice",
				"profile": map[string]any{
					"age":   30,
					"email": "alice@example.com",
					"address": map[string]any{
						"city":    "New York",
						"country": "USA",
					},
				},
			},
			map[string]any{
				"id":   2,
				"name": "Bob",
				"profile": map[string]any{
					"age":   25,
					"email": "bob@example.com",
					"address": map[string]any{
						"city":    "London",
						"country": "UK",
					},
				},
			},
		},
	}
	path := MustParse("$.users[*].profile.address.city")

	b.ResetTimer()
	for b.Loop() {
		_ = path.Select(input)
	}
}

func TestValid(t *testing.T) {
	tests := []struct {
		name  string
		expr  string
		valid bool
	}{
		{
			name:  "valid simple path",
			expr:  "$.store.book",
			valid: true,
		},
		{
			name:  "valid array index",
			expr:  "$[0]",
			valid: true,
		},
		{
			name:  "valid wildcard",
			expr:  "$[*]",
			valid: true,
		},
		{
			name:  "valid slice",
			expr:  "$[0:5:2]",
			valid: true,
		},
		{
			name:  "valid descendant",
			expr:  "$..book",
			valid: true,
		},
		{
			name:  "invalid missing root",
			expr:  "store.book",
			valid: false,
		},
		{
			name:  "invalid syntax",
			expr:  "$[",
			valid: false,
		},
		{
			name:  "invalid empty",
			expr:  "",
			valid: false,
		},
		{
			name:  "valid complex path",
			expr:  "$.store.book[*].author",
			valid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Valid(tt.expr)
			assert.Equal(t, tt.valid, got)
		})
	}
}

func TestQueryJSON_ErrUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "invalid json syntax",
			json: `{invalid}`,
		},
		{
			name: "unclosed object",
			json: `{"a": 1`,
		},
		{
			name: "unclosed array",
			json: `[1, 2, 3`,
		},
		{
			name: "trailing comma",
			json: `{"a": 1,}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := MustParse("$.a")
			_, err := QueryJSON([]byte(tt.json), path)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrUnmarshal)
		})
	}
}

func TestQueryJSONLocated_ErrUnmarshal(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{
			name: "invalid json syntax",
			json: `{invalid}`,
		},
		{
			name: "unclosed object",
			json: `{"a": 1`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := MustParse("$.a")
			_, err := QueryJSONLocated([]byte(tt.json), path)
			require.Error(t, err)
			assert.ErrorIs(t, err, ErrUnmarshal)
		})
	}
}
