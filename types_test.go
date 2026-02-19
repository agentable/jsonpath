package jsonpath

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNameElement_Normalized(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		elem NameElement
		norm string
		ptr  string
	}{
		{
			name: "simple",
			elem: NameElement("a"),
			norm: `['a']`,
			ptr:  `a`,
		},
		{
			name: "escape_apostrophes",
			elem: NameElement("'hi'"),
			norm: `['\'hi\'']`,
			ptr:  "'hi'",
		},
		{
			name: "escape_special",
			elem: NameElement("'\b\f\n\r\t\\'"),
			norm: `['\'\b\f\n\r\t\\\'']`,
			ptr:  "'\b\f\n\r\t\\'",
		},
		{
			name: "escape_vertical_tab",
			elem: NameElement("\u000B"),
			norm: `['\u000b']`,
			ptr:  "\u000B",
		},
		{
			name: "escape_null",
			elem: NameElement("\u0000"),
			norm: `['\u0000']`,
			ptr:  "\u0000",
		},
		{
			name: "escape_control_chars",
			elem: NameElement("\u0001\u0002\u0003\u0004\u0005\u0006\u0007\u000e\u000F"),
			norm: `['\u0001\u0002\u0003\u0004\u0005\u0006\u0007\u000e\u000f']`,
			ptr:  "\u0001\u0002\u0003\u0004\u0005\u0006\u0007\u000e\u000F",
		},
		{
			name: "escape_pointer_chars",
			elem: NameElement("this / ~that"),
			norm: `['this / ~that']`,
			ptr:  "this ~1 ~0that",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			p := NormalizedPath{tc.elem}
			// String includes the $ prefix, so strip it for element check.
			got := p.String()
			a.Equal("$"+tc.norm, got)

			// Check pointer output for single element.
			a.Equal("/"+tc.ptr, p.Pointer())
		})
	}
}

func TestIndexElement_Normalized(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		elem IndexElement
		norm string
		ptr  string
	}{
		{
			name: "zero",
			elem: IndexElement(0),
			norm: "[0]",
			ptr:  "0",
		},
		{
			name: "positive",
			elem: IndexElement(42),
			norm: "[42]",
			ptr:  "42",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)

			p := NormalizedPath{tc.elem}
			a.Equal("$"+tc.norm, p.String())
			a.Equal("/"+tc.ptr, p.Pointer())
		})
	}
}

func TestNormalizedPath_String(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		path NormalizedPath
		str  string
		ptr  string
	}{
		{
			name: "empty",
			path: NormalizedPath{},
			str:  "$",
			ptr:  "",
		},
		{
			name: "single_name",
			path: NormalizedPath{NameElement("a")},
			str:  "$['a']",
			ptr:  "/a",
		},
		{
			name: "single_index",
			path: NormalizedPath{IndexElement(1)},
			str:  "$[1]",
			ptr:  "/1",
		},
		{
			name: "nested",
			path: NormalizedPath{NameElement("a"), NameElement("b"), IndexElement(1)},
			str:  "$['a']['b'][1]",
			ptr:  "/a/b/1",
		},
		{
			name: "unicode_escape",
			path: NormalizedPath{NameElement("\u000B")},
			str:  `$['\u000b']`,
			ptr:  "/\u000b",
		},
		{
			name: "unicode_printable",
			path: NormalizedPath{NameElement("\u0061")},
			str:  "$['a']",
			ptr:  "/a",
		},
		{
			name: "pointer_escapes",
			path: NormalizedPath{NameElement("a~x"), NameElement("b/2"), IndexElement(1)},
			str:  "$['a~x']['b/2'][1]",
			ptr:  "/a~0x/b~12/1",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			a := assert.New(t)
			a.Equal(tc.str, tc.path.String())
			a.Equal(tc.ptr, tc.path.Pointer())
		})
	}
}

func TestNormalizedPath_Compare(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		p1   NormalizedPath
		p2   NormalizedPath
		exp  int
	}{
		{name: "empty", exp: 0},
		{
			name: "same_name",
			p1:   NormalizedPath{NameElement("a")},
			p2:   NormalizedPath{NameElement("a")},
			exp:  0,
		},
		{
			name: "diff_names",
			p1:   NormalizedPath{NameElement("a")},
			p2:   NormalizedPath{NameElement("b")},
			exp:  -1,
		},
		{
			name: "diff_names_rev",
			p1:   NormalizedPath{NameElement("b")},
			p2:   NormalizedPath{NameElement("a")},
			exp:  1,
		},
		{
			name: "longer_first",
			p1:   NormalizedPath{NameElement("a"), NameElement("b")},
			p2:   NormalizedPath{NameElement("a")},
			exp:  1,
		},
		{
			name: "shorter_first",
			p1:   NormalizedPath{NameElement("a")},
			p2:   NormalizedPath{NameElement("a"), NameElement("b")},
			exp:  -1,
		},
		{
			name: "name_vs_index",
			p1:   NormalizedPath{NameElement("a")},
			p2:   NormalizedPath{IndexElement(0)},
			exp:  1,
		},
		{
			name: "index_vs_name",
			p1:   NormalizedPath{IndexElement(0)},
			p2:   NormalizedPath{NameElement("a")},
			exp:  -1,
		},
		{
			name: "same_index",
			p1:   NormalizedPath{IndexElement(42)},
			p2:   NormalizedPath{IndexElement(42)},
			exp:  0,
		},
		{
			name: "diff_indexes",
			p1:   NormalizedPath{IndexElement(42)},
			p2:   NormalizedPath{IndexElement(99)},
			exp:  -1,
		},
		{
			name: "diff_indexes_rev",
			p1:   NormalizedPath{IndexElement(99)},
			p2:   NormalizedPath{IndexElement(42)},
			exp:  1,
		},
		{
			name: "nested_type_diff",
			p1:   NormalizedPath{NameElement("a"), IndexElement(1024)},
			p2:   NormalizedPath{NameElement("a"), NameElement("b")},
			exp:  -1,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.exp, tc.p1.Compare(tc.p2))
		})
	}
}

func TestNormalizedPath_MarshalText(t *testing.T) {
	t.Parallel()

	p := NormalizedPath{NameElement("a"), IndexElement(0)}
	text, err := p.MarshalText()
	assert.NoError(t, err)
	assert.Equal(t, "$['a'][0]", string(text))
}
