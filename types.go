package jsonpath

import (
	"cmp"
	"errors"
	"fmt"
	"iter"
	"slices"
	"strconv"
	"strings"
)

// Sentinel errors.
var (
	// ErrPathParse is returned when a JSONPath expression cannot be parsed.
	ErrPathParse = errors.New("jsonpath: parse error")
	// ErrFunction is returned when a JSONPath function call fails.
	ErrFunction = errors.New("jsonpath: function error")
	// ErrUnmarshal is returned when JSON unmarshaling fails in QueryJSON functions.
	ErrUnmarshal = errors.New("jsonpath: unmarshal error")
)

// PathElement is either a Name (string key) or an Index (array index)
// in a normalized path. Implemented by [NameElement] and [IndexElement].
type PathElement interface {
	pathElement()
	// writeNormalizedTo writes the element formatted as a normalized path
	// element to buf.
	writeNormalizedTo(buf *strings.Builder)
	// writePointerTo writes the element formatted as an RFC 6901 JSON Pointer
	// reference token to buf.
	writePointerTo(buf *strings.Builder)
}

// NameElement is a string key in a normalized path.
type NameElement string

func (NameElement) pathElement() {}

// writeNormalizedTo writes n to buf as ['name'] with proper escaping per
// RFC 9535 §2.7.
func (n NameElement) writeNormalizedTo(buf *strings.Builder) {
	buf.WriteString("['")
	for _, r := range string(n) {
		switch r {
		case '\b':
			buf.WriteString(`\b`)
		case '\f':
			buf.WriteString(`\f`)
		case '\n':
			buf.WriteString(`\n`)
		case '\r':
			buf.WriteString(`\r`)
		case '\t':
			buf.WriteString(`\t`)
		case '\'':
			buf.WriteString(`\'`)
		case '\\':
			buf.WriteString(`\\`)
		case '\x00', '\x01', '\x02', '\x03', '\x04', '\x05', '\x06', '\x07',
			'\x0b', '\x0e', '\x0f':
			fmt.Fprintf(buf, `\u000%x`, r)
		default:
			buf.WriteRune(r)
		}
	}
	buf.WriteString("']")
}

// writePointerTo writes n to buf as an RFC 6901 JSON Pointer reference token,
// escaping ~ as ~0 and / as ~1.
func (n NameElement) writePointerTo(buf *strings.Builder) {
	s := strings.ReplaceAll(string(n), "~", "~0")
	s = strings.ReplaceAll(s, "/", "~1")
	buf.WriteString(s)
}

// IndexElement is an array index in a normalized path.
type IndexElement int

func (IndexElement) pathElement() {}

// writeNormalizedTo writes i to buf as [N].
func (i IndexElement) writeNormalizedTo(buf *strings.Builder) {
	buf.WriteByte('[')
	buf.WriteString(strconv.Itoa(int(i)))
	buf.WriteByte(']')
}

// writePointerTo writes i to buf as its decimal string.
func (i IndexElement) writePointerTo(buf *strings.Builder) {
	buf.WriteString(strconv.Itoa(int(i)))
}

// NormalizedPath is a sequence of Name/Index selectors per RFC 9535 §2.7.
type NormalizedPath []PathElement

// String returns the normalized path string, e.g. $['a'][0].
func (p NormalizedPath) String() string {
	var buf strings.Builder
	buf.WriteByte('$')
	for _, e := range p {
		e.writeNormalizedTo(&buf)
	}
	return buf.String()
}

// Pointer returns an RFC 6901 JSON Pointer string, e.g. /a/0.
func (p NormalizedPath) Pointer() string {
	var buf strings.Builder
	for _, e := range p {
		buf.WriteByte('/')
		e.writePointerTo(&buf)
	}
	return buf.String()
}

// Compare compares p to q and returns -1, 0, or 1. Indexes are always
// considered less than names.
func (p NormalizedPath) Compare(q NormalizedPath) int {
	minLen := min(len(p), len(q))

	for i := range minLen {
		v1, isName1 := p[i].(NameElement)
		v2, isName2 := q[i].(NameElement)

		if isName1 && isName2 {
			if x := cmp.Compare(string(v1), string(v2)); x != 0 {
				return x
			}
			continue
		}

		if isName1 {
			return 1 // name > index
		}

		if isName2 {
			return -1 // index < name
		}

		// Both are IndexElement
		idx1 := p[i].(IndexElement)
		idx2 := q[i].(IndexElement)
		if x := cmp.Compare(int(idx1), int(idx2)); x != 0 {
			return x
		}
	}

	return cmp.Compare(len(p), len(q))
}

// MarshalText marshals p into its normalized path string. Implements
// [encoding.TextMarshaler].
func (p NormalizedPath) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

// LocatedNode pairs a value with the [NormalizedPath] for its location within
// a JSON query argument.
type LocatedNode struct {
	Value any
	Path  NormalizedPath
}

// NodeList is a list of nodes selected by a JSONPath query. Each node
// represents a single JSON value selected from the JSON query argument.
type NodeList []any

// All returns an iterator over all the nodes in list.
func (l NodeList) All() iter.Seq[any] {
	return slices.Values(l)
}

// LocatedNodeList is a list of nodes selected by a JSONPath query, along with
// their [NormalizedPath] locations.
type LocatedNodeList []*LocatedNode

// All returns an iterator over all the located nodes in list.
func (l LocatedNodeList) All() iter.Seq[*LocatedNode] {
	return slices.Values(l)
}

// Values returns an iterator over all the node values in list.
func (l LocatedNodeList) Values() iter.Seq[any] {
	return func(yield func(any) bool) {
		for _, n := range l {
			if !yield(n.Value) {
				return
			}
		}
	}
}

// Paths returns an iterator over all the [NormalizedPath] values in list.
func (l LocatedNodeList) Paths() iter.Seq[NormalizedPath] {
	return func(yield func(NormalizedPath) bool) {
		for _, n := range l {
			if !yield(n.Path) {
				return
			}
		}
	}
}

// Deduplicate deduplicates the nodes in list based on their [NormalizedPath]
// values, modifying the contents of list. It returns the modified list, which
// may have a shorter length, and zeroes the elements between the new length
// and the original length.
func (l LocatedNodeList) Deduplicate() LocatedNodeList {
	if len(l) <= 1 {
		return l
	}

	seen := make(map[string]struct{}, len(l))
	uniq := l[:0]
	for _, n := range l {
		p := n.Path.String()
		if _, exists := seen[p]; !exists {
			seen[p] = struct{}{}
			uniq = append(uniq, n)
		}
	}
	clear(l[len(uniq):])
	return slices.Clip(uniq)
}

// Sort sorts list by the [NormalizedPath] of each node.
func (l LocatedNodeList) Sort() {
	slices.SortFunc(l, func(a, b *LocatedNode) int {
		return a.Path.Compare(b.Path)
	})
}
