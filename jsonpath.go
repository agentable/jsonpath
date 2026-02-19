package jsonpath

import (
	"errors"
	"slices"

	"github.com/agentable/jsonpath/internal/ast"
	"github.com/go-json-experiment/json"
)

// Path is a compiled RFC 9535 JSONPath query. Safe for concurrent use.
type Path struct {
	query *ast.PathQuery
}

// Select returns all nodes matched by p in input.
// input must be the result of json.Unmarshal (any / []any / map[string]any)
// or a value produced by github.com/go-json-experiment/json.
func (p *Path) Select(input any) NodeList {
	if p.query == nil {
		return nil
	}
	res := []any{input}
	segments := p.query.Segments()
	for i := range segments {
		res = applySegment(&segments[i], res, input)
	}
	return NodeList(res)
}

// SelectLocated returns matched nodes paired with their normalized paths.
func (p *Path) SelectLocated(input any) LocatedNodeList {
	if p.query == nil {
		return nil
	}
	res := []*LocatedNode{{Value: input, Path: nil}}
	segments := p.query.Segments()
	for i := range segments {
		res = applySegmentLocated(&segments[i], res, input)
	}
	return LocatedNodeList(res)
}

// String returns the canonical string representation of p.
func (p *Path) String() string {
	if p.query == nil {
		return ""
	}
	return p.query.String()
}

// MarshalText implements encoding.TextMarshaler.
func (p *Path) MarshalText() ([]byte, error) {
	return []byte(p.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (p *Path) UnmarshalText(text []byte) error {
	path, err := Parse(string(text))
	if err != nil {
		return err
	}
	*p = *path
	return nil
}

// Parse compiles a JSONPath expression. Returns ErrPathParse on failure.
func Parse(expr string) (*Path, error) {
	p := NewParser()
	return p.Parse(expr)
}

// MustParse compiles a JSONPath expression. Panics on failure.
func MustParse(expr string) *Path {
	path, err := Parse(expr)
	if err != nil {
		panic(err)
	}
	return path
}

// Valid reports whether expr is a syntactically valid JSONPath expression.
func Valid(expr string) bool {
	_, err := Parse(expr)
	return err == nil
}

// QueryJSON unmarshals src and evaluates path against it.
// Uses github.com/go-json-experiment/json for unmarshaling.
func QueryJSON(src []byte, path *Path) (NodeList, error) {
	var v any
	if err := json.Unmarshal(src, &v, json.DefaultOptionsV2()); err != nil {
		return nil, errors.Join(ErrUnmarshal, err)
	}
	return path.Select(v), nil
}

// QueryJSONLocated is the located variant of QueryJSON.
func QueryJSONLocated(src []byte, path *Path) (LocatedNodeList, error) {
	var v any
	if err := json.Unmarshal(src, &v, json.DefaultOptionsV2()); err != nil {
		return nil, errors.Join(ErrUnmarshal, err)
	}
	return path.SelectLocated(v), nil
}

// extendPath creates a new path by appending elem to path.
// The original path is not modified.
func extendPath(path NormalizedPath, elem PathElement) NormalizedPath {
	return append(slices.Clone(path), elem)
}

// applySegment applies a segment to a list of nodes, returning the new node list.
func applySegment(seg *ast.Segment, nodes []any, root any) []any {
	if len(nodes) == 0 {
		return nodes
	}
	out := make([]any, 0, len(nodes))
	if seg.IsDescendant() {
		for _, n := range nodes {
			out = appendDescendant(out, seg, n, root)
		}
	} else {
		for _, n := range nodes {
			out = appendSelectors(out, seg.Selectors(), n, root)
		}
	}
	return out
}

// appendDescendant recursively applies selectors to node and all its descendants.
func appendDescendant(out []any, seg *ast.Segment, node, root any) []any {
	// Apply selectors to the current node
	out = appendSelectors(out, seg.Selectors(), node, root)

	// Recurse into children
	switch v := node.(type) {
	case map[string]any:
		for _, child := range v {
			out = appendDescendant(out, seg, child, root)
		}
	case []any:
		for _, child := range v {
			out = appendDescendant(out, seg, child, root)
		}
	}
	return out
}

// appendSelectors applies a list of selectors to node, appending matches to out.
func appendSelectors(out []any, selectors []ast.Selector, node, root any) []any {
	for i := range selectors {
		out = appendSelector(out, &selectors[i], node, root)
	}
	return out
}

// appendSelector applies a single selector to node, appending matches to out.
// Uses a switch on SelectorKind to keep the hot path in the instruction cache.
func appendSelector(out []any, sel *ast.Selector, node, root any) []any {
	switch sel.Kind {
	case ast.Name:
		if m, ok := node.(map[string]any); ok {
			if v, ok := m[sel.Name]; ok {
				out = append(out, v)
			}
		}
	case ast.Index:
		if arr, ok := node.([]any); ok {
			idx := normalizeIndex(sel.Index, len(arr))
			if idx >= 0 && idx < len(arr) {
				out = append(out, arr[idx])
			}
		}
	case ast.Slice:
		if arr, ok := node.([]any); ok {
			out = appendSlice(out, arr, sel.Slice)
		}
	case ast.Wildcard:
		switch v := node.(type) {
		case map[string]any:
			for _, val := range v {
				out = append(out, val)
			}
		case []any:
			out = append(out, v...)
		}
	case ast.Filter:
		switch v := node.(type) {
		case map[string]any:
			for _, val := range v {
				if sel.Filter.Eval(val, root) {
					out = append(out, val)
				}
			}
		case []any:
			for _, val := range v {
				if sel.Filter.Eval(val, root) {
					out = append(out, val)
				}
			}
		}
	}
	return out
}

// normalizeIndex converts a possibly-negative index to a non-negative index.
// Negative indices count from the end of the array.
// Returns -1 if the index is out of bounds.
func normalizeIndex(idx int64, length int) int {
	if idx < 0 {
		idx += int64(length)
	}
	if idx < 0 || idx >= int64(length) {
		return -1
	}
	return int(idx)
}

// sliceIndices calculates the indices to select for a slice operation.
// Returns a slice of indices in the order they should be selected.
func sliceIndices(args ast.SliceArgs, length int) []int {
	if length == 0 {
		return nil
	}

	step := int64(1)
	if args.HasStep {
		step = args.Step
	}
	if step == 0 {
		return nil
	}

	var start, end int64
	if step > 0 {
		start = 0
		if args.HasStart {
			start = args.Start
		}
		end = int64(length)
		if args.HasEnd {
			end = args.End
		}
	} else {
		start = int64(length - 1)
		if args.HasStart {
			start = args.Start
		}
		end = -int64(length) - 1
		if args.HasEnd {
			end = args.End
		}
	}

	start, end = normalizeSliceBounds(start, end, step, length)

	var indices []int
	if step > 0 {
		for i := start; i < end; i += step {
			if i >= 0 && i < int64(length) {
				indices = append(indices, int(i))
			}
		}
	} else {
		for i := start; i > end; i += step {
			if i >= 0 && i < int64(length) {
				indices = append(indices, int(i))
			}
		}
	}
	return indices
}

// appendSlice applies a slice selector to an array, appending selected elements to out.
func appendSlice(out []any, arr []any, args ast.SliceArgs) []any {
	for _, idx := range sliceIndices(args, len(arr)) {
		out = append(out, arr[idx])
	}
	return out
}

// normalizeSliceBounds normalizes start and end indices for slice operations
// according to RFC 9535 §2.3.4. Handles negative indices and out-of-bounds
// values based on the step direction.
func normalizeSliceBounds(start, end, step int64, length int) (int64, int64) {
	// Normalize start
	if start < 0 {
		start += int64(length)
		if start < 0 {
			if step > 0 {
				start = 0
			}
		}
	} else if start >= int64(length) {
		if step < 0 {
			start = int64(length - 1)
		}
	}

	// Normalize end
	if end < 0 {
		end += int64(length)
		if end < 0 && step < 0 {
			end = -1
		}
	} else if end > int64(length) {
		end = int64(length)
	}

	return start, end
}

// applySegmentLocated applies a segment to a list of located nodes, returning the new located node list.
func applySegmentLocated(seg *ast.Segment, nodes []*LocatedNode, root any) []*LocatedNode {
	if len(nodes) == 0 {
		return nodes
	}
	out := make([]*LocatedNode, 0, len(nodes))
	if seg.IsDescendant() {
		for _, n := range nodes {
			out = appendDescendantLocated(out, seg, n.Value, n.Path, root)
		}
	} else {
		for _, n := range nodes {
			out = appendSelectorsLocated(out, seg.Selectors(), n.Value, n.Path, root)
		}
	}
	return out
}

// appendDescendantLocated recursively applies selectors to node and all its descendants.
func appendDescendantLocated(out []*LocatedNode, seg *ast.Segment, node any, path NormalizedPath, root any) []*LocatedNode {
	// Apply selectors to the current node
	out = appendSelectorsLocated(out, seg.Selectors(), node, path, root)

	// Recurse into children
	switch v := node.(type) {
	case map[string]any:
		for key, child := range v {
			out = appendDescendantLocated(out, seg, child, extendPath(path, NameElement(key)), root)
		}
	case []any:
		for idx, child := range v {
			out = appendDescendantLocated(out, seg, child, extendPath(path, IndexElement(idx)), root)
		}
	}
	return out
}

// appendSelectorsLocated applies a list of selectors to node, appending matches to out.
func appendSelectorsLocated(out []*LocatedNode, selectors []ast.Selector, node any, path NormalizedPath, root any) []*LocatedNode {
	for i := range selectors {
		out = appendSelectorLocated(out, &selectors[i], node, path, root)
	}
	return out
}

// appendSelectorLocated applies a single selector to node, appending matches to out.
func appendSelectorLocated(out []*LocatedNode, sel *ast.Selector, node any, path NormalizedPath, root any) []*LocatedNode {
	switch sel.Kind {
	case ast.Name:
		if m, ok := node.(map[string]any); ok {
			if v, ok := m[sel.Name]; ok {
				out = append(out, &LocatedNode{Value: v, Path: extendPath(path, NameElement(sel.Name))})
			}
		}
	case ast.Index:
		if arr, ok := node.([]any); ok {
			idx := normalizeIndex(sel.Index, len(arr))
			if idx >= 0 && idx < len(arr) {
				out = append(out, &LocatedNode{Value: arr[idx], Path: extendPath(path, IndexElement(idx))})
			}
		}
	case ast.Slice:
		if arr, ok := node.([]any); ok {
			out = appendSliceLocated(out, arr, path, sel.Slice)
		}
	case ast.Wildcard:
		switch v := node.(type) {
		case map[string]any:
			for key, val := range v {
				out = append(out, &LocatedNode{Value: val, Path: extendPath(path, NameElement(key))})
			}
		case []any:
			for idx, val := range v {
				out = append(out, &LocatedNode{Value: val, Path: extendPath(path, IndexElement(idx))})
			}
		}
	case ast.Filter:
		switch v := node.(type) {
		case map[string]any:
			for key, val := range v {
				if sel.Filter.Eval(val, root) {
					out = append(out, &LocatedNode{Value: val, Path: extendPath(path, NameElement(key))})
				}
			}
		case []any:
			for idx, val := range v {
				if sel.Filter.Eval(val, root) {
					out = append(out, &LocatedNode{Value: val, Path: extendPath(path, IndexElement(idx))})
				}
			}
		}
	}
	return out
}

// appendSliceLocated applies a slice selector to an array, appending selected elements with paths to out.
func appendSliceLocated(out []*LocatedNode, arr []any, path NormalizedPath, args ast.SliceArgs) []*LocatedNode {
	for _, idx := range sliceIndices(args, len(arr)) {
		out = append(out, &LocatedNode{Value: arr[idx], Path: extendPath(path, IndexElement(idx))})
	}
	return out
}
