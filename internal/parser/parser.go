// Package parser provides a recursive descent parser for RFC 9535 JSONPath
// expressions. It consumes tokens from the lexer and produces an AST.
package parser

import (
	"errors"
	"fmt"
	"slices"
	"strconv"

	"github.com/agentable/jsonpath/internal/ast"
	"github.com/agentable/jsonpath/internal/lexer"
)

var (
	// ErrParseEnd is returned when a parse error occurs at the end of input.
	ErrParseEnd = errors.New("parse error at end")
	// ErrParsePosition is returned when a parse error occurs at a specific position.
	ErrParsePosition = errors.New("parse error at position")
	// ErrUnknownFunction is returned when an unknown function is referenced.
	ErrUnknownFunction = errors.New("unknown function")
	// ErrInvalidFunction is returned when a function is invalid.
	ErrInvalidFunction = errors.New("invalid function")
)

// Parser parses JSONPath expressions into AST nodes.
type Parser struct {
	src    string
	tokens []lexer.Token
	pos    int
	funcs  map[string]any // function registry for extensions
}

// New creates a new Parser for the given source string.
func New(src string, funcs map[string]any) (*Parser, error) {
	lex := lexer.New(src)
	// Pre-allocate tokens slice with estimated capacity based on source length
	// Typical JSONPath expressions have ~1 token per 3-4 characters
	tokens := make([]lexer.Token, 0, len(src)/3+1)
	for {
		tok := lex.Scan()
		tokens = append(tokens, tok)
		if tok.Kind == lexer.EOF || tok.Kind == lexer.Invalid {
			break
		}
	}

	// Check for lexer errors
	if len(tokens) > 0 && tokens[len(tokens)-1].Kind == lexer.Invalid {
		return nil, fmt.Errorf("%w: lexer error", tokens[len(tokens)-1].Err())
	}

	return &Parser{
		src:    src,
		tokens: tokens,
		pos:    0,
		funcs:  funcs,
	}, nil
}

// isBlankSpace reports whether b is RFC 9535 blank space (SP / HTAB / LF / CR).
func isBlankSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

// Parse parses a JSONPath query and returns the AST.
func (p *Parser) Parse() (*ast.PathQuery, error) {
	// RFC 9535 requires no leading/trailing whitespace
	if len(p.src) > 0 && isBlankSpace(p.src[0]) {
		return nil, fmt.Errorf("leading whitespace not allowed: %w", ErrParsePosition)
	}
	if len(p.src) > 0 && isBlankSpace(p.src[len(p.src)-1]) {
		return nil, fmt.Errorf("trailing whitespace not allowed: %w", ErrParsePosition)
	}

	// jsonpath-query = root-identifier segments
	if !p.match(lexer.Dollar) && !p.match(lexer.At) {
		return nil, p.error("expected $ or @")
	}

	isRoot := p.previous().Kind == lexer.Dollar

	segments, err := p.parseSegments()
	if err != nil {
		return nil, err
	}

	if !p.isAtEnd() {
		return nil, p.error("unexpected token after path")
	}

	return ast.NewPathQuery(isRoot, segments...), nil
}

// parseSegments parses zero or more segments.
func (p *Parser) parseSegments() ([]ast.Segment, error) {
	var segments []ast.Segment

	for !p.isAtEnd() {
		switch {
		case p.match(lexer.DotDot):
			// descendant segment
			sel, err := p.parseDescendantSegment()
			if err != nil {
				return nil, err
			}
			segments = append(segments, sel)
		case p.match(lexer.LeftBracket):
			// bracketed child segment
			sel, err := p.parseBracketedSelection()
			if err != nil {
				return nil, err
			}
			segments = append(segments, ast.Child(sel...))
		case p.match(lexer.Dot):
			// dot-child segment
			sel, err := p.parseDotChild()
			if err != nil {
				return nil, err
			}
			segments = append(segments, ast.Child(sel))
		default:
			return segments, nil
		}
	}

	return segments, nil
}

// parseDescendantSegment parses a descendant segment after "..".
func (p *Parser) parseDescendantSegment() (ast.Segment, error) {
	// RFC 9535: No whitespace allowed between .. and the following token
	dotDotToken := p.previous()
	if !p.isAtEnd() {
		nextToken := p.peek()
		if dotDotToken.End < nextToken.Start {
			return ast.Segment{}, p.error("whitespace not allowed after ..")
		}
	}

	switch {
	case p.match(lexer.LeftBracket):
		sel, err := p.parseBracketedSelection()
		if err != nil {
			return ast.Segment{}, err
		}
		return ast.Descendant(sel...), nil
	case p.match(lexer.Star):
		return ast.Descendant(ast.WildcardSelector()), nil
	case p.check(lexer.Ident) || p.check(lexer.True) || p.check(lexer.False) || p.check(lexer.Null):
		name := p.advance().Val(p.src)
		return ast.Descendant(ast.NameSelector(name)), nil
	default:
		return ast.Segment{}, p.error("expected [, *, or identifier after ..")
	}
}

// parseDotChild parses a dot-child selector (. followed by * or identifier).
func (p *Parser) parseDotChild() (ast.Selector, error) {
	// RFC 9535: No whitespace allowed between . and the following token
	dotToken := p.previous()
	if !p.isAtEnd() {
		nextToken := p.peek()
		if dotToken.End < nextToken.Start {
			return ast.Selector{}, p.error("whitespace not allowed after .")
		}
	}

	if p.match(lexer.Star) {
		return ast.WildcardSelector(), nil
	}
	// Accept identifiers and keywords (true, false, null) as member names
	if p.check(lexer.Ident) || p.check(lexer.True) || p.check(lexer.False) || p.check(lexer.Null) {
		name := p.advance().Val(p.src)
		return ast.NameSelector(name), nil
	}
	return ast.Selector{}, p.error("expected * or identifier after .")
}

// parseBracketedSelection parses selectors inside brackets.
func (p *Parser) parseBracketedSelection() ([]ast.Selector, error) {
	var selectors []ast.Selector

	for {
		sel, err := p.parseSelector()
		if err != nil {
			return nil, err
		}
		selectors = append(selectors, sel)

		if !p.match(lexer.Comma) {
			break
		}
	}

	if !p.match(lexer.RightBracket) {
		return nil, p.error("expected ] or ,")
	}

	return selectors, nil
}

// parseSelector parses a single selector.
func (p *Parser) parseSelector() (ast.Selector, error) {
	// wildcard
	if p.match(lexer.Star) {
		return ast.WildcardSelector(), nil
	}

	// filter
	if p.match(lexer.Question) {
		expr, err := p.parseFilterExpr()
		if err != nil {
			return ast.Selector{}, err
		}
		return ast.FilterSelector(expr), nil
	}

	// string (name selector)
	if p.check(lexer.String) {
		name := p.advance().Value
		return ast.NameSelector(name), nil
	}

	// integer or slice
	if p.check(lexer.Int) {
		return p.parseIndexOrSlice()
	}

	// slice starting with colon
	if p.match(lexer.Colon) {
		return p.parseSlice(0, false)
	}

	return ast.Selector{}, p.error("expected selector")
}

// parseFilterExpr parses a filter expression: logical-or-expr
func (p *Parser) parseFilterExpr() (*ast.FilterExpr, error) {
	or, err := p.parseLogicalOr()
	if err != nil {
		return nil, err
	}
	return &ast.FilterExpr{Or: or}, nil
}

// parseLogicalOr parses: logical-and-expr *( "||" logical-and-expr )
func (p *Parser) parseLogicalOr() (ast.LogicalOr, error) {
	var ands []ast.LogicalAnd

	and, err := p.parseLogicalAnd()
	if err != nil {
		return nil, err
	}
	ands = append(ands, and)

	for p.match(lexer.Or) {
		and, err := p.parseLogicalAnd()
		if err != nil {
			return nil, err
		}
		ands = append(ands, and)
	}

	return ands, nil
}

// parseLogicalAnd parses: basic-expr *( "&&" basic-expr )
func (p *Parser) parseLogicalAnd() (ast.LogicalAnd, error) {
	var exprs []ast.BasicExpr

	expr, err := p.parseBasicExpr()
	if err != nil {
		return nil, err
	}
	exprs = append(exprs, expr)

	for p.match(lexer.And) {
		expr, err := p.parseBasicExpr()
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, expr)
	}

	return exprs, nil
}

// parseBasicExpr parses: paren-expr / comparison-expr / test-expr
func (p *Parser) parseBasicExpr() (ast.BasicExpr, error) {
	// Negated expression: !( ... ) or !@.foo or !func()
	if p.match(lexer.Not) {
		if p.match(lexer.LeftParen) {
			or, err := p.parseLogicalOr()
			if err != nil {
				return nil, err
			}
			if !p.match(lexer.RightParen) {
				return nil, p.error("expected )")
			}
			return &ast.NotParenExpr{Expr: &or}, nil
		}
		// Negated function call: !match(...) or !search(...)
		if p.check(lexer.Ident) {
			funcExpr, err := p.parseFunctionExpr()
			if err != nil {
				return nil, err
			}
			fe, ok := funcExpr.(*ast.FuncExpr)
			if !ok {
				return nil, p.error("expected function expression")
			}
			if fe.Func().ResultType() != ast.Logical {
				return nil, p.error("only logical functions can be negated")
			}
			return &ast.NegFuncExpr{Func: fe}, nil
		}
		// Negated test expression: !@.foo or !$.foo
		query, err := p.parseFilterQuery()
		if err != nil {
			return nil, err
		}
		return &ast.NonExistExpr{Query: query}, nil
	}

	// Parenthesized expression: ( ... )
	if p.match(lexer.LeftParen) {
		or, err := p.parseLogicalOr()
		if err != nil {
			return nil, err
		}
		if !p.match(lexer.RightParen) {
			return nil, p.error("expected )")
		}
		return &ast.ParenExpr{Expr: &or}, nil
	}

	// Function call
	if p.check(lexer.Ident) {
		funcExpr, err := p.parseFunctionExpr()
		if err != nil {
			return nil, err
		}

		// Check if this is a comparison with a function on the left
		if p.checkCompOp() {
			fe, ok := funcExpr.(*ast.FuncExpr)
			if !ok {
				return nil, p.error("expected function expression")
			}
			// RFC 9535: logical function results cannot be used in comparisons
			if fe.Func().ResultType() == ast.Logical {
				return nil, p.error("logical function result cannot be compared")
			}
			op := p.parseCompOp()
			right, err := p.parseCompValue()
			if err != nil {
				return nil, err
			}
			return &ast.CompExpr{
				Left:  &ast.FuncValue{Func: fe},
				Op:    op,
				Right: right,
			}, nil
		}

		// Otherwise, it must be a logical function (returns bool)
		fe, ok := funcExpr.(*ast.FuncExpr)
		if !ok {
			return nil, p.error("expected function expression")
		}
		if fe.Func().ResultType() != ast.Logical {
			return nil, p.error("value function must be used in comparison")
		}
		return funcExpr, nil
	}

	// Test or comparison expression starting with @ or $
	if p.check(lexer.At) || p.check(lexer.Dollar) {
		return p.parseTestOrComparison()
	}

	// Literal comparison
	if p.check(lexer.String) || p.check(lexer.Int) || p.check(lexer.Number) ||
		p.check(lexer.True) || p.check(lexer.False) || p.check(lexer.Null) {
		return p.parseComparisonFromLiteral()
	}

	return nil, p.error("expected filter expression")
}

// parseTestOrComparison parses a test expression or comparison starting with @ or $
func (p *Parser) parseTestOrComparison() (ast.BasicExpr, error) {
	query, err := p.parseFilterQuery()
	if err != nil {
		return nil, err
	}

	// Check for comparison operator
	if p.checkCompOp() {
		// Queries in comparisons must be singular
		if !query.IsSingular() {
			return nil, p.error("non-singular query is not allowed in comparison")
		}

		op := p.parseCompOp()
		right, err := p.parseCompValue()
		if err != nil {
			return nil, err
		}
		return &ast.CompExpr{
			Left:  &ast.QueryValue{Query: query},
			Op:    op,
			Right: right,
		}, nil
	}

	// Otherwise it's an existence test
	return &ast.ExistExpr{Query: query}, nil
}

// parseComparisonFromLiteral parses a comparison starting with a literal
func (p *Parser) parseComparisonFromLiteral() (ast.BasicExpr, error) {
	left, err := p.parseLiteralValue()
	if err != nil {
		return nil, err
	}

	if !p.checkCompOp() {
		return nil, p.error("expected comparison operator")
	}

	op := p.parseCompOp()
	right, err := p.parseCompValue()
	if err != nil {
		return nil, err
	}

	return &ast.CompExpr{
		Left:  &ast.LiteralValue{Val: left},
		Op:    op,
		Right: right,
	}, nil
}

// parseFunctionExpr parses a function call
func (p *Parser) parseFunctionExpr() (ast.BasicExpr, error) {
	nameToken := p.advance()
	name := nameToken.Val(p.src)

	// RFC 9535: No whitespace allowed between function name and (
	if !p.isAtEnd() {
		nextToken := p.peek()
		if nameToken.End < nextToken.Start {
			return nil, p.error("whitespace not allowed between function name and (")
		}
	}

	if !p.match(lexer.LeftParen) {
		return nil, p.error("expected ( after function name")
	}

	// Parse arguments
	var args []any
	if !p.check(lexer.RightParen) {
		for {
			arg, err := p.parseFunctionArg()
			if err != nil {
				return nil, err
			}
			args = append(args, arg)

			if !p.match(lexer.Comma) {
				break
			}
		}
	}

	if !p.match(lexer.RightParen) {
		return nil, p.error("expected )")
	}

	// Look up function in registry
	fn, ok := p.funcs[name]
	if !ok {
		return nil, fmt.Errorf("%s: %w", name, ErrUnknownFunction)
	}

	funcObj, ok := fn.(ast.Function)
	if !ok {
		return nil, fmt.Errorf("%s: %w", name, ErrInvalidFunction)
	}

	// Determine argument types for validation
	argTypes := make([]ast.ArgType, len(args))
	for i, arg := range args {
		switch a := arg.(type) {
		case *ast.PathQuery:
			// Check if it's singular or not
			if a.IsSingular() {
				argTypes[i] = ast.QueryArg
			} else {
				argTypes[i] = ast.FilterArg
			}
		case *ast.FuncExpr:
			argTypes[i] = ast.FunctionArg
		default:
			// Literal value
			argTypes[i] = ast.Literal
		}
	}

	// Validate argument types
	if err := funcObj.Validate(argTypes); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}

	// Resolve QueryArg: determine if the function expects Nodes or Value for
	// each singular query argument. This affects evaluation behavior — when a
	// function expects NodesType, the node list must be passed as-is rather
	// than extracting the single value.
	for i, at := range argTypes {
		if at != ast.QueryArg {
			continue
		}
		// Test if the function would also accept FilterArg (NodesType) here.
		// If so, the parameter expects nodes — mark as FilterArg so the
		// evaluator passes the raw node list.
		probe := make([]ast.ArgType, len(argTypes))
		copy(probe, argTypes)
		probe[i] = ast.FilterArg
		if funcObj.Validate(probe) == nil {
			argTypes[i] = ast.FilterArg
		}
	}

	return ast.NewFuncExpr(funcObj, argTypes, args...), nil
}

// parseFunctionArg parses a function argument
func (p *Parser) parseFunctionArg() (any, error) {
	// Query argument
	if p.check(lexer.At) || p.check(lexer.Dollar) {
		return p.parseFilterQuery()
	}

	// Nested function call argument
	if p.check(lexer.Ident) {
		return p.parseFunctionExpr()
	}

	// Literal argument
	return p.parseLiteralValue()
}

// parseFilterQuery parses a query starting with @ or $
func (p *Parser) parseFilterQuery() (*ast.PathQuery, error) {
	if !p.match(lexer.Dollar) && !p.match(lexer.At) {
		return nil, p.error("expected $ or @")
	}

	isRoot := p.previous().Kind == lexer.Dollar

	segments, err := p.parseSegments()
	if err != nil {
		return nil, err
	}

	return ast.NewPathQuery(isRoot, segments...), nil
}

// parseCompValue parses a comparable value (literal, query, or function)
func (p *Parser) parseCompValue() (ast.CompValue, error) {
	// Function call
	if p.check(lexer.Ident) {
		funcExpr, err := p.parseFunctionExpr()
		if err != nil {
			return nil, err
		}
		fe, ok := funcExpr.(*ast.FuncExpr)
		if !ok {
			return nil, p.error("expected function expression")
		}
		// RFC 9535: logical function results cannot be used in comparisons
		if fe.Func().ResultType() == ast.Logical {
			return nil, p.error("logical function result cannot be compared")
		}
		return &ast.FuncValue{Func: fe}, nil
	}

	// Query
	if p.check(lexer.At) || p.check(lexer.Dollar) {
		query, err := p.parseFilterQuery()
		if err != nil {
			return nil, err
		}
		// Queries in comparisons must be singular
		if !query.IsSingular() {
			return nil, p.error("non-singular query is not allowed in comparison")
		}
		return &ast.QueryValue{Query: query}, nil
	}

	// Literal
	val, err := p.parseLiteralValue()
	if err != nil {
		return nil, err
	}
	return &ast.LiteralValue{Val: val}, nil
}

// parseLiteralValue parses a literal value
func (p *Parser) parseLiteralValue() (any, error) {
	if p.match(lexer.String) {
		return p.previous().Value, nil
	}
	if p.match(lexer.Int) {
		return strconv.ParseInt(p.previous().Val(p.src), 10, 64)
	}
	if p.match(lexer.Number) {
		return strconv.ParseFloat(p.previous().Val(p.src), 64)
	}
	if p.match(lexer.True) {
		return true, nil
	}
	if p.match(lexer.False) {
		return false, nil
	}
	if p.match(lexer.Null) {
		return ast.JSONNull(), nil
	}
	return nil, p.error("expected literal value")
}

// checkCompOp checks if the current token is a comparison operator
func (p *Parser) checkCompOp() bool {
	return p.check(lexer.Equal) || p.check(lexer.NotEqual) ||
		p.check(lexer.Less) || p.check(lexer.LessEqual) ||
		p.check(lexer.Greater) || p.check(lexer.GreaterEqual)
}

// parseCompOp parses a comparison operator
func (p *Parser) parseCompOp() ast.CompOp {
	if p.match(lexer.Equal) {
		return ast.Equal
	}
	if p.match(lexer.NotEqual) {
		return ast.NotEqual
	}
	if p.match(lexer.Less) {
		return ast.Less
	}
	if p.match(lexer.LessEqual) {
		return ast.LessEqual
	}
	if p.match(lexer.Greater) {
		return ast.Greater
	}
	if p.match(lexer.GreaterEqual) {
		return ast.GreaterEqual
	}
	return ast.Equal // shouldn't reach here
}

// parseIndexOrSlice parses an index or slice selector starting with an integer.
func (p *Parser) parseIndexOrSlice() (ast.Selector, error) {
	startTok := p.advance()
	start, err := strconv.ParseInt(startTok.Val(p.src), 10, 64)
	if err != nil {
		return ast.Selector{}, fmt.Errorf("%w: invalid integer", err)
	}

	// RFC 9535: -0 is not allowed as an index
	if start == 0 && startTok.Val(p.src)[0] == '-' {
		return ast.Selector{}, p.error("-0 is not allowed")
	}

	// RFC 9535: index values must be in [-(2^53-1), 2^53-1]
	const maxIndex = 9007199254740991 // 2^53 - 1
	if start < -maxIndex || start > maxIndex {
		return ast.Selector{}, p.error("index out of range")
	}

	if p.match(lexer.Colon) {
		return p.parseSlice(start, true)
	}

	return ast.IndexSelector(start), nil
}

// parseSlice parses a slice selector.
func (p *Parser) parseSlice(start int64, hasStart bool) (ast.Selector, error) {
	const maxIndex = 9007199254740991 // 2^53 - 1

	args := ast.SliceArgs{
		Start:    start,
		HasStart: hasStart,
	}

	// Parse end
	if p.check(lexer.Int) {
		endTok := p.advance()
		end, err := strconv.ParseInt(endTok.Val(p.src), 10, 64)
		if err != nil {
			return ast.Selector{}, fmt.Errorf("%w: invalid integer", err)
		}
		// RFC 9535: -0 is not allowed
		if end == 0 && endTok.Val(p.src)[0] == '-' {
			return ast.Selector{}, p.error("-0 is not allowed")
		}
		// RFC 9535: index values must be in [-(2^53-1), 2^53-1]
		if end < -maxIndex || end > maxIndex {
			return ast.Selector{}, p.error("index out of range")
		}
		args.End = end
		args.HasEnd = true
	}

	// Parse step
	if p.match(lexer.Colon) {
		if p.check(lexer.Int) {
			stepTok := p.advance()
			step, err := strconv.ParseInt(stepTok.Val(p.src), 10, 64)
			if err != nil {
				return ast.Selector{}, fmt.Errorf("%w: invalid integer", err)
			}
			// RFC 9535: -0 is not allowed
			if step == 0 && stepTok.Val(p.src)[0] == '-' {
				return ast.Selector{}, p.error("-0 is not allowed")
			}
			// RFC 9535: index values must be in [-(2^53-1), 2^53-1]
			if step < -maxIndex || step > maxIndex {
				return ast.Selector{}, p.error("index out of range")
			}
			args.Step = step
			args.HasStep = true
		}
	}

	return ast.SliceSelector(args), nil
}

// Token navigation helpers

func (p *Parser) match(kinds ...lexer.Kind) bool {
	if slices.ContainsFunc(kinds, p.check) {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) check(kind lexer.Kind) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().Kind == kind
}

func (p *Parser) advance() lexer.Token {
	if !p.isAtEnd() {
		p.pos++
	}
	return p.previous()
}

func (p *Parser) isAtEnd() bool {
	return p.pos >= len(p.tokens) || p.peek().Kind == lexer.EOF
}

func (p *Parser) peek() lexer.Token {
	if p.pos < len(p.tokens) {
		return p.tokens[p.pos]
	}
	return lexer.Token{Kind: lexer.EOF}
}

func (p *Parser) previous() lexer.Token {
	if p.pos > 0 && p.pos <= len(p.tokens) {
		return p.tokens[p.pos-1]
	}
	return lexer.Token{Kind: lexer.Invalid}
}

func (p *Parser) error(msg string) error {
	tok := p.peek()
	if tok.Kind == lexer.EOF {
		return fmt.Errorf("%s: %w", msg, ErrParseEnd)
	}
	return fmt.Errorf("%s at position %d: %w", msg, tok.Start, ErrParsePosition)
}
