package litequery

import (
	"strconv"
	"strings"
	"unicode"
)

type formulaTokenKind uint8

const (
	formulaIdentifier formulaTokenKind = iota
	formulaNumber
	formulaOperator
	formulaLeftParen
	formulaRightParen
)

type formulaToken struct {
	kind  formulaTokenKind
	value string
}

func validateFormulas(formulas []Formula, names map[string]struct{}) error {
	formulaRefs := make(map[string][]string, len(formulas))
	formulaTokens := make(map[string][]formulaToken, len(formulas))
	for _, formula := range formulas {
		if !validName(formula.Name) {
			return newError(ErrorInvalidFormula, "formula.name", "formula name must start with a letter and contain only letters, digits, or underscores")
		}
		if err := addName(names, formula.Name, "formulas"); err != nil {
			return err
		}
		tokens, err := tokenizeFormula(formula.Expression)
		if err != nil {
			return err
		}
		if err := validateFormulaTokens(tokens); err != nil {
			return err
		}
		formulaTokens[formula.Name] = tokens
	}
	for _, formula := range formulas {
		tokens := formulaTokens[formula.Name]
		refs := make([]string, 0)
		for _, token := range tokens {
			if token.kind != formulaIdentifier {
				continue
			}
			if _, exists := names[token.value]; !exists {
				return newError(ErrorInvalidFormula, "formula.expression", "formula %q references unknown query %q", formula.Name, token.value)
			}
			refs = append(refs, token.value)
		}
		formulaRefs[formula.Name] = refs
	}
	return validateFormulaCycles(formulaRefs)
}

func tokenizeFormula(expression string) ([]formulaToken, error) {
	input := strings.TrimSpace(expression)
	if input == "" {
		return nil, newError(ErrorInvalidFormula, "formula.expression", "formula expression is required")
	}
	tokens := make([]formulaToken, 0, len(input)/2)
	for index := 0; index < len(input); {
		r := rune(input[index])
		if unicode.IsSpace(r) {
			index++
			continue
		}
		if unicode.IsLetter(r) || r == '_' {
			start := index
			index++
			for index < len(input) {
				r = rune(input[index])
				if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
					break
				}
				index++
			}
			tokens = append(tokens, formulaToken{kind: formulaIdentifier, value: input[start:index]})
			continue
		}
		if unicode.IsDigit(r) || r == '.' {
			start := index
			index++
			for index < len(input) && (unicode.IsDigit(rune(input[index])) || input[index] == '.') {
				index++
			}
			value := input[start:index]
			if _, err := strconv.ParseFloat(value, 64); err != nil {
				return nil, newError(ErrorInvalidFormula, "formula.expression", "invalid number %q", value)
			}
			tokens = append(tokens, formulaToken{kind: formulaNumber, value: value})
			continue
		}
		switch input[index] {
		case '+', '-', '*', '/':
			tokens = append(tokens, formulaToken{kind: formulaOperator, value: input[index : index+1]})
		case '(':
			tokens = append(tokens, formulaToken{kind: formulaLeftParen, value: "("})
		case ')':
			tokens = append(tokens, formulaToken{kind: formulaRightParen, value: ")"})
		default:
			return nil, newError(ErrorInvalidFormula, "formula.expression", "unsupported token %q", input[index:index+1])
		}
		index++
	}
	return tokens, nil
}

func validateFormulaTokens(tokens []formulaToken) error {
	if len(tokens) == 0 {
		return newError(ErrorInvalidFormula, "formula.expression", "formula expression is required")
	}
	expectsOperand := true
	parentheses := 0
	for index, token := range tokens {
		switch token.kind {
		case formulaIdentifier, formulaNumber:
			if !expectsOperand {
				return newError(ErrorInvalidFormula, "formula.expression", "missing operator before token %q", token.value)
			}
			expectsOperand = false
		case formulaLeftParen:
			if !expectsOperand {
				return newError(ErrorInvalidFormula, "formula.expression", "missing operator before parenthesis")
			}
			parentheses++
		case formulaRightParen:
			if expectsOperand || parentheses == 0 {
				return newError(ErrorInvalidFormula, "formula.expression", "unexpected closing parenthesis")
			}
			parentheses--
		case formulaOperator:
			if expectsOperand {
				return newError(ErrorInvalidFormula, "formula.expression", "operator %q has no left operand", token.value)
			}
			if token.value == "/" && index+1 < len(tokens) && tokens[index+1].kind == formulaNumber && isZero(tokens[index+1].value) {
				return newError(ErrorInvalidFormula, "formula.expression", "division by literal zero")
			}
			expectsOperand = true
		}
	}
	if expectsOperand || parentheses != 0 {
		return newError(ErrorInvalidFormula, "formula.expression", "incomplete formula expression")
	}
	return nil
}

func isZero(value string) bool {
	parsed, err := strconv.ParseFloat(value, 64)
	return err == nil && parsed == 0
}

func validateFormulaCycles(graph map[string][]string) error {
	state := make(map[string]uint8, len(graph))
	var visit func(string) error
	visit = func(name string) error {
		switch state[name] {
		case 1:
			return newError(ErrorInvalidFormula, "formula.expression", "formula dependency cycle includes %q", name)
		case 2:
			return nil
		}
		state[name] = 1
		for _, dependency := range graph[name] {
			if _, isFormula := graph[dependency]; isFormula {
				if err := visit(dependency); err != nil {
					return err
				}
			}
		}
		state[name] = 2
		return nil
	}
	for name := range graph {
		if err := visit(name); err != nil {
			return err
		}
	}
	return nil
}
