package litequery

import (
	"math"
	"strconv"
	"strings"

	"github.com/SigNoz/signoz/pkg/units"
)

// FormulaValueType is deliberately separate from telemetry ValueType. Formula
// expressions operate on aggregate result values, never on raw telemetry
// fields.
type FormulaValueType string

const (
	FormulaValueNumber FormulaValueType = "number"
	FormulaValueBool   FormulaValueType = "bool"
)

// FormulaStaticType is the type inferred before execution. Unit is empty only
// for dimensionless numbers; bool values never carry a unit.
type FormulaStaticType struct {
	Kind FormulaValueType
	Unit string
}

func (t FormulaStaticType) valid() bool {
	if t.Kind != FormulaValueNumber && t.Kind != FormulaValueBool {
		return false
	}
	return t.Kind != FormulaValueBool || t.Unit == ""
}

// FormulaBinding describes one named input available to an expression. A
// SeriesSignature is a caller-owned stable representation of timestamp and
// group-by columns. Inputs in an expression must have the same non-empty
// signature, which prevents invalid cross-series arithmetic before execution.
type FormulaBinding struct {
	Type            FormulaStaticType
	SeriesSignature string
}

// FormulaValue is the evaluated value of one input for a timestamp/group key.
// Missing is distinct from a numeric NaN and is deliberately propagated by
// every formula operation, including boolean AND and OR.
type FormulaValue struct {
	Type    FormulaStaticType
	Number  float64
	Bool    bool
	Missing bool
}

type typedFormulaTokenKind uint8

const (
	typedFormulaIdentifier typedFormulaTokenKind = iota
	typedFormulaNumber
	typedFormulaArithmetic
	typedFormulaComparison
	typedFormulaAnd
	typedFormulaOr
	typedFormulaNot
	typedFormulaLeftParen
	typedFormulaRightParen
	typedFormulaComma
	typedFormulaEOF
)

type typedFormulaToken struct {
	kind  typedFormulaTokenKind
	value string
	pos   int
}

type typedFormulaNode interface {
	typedFormulaNode()
}

type typedFormulaNumberNode struct {
	value float64
}

func (*typedFormulaNumberNode) typedFormulaNode() {}

type typedFormulaReferenceNode struct {
	name string
}

func (*typedFormulaReferenceNode) typedFormulaNode() {}

type typedFormulaUnaryNode struct {
	op    string
	value typedFormulaNode
}

func (*typedFormulaUnaryNode) typedFormulaNode() {}

type typedFormulaBinaryNode struct {
	op    string
	left  typedFormulaNode
	right typedFormulaNode
}

func (*typedFormulaBinaryNode) typedFormulaNode() {}

type typedFormulaCallNode struct {
	name string
	args []typedFormulaNode
}

func (*typedFormulaCallNode) typedFormulaNode() {}

type typedFormulaParser struct {
	tokens []typedFormulaToken
	index  int
	refs   []string
}

// TypedFormula is an immutable parsed and type-checked Formula expression.
// Its unexported AST keeps the public contract intentionally small: callers
// receive the canonical expression, inferred type, referenced names and a
// deterministic evaluator.
type TypedFormula struct {
	root            typedFormulaNode
	types           map[typedFormulaNode]formulaInference
	canonical       string
	references      []string
	resultType      FormulaStaticType
	seriesSignature string
}

func (f *TypedFormula) Canonical() string {
	if f == nil {
		return ""
	}
	return f.canonical
}

func (f *TypedFormula) References() []string {
	if f == nil {
		return nil
	}
	return append([]string(nil), f.references...)
}

func (f *TypedFormula) Type() FormulaStaticType {
	if f == nil {
		return FormulaStaticType{}
	}
	return f.resultType
}

func (f *TypedFormula) SeriesSignature() string {
	if f == nil {
		return ""
	}
	return f.seriesSignature
}

// AnalyzeTypedFormula parses and statically validates one expression against
// its already-known named bindings. It is useful for a formula that does not
// depend on another Formula in the same request.
func AnalyzeTypedFormula(expression string, bindings map[string]FormulaBinding) (*TypedFormula, error) {
	root, references, err := parseTypedFormula(expression)
	if err != nil {
		return nil, err
	}
	resolver := func(name string) (FormulaBinding, error) {
		binding, ok := bindings[name]
		if !ok {
			return FormulaBinding{}, newError(ErrorInvalidFormula, "formula.expression", "formula references unknown query %q", name)
		}
		if !binding.Type.valid() {
			return FormulaBinding{}, newError(ErrorInvalidFormula, "formula.expression", "formula input %q has invalid static type", name)
		}
		return binding, nil
	}
	types := make(map[typedFormulaNode]formulaInference)
	inference, err := inferTypedFormula(root, resolver, types)
	if err != nil {
		return nil, err
	}
	return &TypedFormula{
		root:            root,
		types:           types,
		canonical:       formatTypedFormula(root, 0),
		references:      distinctFormulaReferences(references),
		resultType:      inference.typ,
		seriesSignature: inference.seriesSignature,
	}, nil
}

// AnalyzeTypedFormulaSet supports forward formula references while rejecting
// duplicate names, unknown names and dependency cycles. Base bindings represent
// data queries; the returned map contains only named formulas.
func AnalyzeTypedFormulaSet(formulas []Formula, bindings map[string]FormulaBinding) (map[string]*TypedFormula, error) {
	type parsedFormula struct {
		root       typedFormulaNode
		references []string
	}
	parsed := make(map[string]parsedFormula, len(formulas))
	for _, formula := range formulas {
		if !validName(formula.Name) {
			return nil, newError(ErrorInvalidFormula, "formula.name", "formula name must start with a letter and contain only letters, digits, or underscores")
		}
		if _, exists := bindings[formula.Name]; exists {
			return nil, newError(ErrorInvalidFormula, "formula.name", "formula name %q duplicates an input name", formula.Name)
		}
		if _, exists := parsed[formula.Name]; exists {
			return nil, newError(ErrorInvalidFormula, "formula.name", "duplicate formula name %q", formula.Name)
		}
		root, references, err := parseTypedFormula(formula.Expression)
		if err != nil {
			return nil, err
		}
		parsed[formula.Name] = parsedFormula{root: root, references: distinctFormulaReferences(references)}
	}

	results := make(map[string]*TypedFormula, len(formulas))
	state := make(map[string]uint8, len(formulas))
	var analyze func(string) (*TypedFormula, error)
	analyze = func(name string) (*TypedFormula, error) {
		if result, ok := results[name]; ok {
			return result, nil
		}
		switch state[name] {
		case 1:
			return nil, newError(ErrorInvalidFormula, "formula.expression", "formula dependency cycle includes %q", name)
		case 2:
			return results[name], nil
		}
		current, ok := parsed[name]
		if !ok {
			return nil, newError(ErrorInvalidFormula, "formula.expression", "formula references unknown query %q", name)
		}
		state[name] = 1
		resolver := func(reference string) (FormulaBinding, error) {
			if binding, ok := bindings[reference]; ok {
				if !binding.Type.valid() {
					return FormulaBinding{}, newError(ErrorInvalidFormula, "formula.expression", "formula input %q has invalid static type", reference)
				}
				return binding, nil
			}
			formula, ok := parsed[reference]
			if !ok {
				return FormulaBinding{}, newError(ErrorInvalidFormula, "formula.expression", "formula %q references unknown query %q", name, reference)
			}
			_ = formula
			result, err := analyze(reference)
			if err != nil {
				return FormulaBinding{}, err
			}
			return FormulaBinding{Type: result.resultType, SeriesSignature: result.seriesSignature}, nil
		}
		types := make(map[typedFormulaNode]formulaInference)
		inference, err := inferTypedFormula(current.root, resolver, types)
		if err != nil {
			return nil, err
		}
		result := &TypedFormula{
			root:            current.root,
			types:           types,
			canonical:       formatTypedFormula(current.root, 0),
			references:      current.references,
			resultType:      inference.typ,
			seriesSignature: inference.seriesSignature,
		}
		results[name] = result
		state[name] = 2
		return result, nil
	}
	for _, formula := range formulas {
		if _, err := analyze(formula.Name); err != nil {
			return nil, err
		}
	}
	return results, nil
}

// Evaluate executes one typed expression for one aligned timestamp/group key.
// Missing propagates through all operations; a caller must route it through the
// alert No Data policy instead of coercing it to zero or false.
func (f *TypedFormula) Evaluate(values map[string]FormulaValue) (FormulaValue, error) {
	if f == nil || f.root == nil {
		return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "formula is required")
	}
	return evaluateTypedFormula(f.root, f.types, values)
}

func parseTypedFormula(expression string) (typedFormulaNode, []string, error) {
	tokens, err := tokenizeTypedFormula(expression)
	if err != nil {
		return nil, nil, err
	}
	parser := typedFormulaParser{tokens: tokens}
	root, err := parser.parseOr()
	if err != nil {
		return nil, nil, err
	}
	if parser.current().kind != typedFormulaEOF {
		return nil, nil, newError(ErrorInvalidFormula, "formula.expression", "unexpected token %q", parser.current().value)
	}
	return root, parser.refs, nil
}

func tokenizeTypedFormula(expression string) ([]typedFormulaToken, error) {
	input := strings.TrimSpace(expression)
	if input == "" {
		return nil, newError(ErrorInvalidFormula, "formula.expression", "formula expression is required")
	}
	tokens := make([]typedFormulaToken, 0, len(input)/2)
	for index := 0; index < len(input); {
		if isFormulaWhitespace(input[index]) {
			index++
			continue
		}
		start := index
		if isFormulaIdentifierStart(input[index]) {
			index++
			for index < len(input) && isFormulaIdentifierPart(input[index]) {
				index++
			}
			value := input[start:index]
			switch strings.ToUpper(value) {
			case "AND":
				tokens = append(tokens, typedFormulaToken{kind: typedFormulaAnd, value: "AND", pos: start})
			case "OR":
				tokens = append(tokens, typedFormulaToken{kind: typedFormulaOr, value: "OR", pos: start})
			case "NOT":
				tokens = append(tokens, typedFormulaToken{kind: typedFormulaNot, value: "NOT", pos: start})
			default:
				tokens = append(tokens, typedFormulaToken{kind: typedFormulaIdentifier, value: value, pos: start})
			}
			continue
		}
		if isFormulaDigit(input[index]) || input[index] == '.' {
			index++
			for index < len(input) && (isFormulaDigit(input[index]) || input[index] == '.') {
				index++
			}
			value := input[start:index]
			if _, err := strconv.ParseFloat(value, 64); err != nil {
				return nil, newError(ErrorInvalidFormula, "formula.expression", "invalid number %q", value)
			}
			tokens = append(tokens, typedFormulaToken{kind: typedFormulaNumber, value: value, pos: start})
			continue
		}
		if index+1 < len(input) {
			two := input[index : index+2]
			switch two {
			case ">=", "<=", "!=":
				tokens = append(tokens, typedFormulaToken{kind: typedFormulaComparison, value: two, pos: start})
				index += 2
				continue
			case "==":
				return nil, newError(ErrorInvalidFormula, "formula.expression", "operator == is not supported; use =")
			case "&&", "||":
				return nil, newError(ErrorInvalidFormula, "formula.expression", "operator %s is not supported; use AND or OR", two)
			}
		}
		switch input[index] {
		case '+', '-', '*', '/':
			tokens = append(tokens, typedFormulaToken{kind: typedFormulaArithmetic, value: input[index : index+1], pos: start})
		case '>', '<', '=':
			tokens = append(tokens, typedFormulaToken{kind: typedFormulaComparison, value: input[index : index+1], pos: start})
		case '!':
			return nil, newError(ErrorInvalidFormula, "formula.expression", "operator ! is not supported; use NOT or !=")
		case '(':
			tokens = append(tokens, typedFormulaToken{kind: typedFormulaLeftParen, value: "(", pos: start})
		case ')':
			tokens = append(tokens, typedFormulaToken{kind: typedFormulaRightParen, value: ")", pos: start})
		case ',':
			tokens = append(tokens, typedFormulaToken{kind: typedFormulaComma, value: ",", pos: start})
		default:
			return nil, newError(ErrorInvalidFormula, "formula.expression", "unsupported token %q", input[index:index+1])
		}
		index++
	}
	tokens = append(tokens, typedFormulaToken{kind: typedFormulaEOF, pos: len(input)})
	return tokens, nil
}

func isFormulaWhitespace(value byte) bool {
	return value == ' ' || value == '\t' || value == '\r' || value == '\n'
}

func isFormulaIdentifierStart(value byte) bool {
	return (value >= 'A' && value <= 'Z') || (value >= 'a' && value <= 'z')
}

func isFormulaIdentifierPart(value byte) bool {
	return isFormulaIdentifierStart(value) || isFormulaDigit(value) || value == '_'
}

func isFormulaDigit(value byte) bool { return value >= '0' && value <= '9' }

func (p *typedFormulaParser) current() typedFormulaToken { return p.tokens[p.index] }

func (p *typedFormulaParser) advance() typedFormulaToken {
	current := p.current()
	if current.kind != typedFormulaEOF {
		p.index++
	}
	return current
}

func (p *typedFormulaParser) parseOr() (typedFormulaNode, error) {
	left, err := p.parseAnd()
	if err != nil {
		return nil, err
	}
	for p.current().kind == typedFormulaOr {
		p.advance()
		right, err := p.parseAnd()
		if err != nil {
			return nil, err
		}
		left = &typedFormulaBinaryNode{op: "OR", left: left, right: right}
	}
	return left, nil
}

func (p *typedFormulaParser) parseAnd() (typedFormulaNode, error) {
	left, err := p.parseNot()
	if err != nil {
		return nil, err
	}
	for p.current().kind == typedFormulaAnd {
		p.advance()
		right, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		left = &typedFormulaBinaryNode{op: "AND", left: left, right: right}
	}
	return left, nil
}

func (p *typedFormulaParser) parseNot() (typedFormulaNode, error) {
	if p.current().kind == typedFormulaNot {
		p.advance()
		value, err := p.parseNot()
		if err != nil {
			return nil, err
		}
		return &typedFormulaUnaryNode{op: "NOT", value: value}, nil
	}
	return p.parseComparison()
}

func (p *typedFormulaParser) parseComparison() (typedFormulaNode, error) {
	left, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	if p.current().kind != typedFormulaComparison {
		return left, nil
	}
	op := p.advance().value
	right, err := p.parseAdditive()
	if err != nil {
		return nil, err
	}
	if p.current().kind == typedFormulaComparison {
		return nil, newError(ErrorInvalidFormula, "formula.expression", "chained comparisons are not supported; join comparisons with AND")
	}
	return &typedFormulaBinaryNode{op: op, left: left, right: right}, nil
}

func (p *typedFormulaParser) parseAdditive() (typedFormulaNode, error) {
	left, err := p.parseMultiplicative()
	if err != nil {
		return nil, err
	}
	for p.current().kind == typedFormulaArithmetic && (p.current().value == "+" || p.current().value == "-") {
		op := p.advance().value
		right, err := p.parseMultiplicative()
		if err != nil {
			return nil, err
		}
		left = &typedFormulaBinaryNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *typedFormulaParser) parseMultiplicative() (typedFormulaNode, error) {
	left, err := p.parseUnaryNumber()
	if err != nil {
		return nil, err
	}
	for p.current().kind == typedFormulaArithmetic && (p.current().value == "*" || p.current().value == "/") {
		op := p.advance().value
		right, err := p.parseUnaryNumber()
		if err != nil {
			return nil, err
		}
		left = &typedFormulaBinaryNode{op: op, left: left, right: right}
	}
	return left, nil
}

func (p *typedFormulaParser) parseUnaryNumber() (typedFormulaNode, error) {
	if p.current().kind == typedFormulaArithmetic && (p.current().value == "+" || p.current().value == "-") {
		op := p.advance().value
		value, err := p.parseUnaryNumber()
		if err != nil {
			return nil, err
		}
		return &typedFormulaUnaryNode{op: op, value: value}, nil
	}
	return p.parsePrimary()
}

func (p *typedFormulaParser) parsePrimary() (typedFormulaNode, error) {
	token := p.advance()
	switch token.kind {
	case typedFormulaNumber:
		value, err := strconv.ParseFloat(token.value, 64)
		if err != nil {
			return nil, newError(ErrorInvalidFormula, "formula.expression", "invalid number %q", token.value)
		}
		return &typedFormulaNumberNode{value: value}, nil
	case typedFormulaIdentifier:
		if p.current().kind == typedFormulaLeftParen {
			p.advance()
			args := make([]typedFormulaNode, 0, 3)
			if p.current().kind != typedFormulaRightParen {
				for {
					arg, err := p.parseOr()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
					if p.current().kind != typedFormulaComma {
						break
					}
					p.advance()
				}
			}
			if p.current().kind != typedFormulaRightParen {
				return nil, newError(ErrorInvalidFormula, "formula.expression", "missing closing parenthesis for function %q", token.value)
			}
			p.advance()
			return &typedFormulaCallNode{name: strings.ToLower(token.value), args: args}, nil
		}
		p.refs = append(p.refs, token.value)
		return &typedFormulaReferenceNode{name: token.value}, nil
	case typedFormulaLeftParen:
		value, err := p.parseOr()
		if err != nil {
			return nil, err
		}
		if p.current().kind != typedFormulaRightParen {
			return nil, newError(ErrorInvalidFormula, "formula.expression", "missing closing parenthesis")
		}
		p.advance()
		return value, nil
	default:
		return nil, newError(ErrorInvalidFormula, "formula.expression", "expected formula operand at character %d", token.pos+1)
	}
}

type formulaInference struct {
	typ             FormulaStaticType
	seriesSignature string
	constant        *float64
}

type formulaBindingResolver func(string) (FormulaBinding, error)

func inferTypedFormula(node typedFormulaNode, resolve formulaBindingResolver, inferred map[typedFormulaNode]formulaInference) (formulaInference, error) {
	if result, ok := inferred[node]; ok {
		return result, nil
	}
	var (
		result formulaInference
		err    error
	)
	switch current := node.(type) {
	case *typedFormulaNumberNode:
		value := current.value
		result = formulaInference{typ: FormulaStaticType{Kind: FormulaValueNumber}, constant: &value}
	case *typedFormulaReferenceNode:
		binding, resolveErr := resolve(current.name)
		if resolveErr != nil {
			return formulaInference{}, resolveErr
		}
		result = formulaInference{typ: binding.Type, seriesSignature: binding.SeriesSignature}
	case *typedFormulaUnaryNode:
		value, inferErr := inferTypedFormula(current.value, resolve, inferred)
		if inferErr != nil {
			return formulaInference{}, inferErr
		}
		if current.op == "NOT" {
			if value.typ.Kind != FormulaValueBool {
				return formulaInference{}, newError(ErrorInvalidFormula, "formula.expression", "NOT requires a bool operand")
			}
			result = formulaInference{typ: FormulaStaticType{Kind: FormulaValueBool}, seriesSignature: value.seriesSignature}
		} else {
			if value.typ.Kind != FormulaValueNumber {
				return formulaInference{}, newError(ErrorInvalidFormula, "formula.expression", "unary %s requires a number operand", current.op)
			}
			result = value
			if value.constant != nil && current.op == "-" {
				constant := -*value.constant
				result.constant = &constant
			}
		}
	case *typedFormulaBinaryNode:
		left, inferErr := inferTypedFormula(current.left, resolve, inferred)
		if inferErr != nil {
			return formulaInference{}, inferErr
		}
		right, inferErr := inferTypedFormula(current.right, resolve, inferred)
		if inferErr != nil {
			return formulaInference{}, inferErr
		}
		seriesSignature, signatureErr := mergeFormulaSeriesSignatures(left.seriesSignature, right.seriesSignature)
		if signatureErr != nil {
			return formulaInference{}, signatureErr
		}
		switch current.op {
		case "+", "-":
			unit, unitErr := compatibleFormulaUnit(left, right)
			if unitErr != nil {
				return formulaInference{}, unitErr
			}
			result = formulaInference{typ: FormulaStaticType{Kind: FormulaValueNumber, Unit: unit}, seriesSignature: seriesSignature}
			result.constant = foldFormulaConstant(current.op, left.constant, right.constant)
		case "*":
			unit, unitErr := multiplyFormulaUnit(left, right)
			if unitErr != nil {
				return formulaInference{}, unitErr
			}
			result = formulaInference{typ: FormulaStaticType{Kind: FormulaValueNumber, Unit: unit}, seriesSignature: seriesSignature}
			result.constant = foldFormulaConstant(current.op, left.constant, right.constant)
		case "/":
			unit, unitErr := divideFormulaUnit(left, right)
			if unitErr != nil {
				return formulaInference{}, unitErr
			}
			if right.constant != nil && *right.constant == 0 {
				return formulaInference{}, newError(ErrorInvalidFormula, "formula.expression", "division by literal zero")
			}
			result = formulaInference{typ: FormulaStaticType{Kind: FormulaValueNumber, Unit: unit}, seriesSignature: seriesSignature}
			result.constant = foldFormulaConstant(current.op, left.constant, right.constant)
		case ">", ">=", "<", "<=", "=", "!=":
			if _, unitErr := compatibleFormulaUnit(left, right); unitErr != nil {
				return formulaInference{}, unitErr
			}
			result = formulaInference{typ: FormulaStaticType{Kind: FormulaValueBool}, seriesSignature: seriesSignature}
		case "AND", "OR":
			if left.typ.Kind != FormulaValueBool || right.typ.Kind != FormulaValueBool {
				return formulaInference{}, newError(ErrorInvalidFormula, "formula.expression", "%s requires bool operands", current.op)
			}
			result = formulaInference{typ: FormulaStaticType{Kind: FormulaValueBool}, seriesSignature: seriesSignature}
		default:
			err = newError(ErrorInvalidFormula, "formula.expression", "unsupported formula operator %q", current.op)
		}
	case *typedFormulaCallNode:
		arguments := make([]formulaInference, 0, len(current.args))
		for _, arg := range current.args {
			argument, inferErr := inferTypedFormula(arg, resolve, inferred)
			if inferErr != nil {
				return formulaInference{}, inferErr
			}
			arguments = append(arguments, argument)
		}
		result, err = inferTypedFormulaCall(current.name, arguments)
	default:
		err = newError(ErrorInvalidFormula, "formula.expression", "unsupported formula expression")
	}
	if err != nil {
		return formulaInference{}, err
	}
	inferred[node] = result
	return result, nil
}

func inferTypedFormulaCall(name string, args []formulaInference) (formulaInference, error) {
	for _, arg := range args {
		if arg.typ.Kind != FormulaValueNumber {
			return formulaInference{}, newError(ErrorInvalidFormula, "formula.expression", "function %s requires number arguments", name)
		}
	}
	switch name {
	case "abs":
		if len(args) != 1 {
			return formulaInference{}, newError(ErrorInvalidFormula, "formula.expression", "function abs expects 1 argument")
		}
		result := args[0]
		if result.constant != nil {
			constant := math.Abs(*result.constant)
			result.constant = &constant
		}
		return result, nil
	case "min", "max":
		if len(args) != 2 {
			return formulaInference{}, newError(ErrorInvalidFormula, "formula.expression", "function %s expects 2 arguments", name)
		}
		unit, err := compatibleFormulaUnit(args[0], args[1])
		if err != nil {
			return formulaInference{}, err
		}
		signature, err := mergeFormulaSeriesSignatures(args[0].seriesSignature, args[1].seriesSignature)
		if err != nil {
			return formulaInference{}, err
		}
		result := formulaInference{typ: FormulaStaticType{Kind: FormulaValueNumber, Unit: unit}, seriesSignature: signature}
		if args[0].constant != nil && args[1].constant != nil {
			left, err := formulaConstantInUnit(args[0], unit)
			if err != nil {
				return formulaInference{}, err
			}
			right, err := formulaConstantInUnit(args[1], unit)
			if err != nil {
				return formulaInference{}, err
			}
			constant := math.Min(left, right)
			if name == "max" {
				constant = math.Max(left, right)
			}
			result.constant = &constant
		}
		return result, nil
	case "clamp":
		if len(args) != 3 {
			return formulaInference{}, newError(ErrorInvalidFormula, "formula.expression", "function clamp expects 3 arguments")
		}
		unit, err := compatibleFormulaUnit(args[0], args[1])
		if err != nil {
			return formulaInference{}, err
		}
		unit, err = compatibleFormulaUnit(formulaInference{typ: FormulaStaticType{Kind: FormulaValueNumber, Unit: unit}}, args[2])
		if err != nil {
			return formulaInference{}, err
		}
		signature, err := mergeFormulaSeriesSignatures(args[0].seriesSignature, args[1].seriesSignature)
		if err != nil {
			return formulaInference{}, err
		}
		signature, err = mergeFormulaSeriesSignatures(signature, args[2].seriesSignature)
		if err != nil {
			return formulaInference{}, err
		}
		if args[1].constant != nil && args[2].constant != nil && *args[1].constant > *args[2].constant {
			return formulaInference{}, newError(ErrorInvalidFormula, "formula.expression", "function clamp requires low to be less than or equal to high")
		}
		return formulaInference{typ: FormulaStaticType{Kind: FormulaValueNumber, Unit: unit}, seriesSignature: signature}, nil
	default:
		return formulaInference{}, newError(ErrorInvalidFormula, "formula.expression", "unsupported formula function %q", name)
	}
}

func formulaConstantInUnit(value formulaInference, targetUnit string) (float64, error) {
	if value.constant == nil {
		return 0, newError(ErrorInvalidFormula, "formula.expression", "formula value is not constant")
	}
	if targetUnit == "" || value.typ.Unit == "" || value.typ.Unit == targetUnit {
		return *value.constant, nil
	}
	if !units.AreCompatible(units.Unit(value.typ.Unit), units.Unit(targetUnit)) {
		return 0, newError(ErrorInvalidFormula, "formula.expression", "formula constant unit %q is incompatible with %q", value.typ.Unit, targetUnit)
	}
	return units.ConverterFromUnit(units.Unit(value.typ.Unit)).Convert(units.Value{F: *value.constant, U: units.Unit(value.typ.Unit)}, units.Unit(targetUnit)).F, nil
}

func mergeFormulaSeriesSignatures(left, right string) (string, error) {
	if left == "" {
		return right, nil
	}
	if right == "" || left == right {
		return left, nil
	}
	return "", newError(ErrorInvalidFormula, "formula.expression", "formula inputs must use identical timestamp and group columns")
}

func compatibleFormulaUnit(left, right formulaInference) (string, error) {
	if left.typ.Kind != FormulaValueNumber || right.typ.Kind != FormulaValueNumber {
		return "", newError(ErrorInvalidFormula, "formula.expression", "operator requires number operands")
	}
	leftUnit := strings.TrimSpace(left.typ.Unit)
	rightUnit := strings.TrimSpace(right.typ.Unit)
	switch {
	case leftUnit == "" && rightUnit == "":
		return "", nil
	case leftUnit == "":
		if left.constant != nil {
			return rightUnit, nil
		}
	case rightUnit == "":
		if right.constant != nil {
			return leftUnit, nil
		}
	case units.AreCompatible(units.Unit(leftUnit), units.Unit(rightUnit)):
		return leftUnit, nil
	}
	return "", newError(ErrorInvalidFormula, "formula.expression", "number operands use incompatible units %q and %q", leftUnit, rightUnit)
}

func multiplyFormulaUnit(left, right formulaInference) (string, error) {
	if left.typ.Kind != FormulaValueNumber || right.typ.Kind != FormulaValueNumber {
		return "", newError(ErrorInvalidFormula, "formula.expression", "* requires number operands")
	}
	if left.typ.Unit != "" && right.typ.Unit != "" {
		return "", newError(ErrorInvalidFormula, "formula.expression", "cannot multiply two unitful values")
	}
	if left.typ.Unit != "" {
		return left.typ.Unit, nil
	}
	return right.typ.Unit, nil
}

func divideFormulaUnit(left, right formulaInference) (string, error) {
	if left.typ.Kind != FormulaValueNumber || right.typ.Kind != FormulaValueNumber {
		return "", newError(ErrorInvalidFormula, "formula.expression", "/ requires number operands")
	}
	leftUnit := strings.TrimSpace(left.typ.Unit)
	rightUnit := strings.TrimSpace(right.typ.Unit)
	switch {
	case leftUnit == "" && rightUnit == "":
		return "", nil
	case leftUnit != "" && rightUnit == "":
		return leftUnit, nil
	case leftUnit != "" && rightUnit != "" && units.AreCompatible(units.Unit(leftUnit), units.Unit(rightUnit)):
		return "", nil
	default:
		return "", newError(ErrorInvalidFormula, "formula.expression", "division requires a dimensionless divisor or compatible units")
	}
}

func foldFormulaConstant(operator string, left, right *float64) *float64 {
	if left == nil || right == nil {
		return nil
	}
	var result float64
	switch operator {
	case "+":
		result = *left + *right
	case "-":
		result = *left - *right
	case "*":
		result = *left * *right
	case "/":
		if *right == 0 {
			return nil
		}
		result = *left / *right
	default:
		return nil
	}
	return &result
}

func evaluateTypedFormula(node typedFormulaNode, types map[typedFormulaNode]formulaInference, values map[string]FormulaValue) (FormulaValue, error) {
	inference, ok := types[node]
	if !ok {
		return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "formula was not type checked")
	}
	switch current := node.(type) {
	case *typedFormulaNumberNode:
		return FormulaValue{Type: FormulaStaticType{Kind: FormulaValueNumber}, Number: current.value}, nil
	case *typedFormulaReferenceNode:
		value, ok := values[current.name]
		if !ok {
			return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "formula value %q is missing", current.name)
		}
		if value.Type.Kind != inference.typ.Kind {
			return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "formula value %q has type %q, want %q", current.name, value.Type.Kind, inference.typ.Kind)
		}
		if value.Type.Kind == FormulaValueBool {
			if value.Type.Unit != "" {
				return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "bool formula value %q must not have a unit", current.name)
			}
			value.Type = inference.typ
			return value, nil
		}
		return normalizeFormulaValue(value, inference.typ)
	case *typedFormulaUnaryNode:
		value, err := evaluateTypedFormula(current.value, types, values)
		if err != nil {
			return FormulaValue{}, err
		}
		if value.Missing {
			return FormulaValue{Type: inference.typ, Missing: true}, nil
		}
		if current.op == "NOT" {
			return FormulaValue{Type: inference.typ, Bool: !value.Bool}, nil
		}
		if current.op == "-" {
			return checkedFormulaNumber(inference.typ, -value.Number), nil
		}
		return checkedFormulaNumber(inference.typ, value.Number), nil
	case *typedFormulaBinaryNode:
		left, err := evaluateTypedFormula(current.left, types, values)
		if err != nil {
			return FormulaValue{}, err
		}
		right, err := evaluateTypedFormula(current.right, types, values)
		if err != nil {
			return FormulaValue{}, err
		}
		if left.Missing || right.Missing {
			return FormulaValue{Type: inference.typ, Missing: true}, nil
		}
		switch current.op {
		case "+", "-", "*", "/":
			return evaluateTypedArithmetic(current.op, inference.typ, left, right)
		case ">", ">=", "<", "<=", "=", "!=":
			return evaluateTypedComparison(current.op, left, right)
		case "AND":
			return FormulaValue{Type: inference.typ, Bool: left.Bool && right.Bool}, nil
		case "OR":
			return FormulaValue{Type: inference.typ, Bool: left.Bool || right.Bool}, nil
		}
	case *typedFormulaCallNode:
		arguments := make([]FormulaValue, 0, len(current.args))
		for _, arg := range current.args {
			value, err := evaluateTypedFormula(arg, types, values)
			if err != nil {
				return FormulaValue{}, err
			}
			arguments = append(arguments, value)
		}
		for _, arg := range arguments {
			if arg.Missing {
				return FormulaValue{Type: inference.typ, Missing: true}, nil
			}
		}
		return evaluateTypedFormulaCall(current.name, inference.typ, arguments)
	}
	return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "unsupported formula expression")
}

func evaluateTypedFormulaCall(name string, resultType FormulaStaticType, args []FormulaValue) (FormulaValue, error) {
	switch name {
	case "abs":
		if len(args) != 1 || args[0].Type.Kind != FormulaValueNumber {
			return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "function abs expects 1 number argument")
		}
		return checkedFormulaNumber(resultType, math.Abs(args[0].Number)), nil
	case "min", "max":
		if len(args) != 2 || args[0].Type.Kind != FormulaValueNumber || args[1].Type.Kind != FormulaValueNumber {
			return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "function %s expects 2 number arguments", name)
		}
		left, err := formulaNumberInUnit(args[0], resultType.Unit)
		if err != nil {
			return FormulaValue{}, err
		}
		right, err := formulaNumberInUnit(args[1], resultType.Unit)
		if err != nil {
			return FormulaValue{}, err
		}
		if name == "min" {
			return checkedFormulaNumber(resultType, math.Min(left, right)), nil
		}
		return checkedFormulaNumber(resultType, math.Max(left, right)), nil
	case "clamp":
		if len(args) != 3 || args[0].Type.Kind != FormulaValueNumber || args[1].Type.Kind != FormulaValueNumber || args[2].Type.Kind != FormulaValueNumber {
			return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "function clamp expects 3 number arguments")
		}
		value, err := formulaNumberInUnit(args[0], resultType.Unit)
		if err != nil {
			return FormulaValue{}, err
		}
		low, err := formulaNumberInUnit(args[1], resultType.Unit)
		if err != nil {
			return FormulaValue{}, err
		}
		high, err := formulaNumberInUnit(args[2], resultType.Unit)
		if err != nil {
			return FormulaValue{}, err
		}
		if low > high {
			return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "function clamp requires low to be less than or equal to high")
		}
		return checkedFormulaNumber(resultType, math.Min(math.Max(value, low), high)), nil
	default:
		return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "unsupported formula function %q", name)
	}
}

func evaluateTypedArithmetic(operator string, resultType FormulaStaticType, left, right FormulaValue) (FormulaValue, error) {
	if left.Type.Kind != FormulaValueNumber || right.Type.Kind != FormulaValueNumber {
		return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "%s requires number operands", operator)
	}
	switch operator {
	case "+", "-":
		leftNumber, err := formulaNumberInUnit(left, resultType.Unit)
		if err != nil {
			return FormulaValue{}, err
		}
		rightNumber, err := formulaNumberInUnit(right, resultType.Unit)
		if err != nil {
			return FormulaValue{}, err
		}
		if operator == "+" {
			return checkedFormulaNumber(resultType, leftNumber+rightNumber), nil
		}
		return checkedFormulaNumber(resultType, leftNumber-rightNumber), nil
	case "*":
		return checkedFormulaNumber(resultType, left.Number*right.Number), nil
	case "/":
		if right.Number == 0 {
			return FormulaValue{Type: resultType, Missing: true}, nil
		}
		if left.Type.Unit != "" && right.Type.Unit != "" {
			rightNumber, err := formulaNumberInUnit(right, left.Type.Unit)
			if err != nil {
				return FormulaValue{}, err
			}
			if rightNumber == 0 {
				return FormulaValue{Type: resultType, Missing: true}, nil
			}
			return checkedFormulaNumber(resultType, left.Number/rightNumber), nil
		}
		return checkedFormulaNumber(resultType, left.Number/right.Number), nil
	default:
		return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "unsupported arithmetic operator %q", operator)
	}
}

func evaluateTypedComparison(operator string, left, right FormulaValue) (FormulaValue, error) {
	if left.Type.Kind != FormulaValueNumber || right.Type.Kind != FormulaValueNumber {
		return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "%s requires number operands", operator)
	}
	targetUnit := left.Type.Unit
	if targetUnit == "" {
		targetUnit = right.Type.Unit
	}
	leftNumber, err := formulaNumberInUnit(left, targetUnit)
	if err != nil {
		return FormulaValue{}, err
	}
	rightNumber, err := formulaNumberInUnit(right, targetUnit)
	if err != nil {
		return FormulaValue{}, err
	}
	var result bool
	switch operator {
	case ">":
		result = leftNumber > rightNumber
	case ">=":
		result = leftNumber >= rightNumber
	case "<":
		result = leftNumber < rightNumber
	case "<=":
		result = leftNumber <= rightNumber
	case "=":
		result = leftNumber == rightNumber
	case "!=":
		result = leftNumber != rightNumber
	default:
		return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "unsupported comparison operator %q", operator)
	}
	return FormulaValue{Type: FormulaStaticType{Kind: FormulaValueBool}, Bool: result}, nil
}

func normalizeFormulaValue(value FormulaValue, target FormulaStaticType) (FormulaValue, error) {
	if value.Missing || math.IsNaN(value.Number) || math.IsInf(value.Number, 0) {
		return FormulaValue{Type: target, Missing: true}, nil
	}
	if target.Unit == "" {
		if value.Type.Unit != "" {
			return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "formula value unit %q does not match dimensionless input", value.Type.Unit)
		}
		value.Type = target
		return value, nil
	}
	if value.Type.Unit == "" {
		return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "formula value is missing required unit %q", target.Unit)
	}
	if !units.AreCompatible(units.Unit(value.Type.Unit), units.Unit(target.Unit)) {
		return FormulaValue{}, newError(ErrorInvalidFormula, "formula.expression", "formula value unit %q is incompatible with %q", value.Type.Unit, target.Unit)
	}
	value.Number = units.ConverterFromUnit(units.Unit(value.Type.Unit)).Convert(units.Value{F: value.Number, U: units.Unit(value.Type.Unit)}, units.Unit(target.Unit)).F
	value.Type = target
	return value, nil
}

func formulaNumberInUnit(value FormulaValue, targetUnit string) (float64, error) {
	if value.Type.Kind != FormulaValueNumber {
		return 0, newError(ErrorInvalidFormula, "formula.expression", "formula value is not numeric")
	}
	if targetUnit == "" || value.Type.Unit == "" || value.Type.Unit == targetUnit {
		return value.Number, nil
	}
	if !units.AreCompatible(units.Unit(value.Type.Unit), units.Unit(targetUnit)) {
		return 0, newError(ErrorInvalidFormula, "formula.expression", "formula value unit %q is incompatible with %q", value.Type.Unit, targetUnit)
	}
	return units.ConverterFromUnit(units.Unit(value.Type.Unit)).Convert(units.Value{F: value.Number, U: units.Unit(value.Type.Unit)}, units.Unit(targetUnit)).F, nil
}

func checkedFormulaNumber(typ FormulaStaticType, value float64) FormulaValue {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return FormulaValue{Type: typ, Missing: true}
	}
	return FormulaValue{Type: typ, Number: value}
}

func formatTypedFormula(node typedFormulaNode, parentPrecedence int) string {
	precedence := typedFormulaPrecedence(node)
	var text string
	switch current := node.(type) {
	case *typedFormulaNumberNode:
		text = strconv.FormatFloat(current.value, 'f', -1, 64)
	case *typedFormulaReferenceNode:
		text = current.name
	case *typedFormulaUnaryNode:
		text = current.op + " " + formatTypedFormula(current.value, precedence)
		if current.op == "+" || current.op == "-" {
			text = current.op + formatTypedFormula(current.value, precedence)
		}
	case *typedFormulaBinaryNode:
		left := formatTypedFormula(current.left, precedence)
		rightPrecedence := precedence
		if current.op == "-" || current.op == "/" || isTypedFormulaComparison(current.op) {
			rightPrecedence++
		}
		right := formatTypedFormula(current.right, rightPrecedence)
		text = left + " " + current.op + " " + right
	case *typedFormulaCallNode:
		args := make([]string, 0, len(current.args))
		for _, arg := range current.args {
			args = append(args, formatTypedFormula(arg, 0))
		}
		text = current.name + "(" + strings.Join(args, ", ") + ")"
	}
	if precedence < parentPrecedence {
		return "(" + text + ")"
	}
	return text
}

func typedFormulaPrecedence(node typedFormulaNode) int {
	switch current := node.(type) {
	case *typedFormulaBinaryNode:
		switch current.op {
		case "OR":
			return 1
		case "AND":
			return 2
		case ">", ">=", "<", "<=", "=", "!=":
			return 4
		case "+", "-":
			return 5
		case "*", "/":
			return 6
		}
	case *typedFormulaUnaryNode:
		return 7
	}
	return 8
}

func isTypedFormulaComparison(operator string) bool {
	switch operator {
	case ">", ">=", "<", "<=", "=", "!=":
		return true
	default:
		return false
	}
}

func distinctFormulaReferences(references []string) []string {
	result := make([]string, 0, len(references))
	seen := make(map[string]struct{}, len(references))
	for _, reference := range references {
		if _, ok := seen[reference]; ok {
			continue
		}
		seen[reference] = struct{}{}
		result = append(result, reference)
	}
	return result
}
