// Package scim implements SCIM 2.0 filter parsing per RFC 7644 Section 3.4.2.2.
// This file provides a lexer, parser, and evaluator for SCIM filter expressions.
package scim

import (
	"fmt"
	"strings"
	"time"
)

// --- AST Nodes ---

// FilterExpr is the abstract syntax tree for a parsed SCIM filter.
type FilterExpr interface {
	// Evaluate returns true if the given attribute map matches this filter.
	Evaluate(attrs map[string]any) bool
	String() string
}

// ComparisonOp represents a SCIM comparison operator.
type ComparisonOp string

const (
	OpEq ComparisonOp = "eq"
	OpNe ComparisonOp = "ne"
	OpCo ComparisonOp = "co"
	OpSw ComparisonOp = "sw"
	OpEw ComparisonOp = "ew"
	OpPr ComparisonOp = "pr"
	OpGt ComparisonOp = "gt"
	OpGe ComparisonOp = "ge"
	OpLt ComparisonOp = "lt"
	OpLe ComparisonOp = "le"
)

// AttrExpression is a leaf comparison (e.g., userName eq "john").
type AttrExpression struct {
	AttrPath string      // e.g., "userName", "emails[type eq \"work\"].value"
	Op       ComparisonOp
	Value    any // string, bool, number
}

func (e *AttrExpression) Evaluate(attrs map[string]any) bool {
	val := resolveAttrPath(attrs, e.AttrPath)
	return evaluateComparison(val, e.Op, e.Value)
}

func (e *AttrExpression) String() string {
	if e.Op == OpPr {
		return fmt.Sprintf("%s pr", e.AttrPath)
	}
	return fmt.Sprintf("%s %s %v", e.AttrPath, e.Op, e.Value)
}

// AndExpr is a logical AND of two sub-expressions.
type AndExpr struct {
	Left, Right FilterExpr
}

func (a *AndExpr) Evaluate(attrs map[string]any) bool {
	return a.Left.Evaluate(attrs) && a.Right.Evaluate(attrs)
}

func (a *AndExpr) String() string {
	return fmt.Sprintf("(%s and %s)", a.Left, a.Right)
}

// OrExpr is a logical OR of two sub-expressions.
type OrExpr struct {
	Left, Right FilterExpr
}

func (o *OrExpr) Evaluate(attrs map[string]any) bool {
	return o.Left.Evaluate(attrs) || o.Right.Evaluate(attrs)
}

func (o *OrExpr) String() string {
	return fmt.Sprintf("(%s or %s)", o.Left, o.Right)
}

// NotExpr is a logical NOT of a sub-expression.
type NotExpr struct {
	Inner FilterExpr
}

func (n *NotExpr) Evaluate(attrs map[string]any) bool {
	return !n.Inner.Evaluate(attrs)
}

func (n *NotExpr) String() string {
	return fmt.Sprintf("not (%s)", n.Inner)
}

// --- Lexer ---

type tokenKind int

const (
	tkEOF tokenKind = iota
	tkIdent      // attribute name or operator keyword
	tkString     // quoted value
	tkNumber     // numeric value
	tkLParen     // (
	tkRParen     // )
	tkLBracket   // [
	tkRBracket   // ]
	tkDot        // .
	tkAnd        // and
	tkOr         // or
	tkNot        // not
)

type token struct {
	kind  tokenKind
	value string
}

type lexer struct {
	input string
	pos   int
}

func newLexer(input string) *lexer {
	return &lexer{input: input}
}

func (l *lexer) tokenize() ([]token, error) {
	var tokens []token
	for l.pos < len(l.input) {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			break
		}

		ch := l.input[l.pos]

		switch {
		case ch == '(':
			tokens = append(tokens, token{tkLParen, "("})
			l.pos++
		case ch == ')':
			tokens = append(tokens, token{tkRParen, ")"})
			l.pos++
		case ch == '[':
			tokens = append(tokens, token{tkLBracket, "["})
			l.pos++
		case ch == ']':
			tokens = append(tokens, token{tkRBracket, "]"})
			l.pos++
		case ch == '.':
			tokens = append(tokens, token{tkDot, "."})
			l.pos++
		case ch == '"':
			val, err := l.readString()
			if err != nil {
				return nil, err
			}
			tokens = append(tokens, token{tkString, val})
		case isDigit(ch) || (ch == '-' && l.pos+1 < len(l.input) && isDigit(l.input[l.pos+1])):
			val := l.readNumber()
			tokens = append(tokens, token{tkNumber, val})
		case isAlpha(ch) || ch == '_':
			val := l.readIdent()
			lower := strings.ToLower(val)
			switch lower {
			case "and":
				tokens = append(tokens, token{tkAnd, val})
			case "or":
				tokens = append(tokens, token{tkOr, val})
			case "not":
				tokens = append(tokens, token{tkNot, val})
			default:
				tokens = append(tokens, token{tkIdent, val})
			}
		default:
			return nil, fmt.Errorf("unexpected character %q at position %d", ch, l.pos)
		}
	}
	tokens = append(tokens, token{tkEOF, ""})
	return tokens, nil
}

func (l *lexer) skipWhitespace() {
	for l.pos < len(l.input) && (l.input[l.pos] == ' ' || l.input[l.pos] == '\t' || l.input[l.pos] == '\n') {
		l.pos++
	}
}

func (l *lexer) readString() (string, error) {
	l.pos++ // skip opening quote
	var sb strings.Builder
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == '\\' && l.pos+1 < len(l.input) {
			next := l.input[l.pos+1]
			sb.WriteByte(next)
			l.pos += 2
			continue
		}
		if ch == '"' {
			l.pos++
			return sb.String(), nil
		}
		sb.WriteByte(ch)
		l.pos++
	}
	return "", fmt.Errorf("unterminated string literal")
}

func (l *lexer) readNumber() string {
	start := l.pos
	if l.input[l.pos] == '-' {
		l.pos++
	}
	for l.pos < len(l.input) && (isDigit(l.input[l.pos]) || l.input[l.pos] == '.') {
		l.pos++
	}
	return l.input[start:l.pos]
}

func (l *lexer) readIdent() string {
	start := l.pos
	for l.pos < len(l.input) && (isAlpha(l.input[l.pos]) || isDigit(l.input[l.pos]) || l.input[l.pos] == '_') {
		l.pos++
	}
	return l.input[start:l.pos]
}

func isDigit(ch byte) bool { return ch >= '0' && ch <= '9' }
func isAlpha(ch byte) bool { return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') }

// --- Parser ---

type parser struct {
	tokens []token
	pos    int
}

// ParseFilter parses a SCIM filter expression into an AST.
// Returns nil and nil error for empty input (no filter).
func ParseFilter(filter string) (FilterExpr, error) {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return nil, nil
	}

	lex := newLexer(filter)
	tokens, err := lex.tokenize()
	if err != nil {
		return nil, fmt.Errorf("filter lex error: %w", err)
	}

	p := &parser{tokens: tokens}
	expr, err := p.parseOr()
	if err != nil {
		return nil, err
	}
	if p.peek().kind != tkEOF {
		return nil, fmt.Errorf("unexpected token %q after filter", p.peek().value)
	}
	return expr, nil
}

// parseOr handles the lowest precedence operator: or
func (p *parser) parseOr() (FilterExpr, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}

	for p.peek().kind == tkOr {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &OrExpr{Left: left, Right: right}
	}
	return left, nil
}

// parseAnd handles the and operator
func (p *parser) parseAnd() (FilterExpr, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}

	for p.peek().kind == tkAnd {
		p.advance()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &AndExpr{Left: left, Right: right}
	}
	return left, nil
}

// parseNot handles the not operator (unary prefix)
func (p *parser) parseNot() (FilterExpr, error) {
	if p.peek().kind == tkNot {
		p.advance()
		inner, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &NotExpr{Inner: inner}, nil
	}
	return p.parsePrimary()
}

// parsePrimary handles parentheses and leaf expressions
func (p *parser) parsePrimary() (FilterExpr, error) {
	tok := p.peek()

	if tok.kind == tkLParen {
		p.advance()
		expr, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tkRParen {
			return nil, fmt.Errorf("expected ')' but got %q", p.peek().value)
		}
		p.advance()
		return expr, nil
	}

	return p.parseAttrExpr()
}

// parseAttrExpr parses a leaf comparison: attrPath op value
func (p *parser) parseAttrExpr() (FilterExpr, error) {
	tok := p.peek()
	if tok.kind != tkIdent {
		return nil, fmt.Errorf("expected attribute name but got %q", tok.value)
	}

	// Read the full attribute path including sub-attributes, brackets, etc.
	attrPath, err := p.parseAttrPath()
	if err != nil {
		return nil, err
	}

	// Next token should be the operator
	opTok := p.peek()
	if opTok.kind != tkIdent {
		return nil, fmt.Errorf("expected comparison operator after %q but got %q", attrPath, opTok.value)
	}

	opStr := strings.ToLower(opTok.value)
	op, valid := parseComparisonOp(opStr)
	if !valid {
		return nil, fmt.Errorf("unknown comparison operator %q", opTok.value)
	}
	p.advance()

	// 'pr' (present) has no value
	if op == OpPr {
		return &AttrExpression{AttrPath: attrPath, Op: op}, nil
	}

	// Read the comparison value
	valTok := p.peek()
	var val any
	switch valTok.kind {
	case tkString:
		val = valTok.value
	case tkNumber:
		val = valTok.value
	case tkIdent:
		// true, false, null
		lower := strings.ToLower(valTok.value)
		switch lower {
		case "true":
			val = true
		case "false":
			val = false
		case "null":
			val = nil
		default:
			val = valTok.value // unquoted string
		}
	default:
		return nil, fmt.Errorf("expected value after operator %q but got %q", opStr, valTok.value)
	}
	p.advance()

	return &AttrExpression{AttrPath: attrPath, Op: op, Value: val}, nil
}

// parseAttrPath reads a full SCIM attribute path including:
//   - Simple: userName
//   - Sub-attribute: name.familyName
//   - Multi-valued with filter: emails[type eq "work"].value
func (p *parser) parseAttrPath() (string, error) {
	var sb strings.Builder

	// First identifier
	tok := p.peek()
	if tok.kind != tkIdent {
		return "", fmt.Errorf("expected attribute name but got %q", tok.value)
	}
	sb.WriteString(tok.value)
	p.advance()

	// Read extensions: .subAttr, [filter], etc.
	for {
		next := p.peek()
		switch next.kind {
		case tkDot:
			sb.WriteString(".")
			p.advance()
			subTok := p.peek()
			if subTok.kind != tkIdent {
				return "", fmt.Errorf("expected sub-attribute name after '.' but got %q", subTok.value)
			}
			sb.WriteString(subTok.value)
			p.advance()

		case tkLBracket:
			// Multi-valued attribute filter: [type eq "work"]
			sb.WriteString("[")
			p.advance()
			// Read until matching ]
			depth := 1
			for p.peek().kind != tkEOF {
				t := p.peek()
				if t.kind == tkLBracket {
					depth++
				}
				if t.kind == tkRBracket {
					depth--
					if depth == 0 {
						break
					}
				}
				// Reconstruct the inner filter string
				switch t.kind {
				case tkString:
					sb.WriteString(fmt.Sprintf("\"%s\"", t.value))
				default:
					sb.WriteString(t.value)
				}
				sb.WriteString(" ")
				p.advance()
			}
			if p.peek().kind != tkRBracket {
				return "", fmt.Errorf("unterminated '[' in attribute path")
			}
			p.advance() // consume ]
			sb.WriteString("]")

		default:
			return sb.String(), nil
		}
	}
}

func parseComparisonOp(s string) (ComparisonOp, bool) {
	switch s {
	case "eq":
		return OpEq, true
	case "ne":
		return OpNe, true
	case "co":
		return OpCo, true
	case "sw":
		return OpSw, true
	case "ew":
		return OpEw, true
	case "pr":
		return OpPr, true
	case "gt":
		return OpGt, true
	case "ge":
		return OpGe, true
	case "lt":
		return OpLt, true
	case "le":
		return OpLe, true
	default:
		return "", false
	}
}

func (p *parser) peek() token {
	if p.pos >= len(p.tokens) {
		return token{kind: tkEOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) advance() {
	if p.pos < len(p.tokens) {
		p.pos++
	}
}

// --- Evaluation ---

// resolveAttrPath resolves an attribute path against a flat attrs map.
// Supports:
//   - Simple: userName
//   - Sub-attribute: name.familyName
//   - Multi-valued filter: emails[type eq "work"].value
func resolveAttrPath(attrs map[string]any, path string) any {
	// Simple attribute lookup
	if !strings.Contains(path, ".") && !strings.Contains(path, "[") {
		// Try case-insensitive lookup
		for k, v := range attrs {
			if strings.EqualFold(k, path) {
				return v
			}
		}
		return nil
	}

	// Handle sub-attribute path: name.familyName
	if strings.Contains(path, ".") && !strings.Contains(path, "[") {
		parts := strings.SplitN(path, ".", 2)
		parent := resolveAttrPath(attrs, parts[0])
		if m, ok := parent.(map[string]any); ok {
			for k, v := range m {
				if strings.EqualFold(k, parts[1]) {
					return v
				}
			}
		}
		return nil
	}

	// Handle multi-valued attribute with filter: emails[type eq "work"].value
	if strings.Contains(path, "[") {
		bracketIdx := strings.Index(path, "[")
		attrName := path[:bracketIdx]
		closeIdx := strings.Index(path[bracketIdx:], "]")
		if closeIdx < 0 {
			return nil
		}
		filterStr := path[bracketIdx+1 : bracketIdx+closeIdx]
		subPath := ""
		if bracketIdx+closeIdx+1 < len(path) {
			rest := path[bracketIdx+closeIdx+1:]
			subPath = strings.TrimPrefix(rest, ".")
		}

		// Get the array attribute
		arr := resolveAttrPath(attrs, attrName)
		items, ok := arr.([]any)
		if !ok {
			return nil
		}

		// Parse and apply the inner filter
		innerFilter, err := ParseFilter(filterStr)
		if err != nil {
			return nil
		}

		var results []any
		for _, item := range items {
			if m, ok := item.(map[string]any); ok {
				if innerFilter == nil || innerFilter.Evaluate(m) {
					if subPath != "" {
						results = append(results, resolveAttrPath(m, subPath))
					} else {
						results = append(results, item)
					}
				}
			}
		}

		if len(results) == 0 {
			return nil
		}
		if len(results) == 1 {
			return results[0]
		}
		return results
	}

	return nil
}

// evaluateComparison applies a SCIM comparison operator.
func evaluateComparison(actual any, op ComparisonOp, expected any) bool {
	switch op {
	case OpPr:
		return actual != nil
	case OpEq:
		return valuesEqual(actual, expected)
	case OpNe:
		return !valuesEqual(actual, expected)
	case OpCo:
		return strings.Contains(toStr(actual), toStr(expected))
	case OpSw:
		return strings.HasPrefix(toStr(actual), toStr(expected))
	case OpEw:
		return strings.HasSuffix(toStr(actual), toStr(expected))
	case OpGt:
		return compareValues(actual, expected) > 0
	case OpGe:
		return compareValues(actual, expected) >= 0
	case OpLt:
		return compareValues(actual, expected) < 0
	case OpLe:
		return compareValues(actual, expected) <= 0
	}
	return false
}

func valuesEqual(a, b any) bool {
	if a == nil || b == nil {
		return a == b
	}
	return strings.EqualFold(toStr(a), toStr(b))
}

func toStr(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case float64:
		return fmt.Sprintf("%g", val)
	case int:
		return fmt.Sprintf("%d", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case time.Time:
		return val.Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func compareValues(a, b any) int {
	// Try time comparison first
	aStr, bStr := toStr(a), toStr(b)

	ta, errA := time.Parse(time.RFC3339, aStr)
	tb, errB := time.Parse(time.RFC3339, bStr)
	if errA == nil && errB == nil {
		if ta.Before(tb) {
			return -1
		}
		if ta.After(tb) {
			return 1
		}
		return 0
	}

	// Fall back to string comparison
	return strings.Compare(aStr, bStr)
}
