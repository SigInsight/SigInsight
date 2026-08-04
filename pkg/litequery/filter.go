package litequery

import (
	"regexp"
	"strings"
)

type BooleanOperator string

const (
	BooleanAnd BooleanOperator = "and"
	BooleanOr  BooleanOperator = "or"
)

type FilterOperator string

const (
	FilterEqual       FilterOperator = "eq"
	FilterNotEqual    FilterOperator = "neq"
	FilterGreaterThan FilterOperator = "gt"
	FilterGreaterEq   FilterOperator = "gte"
	FilterLessThan    FilterOperator = "lt"
	FilterLessEq      FilterOperator = "lte"
	FilterIn          FilterOperator = "in"
	FilterNotIn       FilterOperator = "not_in"
	FilterExists      FilterOperator = "exists"
	FilterNotExists   FilterOperator = "not_exists"
	FilterContains    FilterOperator = "contains"
	FilterNotContains FilterOperator = "not_contains"
	FilterLike        FilterOperator = "like"
	FilterNotLike     FilterOperator = "not_like"
	FilterILike       FilterOperator = "ilike"
	FilterNotILike    FilterOperator = "not_ilike"
	FilterRegexp      FilterOperator = "regexp"
	FilterNotRegexp   FilterOperator = "not_regexp"
)

const maxPatternLength = 1024

type ValueKind string

const (
	ValueNone       ValueKind = "none"
	ValueString     ValueKind = "string"
	ValueNumber     ValueKind = "number"
	ValueBool       ValueKind = "bool"
	ValueStringList ValueKind = "string_list"
	ValueNumberList ValueKind = "number_list"
	ValueBoolList   ValueKind = "bool_list"
)

type Value struct {
	Kind    ValueKind
	String  string
	Number  float64
	Bool    bool
	Strings []string
	Numbers []float64
	Bools   []bool
}

type FilterNode interface {
	filterNode()
}

type LogicalFilter struct {
	Operator BooleanOperator
	Items    []FilterNode
}

func (LogicalFilter) filterNode() {}

type Predicate struct {
	Field FieldRef
	Op    FilterOperator
	Value Value
}

func (Predicate) filterNode() {}

func validateFilter(node FilterNode, depth *int, nodes *int, limits Limits) error {
	if node == nil {
		return nil
	}
	*depth++
	defer func() { *depth-- }()
	if *depth > limits.MaxFilterDepth {
		return newError(ErrorBudgetExceeded, "filter", "filter nesting exceeds %d", limits.MaxFilterDepth)
	}
	*nodes++
	if *nodes > limits.MaxFilterNodes {
		return newError(ErrorBudgetExceeded, "filter", "filter contains more than %d nodes", limits.MaxFilterNodes)
	}

	switch current := node.(type) {
	case LogicalFilter:
		if current.Operator != BooleanAnd && current.Operator != BooleanOr {
			return newError(ErrorInvalidFilter, "filter.operator", "unsupported logical operator %q", current.Operator)
		}
		if len(current.Items) < 2 {
			return newError(ErrorInvalidFilter, "filter.items", "logical filter requires at least two items")
		}
		for _, item := range current.Items {
			if err := validateFilter(item, depth, nodes, limits); err != nil {
				return err
			}
		}
		return nil
	case Predicate:
		return validatePredicate(current)
	default:
		return newError(ErrorInvalidFilter, "filter", "unsupported filter node %T", node)
	}
}

func validatePredicate(p Predicate) error {
	if err := validateField(p.Field, "filter.field"); err != nil {
		return err
	}
	switch p.Op {
	case FilterEqual, FilterNotEqual, FilterGreaterThan, FilterGreaterEq, FilterLessThan, FilterLessEq:
		if p.Value.Kind == ValueNone || isListValue(p.Value.Kind) {
			return newError(ErrorInvalidFilter, "filter.value", "%s requires a scalar value", p.Op)
		}
		if !valueMatchesField(p.Value.Kind, p.Field.Type) {
			return newError(ErrorInvalidFilter, "filter.value", "%s value does not match %s field", p.Value.Kind, p.Field.Type)
		}
		if p.Value.Kind == ValueNumber && !finite(p.Value.Number) {
			return newError(ErrorInvalidFilter, "filter.value", "numeric filter value must be finite")
		}
	case FilterIn, FilterNotIn:
		if !listMatchesField(p.Value, p.Field.Type) {
			return newError(ErrorInvalidFilter, "filter.value", "%s requires a non-empty homogeneous list matching the field type", p.Op)
		}
		for _, value := range p.Value.Numbers {
			if !finite(value) {
				return newError(ErrorInvalidFilter, "filter.value", "numeric filter list values must be finite")
			}
		}
	case FilterExists, FilterNotExists:
		if p.Value.Kind != ValueNone {
			return newError(ErrorInvalidFilter, "filter.value", "%s does not accept a value", p.Op)
		}
	case FilterContains, FilterNotContains, FilterLike, FilterNotLike, FilterILike, FilterNotILike, FilterRegexp, FilterNotRegexp:
		if p.Field.Type != ValueTypeString || p.Value.Kind != ValueString || strings.TrimSpace(p.Value.String) == "" {
			return newError(ErrorInvalidFilter, "filter", "%s requires a non-empty string value and string field", p.Op)
		}
		if len(p.Value.String) > maxPatternLength {
			return newError(ErrorBudgetExceeded, "filter.value", "%s pattern exceeds %d bytes", p.Op, maxPatternLength)
		}
		if p.Op == FilterRegexp || p.Op == FilterNotRegexp {
			if _, err := regexp.Compile(p.Value.String); err != nil {
				return newError(ErrorInvalidFilter, "filter.value", "invalid RE2-compatible regular expression")
			}
		}
	default:
		return newError(ErrorInvalidFilter, "filter.operator", "unsupported filter operator %q", p.Op)
	}
	return nil
}

func isListValue(kind ValueKind) bool {
	return kind == ValueStringList || kind == ValueNumberList || kind == ValueBoolList
}

func listMatchesField(value Value, fieldType ValueType) bool {
	return (value.Kind == ValueStringList && fieldType == ValueTypeString && len(value.Strings) != 0) ||
		(value.Kind == ValueNumberList && fieldType == ValueTypeNumber && len(value.Numbers) != 0) ||
		(value.Kind == ValueBoolList && fieldType == ValueTypeBool && len(value.Bools) != 0)
}

func valueMatchesField(kind ValueKind, fieldType ValueType) bool {
	return (kind == ValueString && fieldType == ValueTypeString) ||
		(kind == ValueNumber && fieldType == ValueTypeNumber) ||
		(kind == ValueBool && fieldType == ValueTypeBool)
}
