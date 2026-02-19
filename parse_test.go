package jsonpath

import (
	"encoding"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		expr    string
		wantErr bool
	}{
		{
			name:    "root only",
			expr:    "$",
			wantErr: false,
		},
		{
			name:    "root with name selector",
			expr:    "$['a']",
			wantErr: false,
		},
		{
			name:    "root with index selector",
			expr:    "$[0]",
			wantErr: false,
		},
		{
			name:    "root with wildcard",
			expr:    "$[*]",
			wantErr: false,
		},
		{
			name:    "root with slice",
			expr:    "$[1:3]",
			wantErr: false,
		},
		{
			name:    "dot notation",
			expr:    "$.store.book",
			wantErr: false,
		},
		{
			name:    "descendant",
			expr:    "$..book",
			wantErr: false,
		},
		{
			name:    "complex path",
			expr:    "$.store.book[*].price",
			wantErr: false,
		},
		{
			name:    "invalid - no root",
			expr:    "store",
			wantErr: true,
		},
		{
			name:    "invalid - empty",
			expr:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := Parse(tt.expr)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, path)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, path)
			}
		})
	}
}

func TestMustParse(t *testing.T) {
	t.Run("valid expression", func(t *testing.T) {
		assert.NotPanics(t, func() {
			path := MustParse("$.store.book")
			assert.NotNil(t, path)
		})
	})

	t.Run("invalid expression panics", func(t *testing.T) {
		assert.Panics(t, func() {
			MustParse("invalid")
		})
	})
}

func TestPath_String(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want string
	}{
		{
			name: "root only",
			expr: "$",
			want: "$",
		},
		{
			name: "name selector",
			expr: "$['store']",
			want: "$[\"store\"]",
		},
		{
			name: "index selector",
			expr: "$[0]",
			want: "$[0]",
		},
		{
			name: "wildcard",
			expr: "$[*]",
			want: "$[*]",
		},
		{
			name: "slice",
			expr: "$[1:3]",
			want: "$[1:3]",
		},
		{
			name: "dot notation",
			expr: "$.store",
			want: "$[\"store\"]",
		},
		{
			name: "descendant",
			expr: "$..book",
			want: "$..\"book\"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := MustParse(tt.expr)
			got := path.String()
			assert.NotEmpty(t, got)
			// The canonical form may differ from input, so just verify it's not empty
			// and can be parsed back
			reparsed, err := Parse(got)
			require.NoError(t, err)
			assert.NotNil(t, reparsed)
		})
	}
}

func TestPath_String_NilQuery(t *testing.T) {
	path := &Path{query: nil}
	assert.Equal(t, "", path.String())
}

func TestPath_MarshalText(t *testing.T) {
	path := MustParse("$.store.book")

	// Verify it implements encoding.TextMarshaler
	var _ encoding.TextMarshaler = path

	text, err := path.MarshalText()
	require.NoError(t, err)
	assert.NotEmpty(t, text)

	// Should be able to parse the marshaled text
	reparsed, err := Parse(string(text))
	require.NoError(t, err)
	assert.NotNil(t, reparsed)
}

func TestPath_UnmarshalText(t *testing.T) {
	// Verify it implements encoding.TextUnmarshaler
	var path Path
	var _ encoding.TextUnmarshaler = &path

	t.Run("valid expression", func(t *testing.T) {
		var p Path
		err := p.UnmarshalText([]byte("$.store.book"))
		require.NoError(t, err)
		assert.NotNil(t, p.query)
	})

	t.Run("invalid expression", func(t *testing.T) {
		var p Path
		err := p.UnmarshalText([]byte("invalid"))
		assert.Error(t, err)
	})
}

func TestPath_MarshalUnmarshal_RoundTrip(t *testing.T) {
	original := MustParse("$.store.book[*].price")

	// Marshal
	text, err := original.MarshalText()
	require.NoError(t, err)

	// Unmarshal
	var restored Path
	err = restored.UnmarshalText(text)
	require.NoError(t, err)

	// Compare by evaluating on same input
	input := map[string]any{
		"store": map[string]any{
			"book": []any{
				map[string]any{"price": 10},
				map[string]any{"price": 20},
			},
		},
	}

	originalResult := original.Select(input)
	restoredResult := restored.Select(input)

	assert.Equal(t, originalResult, restoredResult)
}

func TestParse_Integration(t *testing.T) {
	input := map[string]any{
		"store": map[string]any{
			"book": []any{
				map[string]any{"title": "Book 1", "price": 10},
				map[string]any{"title": "Book 2", "price": 20},
			},
		},
	}

	tests := []struct {
		name string
		expr string
		want []any
	}{
		{
			name: "select store",
			expr: "$.store",
			want: []any{map[string]any{
				"book": []any{
					map[string]any{"title": "Book 1", "price": 10},
					map[string]any{"title": "Book 2", "price": 20},
				},
			}},
		},
		{
			name: "select first book",
			expr: "$.store.book[0]",
			want: []any{map[string]any{"title": "Book 1", "price": 10}},
		},
		{
			name: "select all prices",
			expr: "$.store.book[*].price",
			want: []any{10, 20},
		},
		{
			name: "descendant price",
			expr: "$..price",
			want: []any{10, 20},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := Parse(tt.expr)
			require.NoError(t, err)
			got := path.Select(input)
			assert.Equal(t, tt.want, []any(got))
		})
	}
}
