package ast

import "strings"

// PathQuery is the root of a compiled JSONPath expression. It holds a sequence
// of segments and whether the query is rooted ($) or relative (@).
type PathQuery struct {
	segments []Segment
	root     bool
}

// NewPathQuery creates a [PathQuery]. When root is true it indicates a
// root-identifier ($) query; when false it indicates a current-node (@) query
// used in filter sub-expressions.
func NewPathQuery(root bool, segments ...Segment) *PathQuery {
	return &PathQuery{root: root, segments: segments}
}

// Segments returns the query's segments.
func (q *PathQuery) Segments() []Segment { return q.segments }

// IsRoot reports whether the query starts from the root ($).
func (q *PathQuery) IsRoot() bool { return q.root }

// IsSingular reports whether the query always selects at most one node.
// A query is singular when every segment is singular (child segment with
// exactly one name or index selector) and no segment is a descendant segment.
func (q *PathQuery) IsSingular() bool {
	for i := range q.segments {
		if q.segments[i].IsDescendant() {
			return false
		}
		if !q.segments[i].IsSingular() {
			return false
		}
	}
	return true
}

// Singular returns the [SingularQuery] variant of q if q is a singular query,
// or nil otherwise.
func (q *PathQuery) Singular() *SingularQuery {
	if !q.IsSingular() {
		return nil
	}
	sels := make([]Selector, len(q.segments))
	for i := range q.segments {
		sels[i] = q.segments[i].Selectors()[0]
	}
	return &SingularQuery{selectors: sels, relative: !q.root}
}

// writeTo writes the canonical string representation of q to buf.
func (q *PathQuery) writeTo(buf *strings.Builder) {
	if q.root {
		buf.WriteByte('$')
	} else {
		buf.WriteByte('@')
	}
	for i := range q.segments {
		q.segments[i].writeTo(buf)
	}
}

// String returns the canonical string representation of the query,
// e.g. $["a"][0] or @["name"].
func (q *PathQuery) String() string {
	var buf strings.Builder
	q.writeTo(&buf)
	return buf.String()
}

// Select evaluates the query against the given current and root nodes.
// For root queries ($), it evaluates against root. For relative queries (@),
// it evaluates against current.
func (q *PathQuery) Select(current, root any) []any {
	start := root
	if !q.root {
		start = current
	}

	result := []any{start}
	for i := range q.segments {
		result = q.segments[i].Apply(result, root)
	}
	return result
}

// SingularQuery is a JSONPath query that is guaranteed to select at most one
// node. It is composed of a flat list of name/index selectors extracted from
// singular segments. Per RFC 9535, singular queries can be used as comparison
// operands and as arguments to the value() function.
type SingularQuery struct {
	selectors []Selector
	relative  bool // true for @ (current-node), false for $ (root)
}

// NewSingularQuery creates a [SingularQuery]. When relative is true, the query
// starts from the current node (@); otherwise from the root ($).
func NewSingularQuery(relative bool, selectors ...Selector) *SingularQuery {
	return &SingularQuery{selectors: selectors, relative: relative}
}

// Selectors returns the singular query's selectors.
func (sq *SingularQuery) Selectors() []Selector { return sq.selectors }

// IsRelative reports whether the query is relative (@) rather than rooted ($).
func (sq *SingularQuery) IsRelative() bool { return sq.relative }

// writeTo writes the canonical string representation to buf.
func (sq *SingularQuery) writeTo(buf *strings.Builder) {
	if sq.relative {
		buf.WriteByte('@')
	} else {
		buf.WriteByte('$')
	}
	for i := range sq.selectors {
		buf.WriteByte('[')
		sq.selectors[i].writeTo(buf)
		buf.WriteByte(']')
	}
}

// String returns the canonical string representation of the singular query.
func (sq *SingularQuery) String() string {
	var buf strings.Builder
	sq.writeTo(&buf)
	return buf.String()
}
