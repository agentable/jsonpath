package ast

import (
	"strconv"
	"strings"
)

// SelectorKind identifies the variant stored in a [Selector].
type SelectorKind uint8

const (
	Name     SelectorKind = iota // member name selector
	Index                        // array index selector
	Slice                        // array slice selector
	Wildcard                     // wildcard selector
	Filter                       // filter selector
)

// Selector is a tagged union representing one of the five RFC 9535 selector
// types. Using a concrete struct (instead of an interface) keeps selector
// slices contiguous in memory for cache efficiency.
type Selector struct {
	Kind   SelectorKind
	Name   string     // KindName: the member name
	Index  int64      // KindIndex: the array index (may be negative)
	Slice  SliceArgs  // KindSlice
	Filter *FilterExpr // KindFilter
}

// SliceArgs holds the optional start, end, step for a slice selector.
type SliceArgs struct {
	Start    int64
	End      int64
	Step     int64
	HasStart bool
	HasEnd   bool
	HasStep  bool
}

// NameSelector returns a Selector for a member name.
func NameSelector(name string) Selector {
	return Selector{Kind: Name, Name: name}
}

// IndexSelector returns a Selector for an array index.
func IndexSelector(idx int64) Selector {
	return Selector{Kind: Index, Index: idx}
}

// SliceSelector returns a Selector for an array slice.
func SliceSelector(args SliceArgs) Selector {
	return Selector{Kind: Slice, Slice: args}
}

// WildcardSelector returns a wildcard Selector.
func WildcardSelector() Selector {
	return Selector{Kind: Wildcard}
}

// FilterSelector returns a filter Selector.
func FilterSelector(expr *FilterExpr) Selector {
	return Selector{Kind: Filter, Filter: expr}
}

// IsSingular reports whether the selector can select at most one node.
// Only name and index selectors are singular.
func (s *Selector) IsSingular() bool {
	return s.Kind == Name || s.Kind == Index
}

// writeTo writes the canonical string representation of s to buf.
func (s *Selector) writeTo(buf *strings.Builder) {
	switch s.Kind {
	case Name:
		buf.WriteString(strconv.Quote(s.Name))
	case Index:
		buf.WriteString(strconv.FormatInt(s.Index, 10))
	case Slice:
		s.Slice.writeTo(buf)
	case Wildcard:
		buf.WriteByte('*')
	case Filter:
		buf.WriteString("?")
		// Full filter expression serialization will be added with filter support.
	}
}

// String returns the canonical string representation of s.
func (s *Selector) String() string {
	var buf strings.Builder
	s.writeTo(&buf)
	return buf.String()
}

// Apply applies the selector to a node and appends matching results to out.
func (s *Selector) Apply(out []any, node, root any) []any {
	switch s.Kind {
	case Name:
		if m, ok := node.(map[string]any); ok {
			if v, ok := m[s.Name]; ok {
				out = append(out, v)
			}
		}
	case Index:
		if arr, ok := node.([]any); ok {
			idx := s.Index
			if idx < 0 {
				idx += int64(len(arr))
			}
			if idx >= 0 && idx < int64(len(arr)) {
				out = append(out, arr[idx])
			}
		}
	case Slice:
		if arr, ok := node.([]any); ok {
			out = s.applySlice(out, arr)
		}
	case Wildcard:
		switch n := node.(type) {
		case map[string]any:
			for _, v := range n {
				out = append(out, v)
			}
		case []any:
			out = append(out, n...)
		}
	case Filter:
		switch n := node.(type) {
		case map[string]any:
			for _, v := range n {
				if s.Filter.Eval(v, root) {
					out = append(out, v)
				}
			}
		case []any:
			for _, v := range n {
				if s.Filter.Eval(v, root) {
					out = append(out, v)
				}
			}
		}
	}
	return out
}

// applySlice applies a slice selector to an array.
func (s *Selector) applySlice(out []any, arr []any) []any {
	length := int64(len(arr))
	if length == 0 {
		return out
	}

	// Normalize start, end, step
	start := s.Slice.Start
	end := s.Slice.End
	step := s.Slice.Step

	if !s.Slice.HasStep {
		step = 1
	}

	// Set defaults based on step direction
	switch {
	case step > 0:
		if !s.Slice.HasStart {
			start = 0
		}
		if !s.Slice.HasEnd {
			end = length
		}
	case step < 0:
		if !s.Slice.HasStart {
			start = length - 1
		}
		if !s.Slice.HasEnd {
			end = -length - 1
		}
	default:
		// step == 0
		return out
	}

	// Handle negative indices
	if start < 0 {
		start += length
	}
	if end < 0 {
		end += length
	}

	// Clamp to array bounds
	if step > 0 {
		if start < 0 {
			start = 0
		}
		if start > length {
			start = length
		}
		if end < 0 {
			end = 0
		}
		if end > length {
			end = length
		}
	} else {
		// For negative step, allow end to be -1 (before first element)
		if start < 0 {
			start = 0
		}
		if start >= length {
			start = length - 1
		}
		if end < -1 {
			end = -1
		}
		if end >= length {
			end = length - 1
		}
	}

	// Iterate based on step direction
	if step > 0 {
		for i := start; i < end; i += step {
			out = append(out, arr[i])
		}
	} else {
		for i := start; i > end; i += step {
			out = append(out, arr[i])
		}
	}

	return out
}

// writeTo writes the canonical slice notation (e.g. "1:5:2") to buf.
func (a *SliceArgs) writeTo(buf *strings.Builder) {
	if a.HasStart {
		buf.WriteString(strconv.FormatInt(a.Start, 10))
	}
	buf.WriteByte(':')
	if a.HasEnd {
		buf.WriteString(strconv.FormatInt(a.End, 10))
	}
	if a.HasStep {
		buf.WriteByte(':')
		buf.WriteString(strconv.FormatInt(a.Step, 10))
	}
}
