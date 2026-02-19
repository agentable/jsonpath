package ast

// FilterExpr represents a filter expression tree (?logical-expr) per RFC 9535 §2.3.5.
type FilterExpr struct {
	Or LogicalOr
}

// Eval evaluates the filter expression against the current node.
func (f *FilterExpr) Eval(current, root any) bool {
	return f.Or.Eval(current, root)
}

// LogicalOr is a sequence of LogicalAnd expressions joined by ||.
// Short-circuits on first true.
type LogicalOr []LogicalAnd

// Eval returns true if any LogicalAnd expression is true.
func (lo LogicalOr) Eval(current, root any) bool {
	for i := range lo {
		if lo[i].Eval(current, root) {
			return true
		}
	}
	return false
}

// LogicalAnd is a sequence of BasicExpr joined by &&.
// Short-circuits on first false.
type LogicalAnd []BasicExpr

// Eval returns true if all BasicExpr are true.
func (la LogicalAnd) Eval(current, root any) bool {
	for i := range la {
		if !la[i].Eval(current, root) {
			return false
		}
	}
	return true
}

// BasicExpr is a filter expression that evaluates to a boolean.
type BasicExpr interface {
	Eval(current, root any) bool
}

// ExistExpr tests if a query selects at least one node.
type ExistExpr struct {
	Query *PathQuery
}

// Eval returns true if the query selects at least one node.
func (e *ExistExpr) Eval(current, root any) bool {
	// Special case: bare @ or $ with no segments always exists
	if len(e.Query.Segments()) == 0 {
		return true
	}
	nodes := e.Query.Select(current, root)
	return len(nodes) > 0
}

// NonExistExpr tests if a query selects no nodes.
type NonExistExpr struct {
	Query *PathQuery
}

// Eval returns true if the query selects no nodes.
func (e *NonExistExpr) Eval(current, root any) bool {
	// Special case: bare @ or $ with no segments always exists, so negation is false
	if len(e.Query.Segments()) == 0 {
		return false
	}
	nodes := e.Query.Select(current, root)
	return len(nodes) == 0
}

// ParenExpr is a parenthesized logical expression.
type ParenExpr struct {
	Expr *LogicalOr
}

// Eval evaluates the parenthesized expression.
func (p *ParenExpr) Eval(current, root any) bool {
	return p.Expr.Eval(current, root)
}

// NotParenExpr is a negated parenthesized logical expression.
type NotParenExpr struct {
	Expr *LogicalOr
}

// Eval evaluates the negated parenthesized expression.
func (n *NotParenExpr) Eval(current, root any) bool {
	return !n.Expr.Eval(current, root)
}

// NegFuncExpr is a negated logical function call expression (!match(), !search()).
type NegFuncExpr struct {
	Func *FuncExpr
}

// Eval evaluates the negated function call.
func (n *NegFuncExpr) Eval(current, root any) bool {
	return !n.Func.Eval(current, root)
}

// CompOp is a comparison operator.
type CompOp uint8

const (
	Equal        CompOp = iota // ==
	NotEqual                   // !=
	Less                       // <
	LessEqual                  // <=
	Greater                    // >
	GreaterEqual               // >=
)

// CompExpr is a comparison expression.
type CompExpr struct {
	Left  CompValue
	Op    CompOp
	Right CompValue
}

// Eval evaluates the comparison expression.
func (c *CompExpr) Eval(current, root any) bool {
	left := c.Left.Value(current, root)
	right := c.Right.Value(current, root)

	switch c.Op {
	case Equal:
		return equalTo(left, right)
	case NotEqual:
		return !equalTo(left, right)
	case Less:
		return sameType(left, right) && lessThan(left, right)
	case LessEqual:
		return sameType(left, right) && (lessThan(left, right) || equalTo(left, right))
	case Greater:
		return sameType(left, right) && !lessThan(left, right) && !equalTo(left, right)
	case GreaterEqual:
		return sameType(left, right) && !lessThan(left, right)
	}
	return false
}

// CompValue represents a comparable value in a comparison expression.
type CompValue interface {
	Value(current, root any) any
}

// LiteralValue is a literal value (string, number, bool, null).
type LiteralValue struct {
	Val any
}

// Value returns the literal value.
func (l *LiteralValue) Value(current, root any) any {
	return l.Val
}

// QueryValue is a singular query that produces a single value.
type QueryValue struct {
	Query *PathQuery
}

// Value returns the first value selected by the query, or a special "nothing" sentinel if none.
// We use a private sentinel type to distinguish "no value" from "null value".
func (q *QueryValue) Value(current, root any) any {
	nodes := q.Query.Select(current, root)
	if len(nodes) != 1 {
		return nothing{}
	}
	return nodes[0]
}

// nothing is a sentinel type representing "no value" (distinct from nil/null).
type nothing struct{}

// jsonNull is a sentinel type representing a literal JSON null value.
type jsonNull struct{}

// JSONNull returns a sentinel value representing a literal JSON null.
func JSONNull() jsonNull {
	return jsonNull{}
}

// FuncValue is a function call that produces a value.
type FuncValue struct {
	Func *FuncExpr
}

// Value returns the result of the function call.
func (f *FuncValue) Value(current, root any) any {
	return f.Func.Call(current, root)
}

// sameType returns true if both values have compatible types for ordering comparison.
func sameType(a, b any) bool {
	// If either value is "nothing", they're not comparable
	if _, ok := a.(nothing); ok {
		return false
	}
	if _, ok := b.(nothing); ok {
		return false
	}

	_, aIsJSONNull := a.(jsonNull)
	_, bIsJSONNull := b.(jsonNull)

	// JSON null and Go nil are the same for comparison
	aIsNull := aIsJSONNull || a == nil
	bIsNull := bIsJSONNull || b == nil

	// Nulls are only comparable to other nulls for ordering
	if aIsNull || bIsNull {
		return aIsNull && bIsNull
	}

	// Numeric types are compatible
	if isNumeric(a) && isNumeric(b) {
		return true
	}

	// Otherwise, types must match exactly
	switch a.(type) {
	case string:
		_, ok := b.(string)
		return ok
	case bool:
		_, ok := b.(bool)
		return ok
	default:
		return false
	}
}

// isNumeric returns true if v is a numeric type.
func isNumeric(v any) bool {
	switch v.(type) {
	case int, int8, int16, int32, int64:
		return true
	case uint, uint8, uint16, uint32, uint64:
		return true
	case float32, float64:
		return true
	default:
		return false
	}
}

// equalTo returns true if a equals b, with numeric type coercion and deep equality.
func equalTo(a, b any) bool {
	_, aIsNothing := a.(nothing)
	_, bIsNothing := b.(nothing)
	_, aIsJSONNull := a.(jsonNull)
	_, bIsJSONNull := b.(jsonNull)

	// Treat nothing and nil as the same "no value" sentinel
	aIsNoValue := aIsNothing || (a == nil && !aIsJSONNull && !bIsJSONNull)
	bIsNoValue := bIsNothing || (b == nil && !aIsJSONNull && !bIsJSONNull)

	if aIsNoValue && bIsNoValue {
		return true
	}
	if aIsNoValue || bIsNoValue {
		return false
	}

	// JSON null (literal) equals Go nil (from document)
	if (aIsJSONNull && b == nil) || (a == nil && bIsJSONNull) || (aIsJSONNull && bIsJSONNull) {
		return true
	}
	if aIsJSONNull || bIsJSONNull {
		return false
	}

	// Numeric comparison with coercion
	if isNumeric(a) && isNumeric(b) {
		return toFloat64(a) == toFloat64(b)
	}

	// Deep equality for arrays
	aArr, aIsArr := a.([]any)
	bArr, bIsArr := b.([]any)
	if aIsArr && bIsArr {
		if len(aArr) != len(bArr) {
			return false
		}
		for i := range aArr {
			if !equalTo(aArr[i], bArr[i]) {
				return false
			}
		}
		return true
	}

	// Deep equality for objects
	aObj, aIsObj := a.(map[string]any)
	bObj, bIsObj := b.(map[string]any)
	if aIsObj && bIsObj {
		if len(aObj) != len(bObj) {
			return false
		}
		for k, v := range aObj {
			bv, ok := bObj[k]
			if !ok || !equalTo(v, bv) {
				return false
			}
		}
		return true
	}

	// If one is array/object and the other isn't, they're not equal
	if aIsArr || bIsArr || aIsObj || bIsObj {
		return false
	}

	// Direct comparison for other types (string, bool)
	return a == b
}

// lessThan returns true if a < b. Assumes sameType(a, b) is true.
func lessThan(a, b any) bool {
	if a == nil || b == nil {
		return false
	}

	// Numeric comparison
	if isNumeric(a) && isNumeric(b) {
		return toFloat64(a) < toFloat64(b)
	}

	// String comparison
	if sa, ok := a.(string); ok {
		if sb, ok := b.(string); ok {
			return sa < sb
		}
	}

	return false
}

// toFloat64 converts a numeric value to float64.
func toFloat64(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int8:
		return float64(n)
	case int16:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case uint:
		return float64(n)
	case uint8:
		return float64(n)
	case uint16:
		return float64(n)
	case uint32:
		return float64(n)
	case uint64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	default:
		return 0
	}
}
