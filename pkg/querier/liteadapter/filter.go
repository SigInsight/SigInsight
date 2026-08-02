package liteadapter

import (
	"strconv"
	"strings"

	"github.com/SigNoz/signoz/pkg/litequery"
	grammar "github.com/SigNoz/signoz/pkg/parser/grammar"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/antlr4-go/antlr/v4"
)

// ParseFilter uses the existing V5 grammar but emits the restricted filter
// AST. This keeps the public syntax stable without inheriting its SQL visitor.
// Callers outside the V5 adapter use it for small, signal-specific readers
// whose public filter syntax predates the lightweight engine.
func ParseFilter(expression string, signal litequery.Signal) (litequery.FilterNode, error) {
	return parseFilter(expression, signal, MetricMetadata{})
}

// ParseFilterWithMetadata resolves unqualified telemetry fields before
// emitting the lightweight filter AST. Callers should provide metadata for
// user-entered log and trace filters, where a name can refer to either a
// resource or record attribute.
func ParseFilterWithMetadata(expression string, signal litequery.Signal, fieldKeys map[string][]*telemetrytypes.TelemetryFieldKey) (litequery.FilterNode, error) {
	return parseFilter(expression, signal, MetricMetadata{FieldKeys: fieldKeys})
}

func parseFilter(expression string, signal litequery.Signal, metadata MetricMetadata) (litequery.FilterNode, error) {
	input := antlr.NewInputStream(expression)
	lexer := grammar.NewFilterQueryLexer(input)
	listener := &syntaxErrors{}
	lexer.RemoveErrorListeners()
	lexer.AddErrorListener(listener)
	tokens := antlr.NewCommonTokenStream(lexer, 0)
	parser := grammar.NewFilterQueryParser(tokens)
	parser.RemoveErrorListeners()
	parser.AddErrorListener(listener)
	tree := parser.Query()
	if len(listener.errors) != 0 {
		return nil, unsupported("invalid filter syntax")
	}
	return parseOr(tree.Expression().OrExpression(), signal, metadata)
}

type syntaxErrors struct {
	*antlr.DefaultErrorListener
	errors []string
}

func (s *syntaxErrors) SyntaxError(_ antlr.Recognizer, _ interface{}, _ int, _ int, message string, _ antlr.RecognitionException) {
	s.errors = append(s.errors, message)
}

func parseOr(context grammar.IOrExpressionContext, signal litequery.Signal, metadata MetricMetadata) (litequery.FilterNode, error) {
	items := context.AllAndExpression()
	parsed := make([]litequery.FilterNode, 0, len(items))
	for _, item := range items {
		node, err := parseAnd(item, signal, metadata)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, node)
	}
	return combine(litequery.BooleanOr, parsed), nil
}

func parseAnd(context grammar.IAndExpressionContext, signal litequery.Signal, metadata MetricMetadata) (litequery.FilterNode, error) {
	items := context.AllUnaryExpression()
	parsed := make([]litequery.FilterNode, 0, len(items))
	for _, item := range items {
		node, err := parseUnary(item, signal, metadata)
		if err != nil {
			return nil, err
		}
		parsed = append(parsed, node)
	}
	return combine(litequery.BooleanAnd, parsed), nil
}

func combine(operator litequery.BooleanOperator, items []litequery.FilterNode) litequery.FilterNode {
	if len(items) == 1 {
		return items[0]
	}
	return litequery.LogicalFilter{Operator: operator, Items: items}
}

func parseUnary(context grammar.IUnaryExpressionContext, signal litequery.Signal, metadata MetricMetadata) (litequery.FilterNode, error) {
	if context.NOT() != nil {
		return nil, unsupported("unary NOT filter")
	}
	return parsePrimary(context.Primary(), signal, metadata)
}

func parsePrimary(context grammar.IPrimaryContext, signal litequery.Signal, metadata MetricMetadata) (litequery.FilterNode, error) {
	if context.OrExpression() != nil {
		return parseOr(context.OrExpression(), signal, metadata)
	}
	if context.Comparison() != nil {
		return parseComparison(context.Comparison(), signal, metadata)
	}
	return nil, unsupported("full-text or function filter")
}

func parseComparison(context grammar.IComparisonContext, signal litequery.Signal, metadata MetricMetadata) (litequery.FilterNode, error) {
	if context.BETWEEN() != nil || context.LIKE() != nil || context.ILIKE() != nil || context.REGEXP() != nil {
		return nil, unsupported("BETWEEN, LIKE, ILIKE, or REGEXP filter")
	}
	if context.CONTAINS() != nil && context.NOT() != nil {
		return nil, unsupported("NOT CONTAINS filter")
	}
	if context.NOT() != nil && context.EXISTS() == nil && context.NotInClause() == nil {
		return nil, unsupported("negative filter operator")
	}

	op, value, fallbackType, err := comparisonOperatorAndValue(context)
	if err != nil {
		return nil, err
	}
	field, err := textFieldToLite(context.Key().GetText(), signal, fallbackType, metadata)
	if err != nil {
		return nil, err
	}
	return litequery.Predicate{Field: field, Op: op, Value: value}, nil
}

func comparisonOperatorAndValue(context grammar.IComparisonContext) (litequery.FilterOperator, litequery.Value, litequery.ValueType, error) {
	if context.EXISTS() != nil {
		if context.NOT() != nil {
			return litequery.FilterNotExists, litequery.Value{Kind: litequery.ValueNone}, litequery.ValueTypeString, nil
		}
		return litequery.FilterExists, litequery.Value{Kind: litequery.ValueNone}, litequery.ValueTypeString, nil
	}
	if context.InClause() != nil || context.NotInClause() != nil {
		var clause interface {
			ValueList() grammar.IValueListContext
			Value() grammar.IValueContext
		}
		clause = context.InClause()
		op := litequery.FilterIn
		if clause == nil {
			clause = context.NotInClause()
			op = litequery.FilterNotIn
		}
		value, fallbackType, err := parseList(clause)
		if err != nil {
			return "", litequery.Value{}, "", err
		}
		return op, value, fallbackType, nil
	}
	values := context.AllValue()
	if len(values) != 1 {
		return "", litequery.Value{}, "", unsupported("filter with multiple values")
	}
	value, fallbackType, err := parseValue(values[0])
	if err != nil {
		return "", litequery.Value{}, "", err
	}
	switch {
	case context.EQUALS() != nil:
		return litequery.FilterEqual, value, fallbackType, nil
	case context.NOT_EQUALS() != nil || context.NEQ() != nil:
		return litequery.FilterNotEqual, value, fallbackType, nil
	case context.LT() != nil:
		return litequery.FilterLessThan, value, fallbackType, nil
	case context.LE() != nil:
		return litequery.FilterLessEq, value, fallbackType, nil
	case context.GT() != nil:
		return litequery.FilterGreaterThan, value, fallbackType, nil
	case context.GE() != nil:
		return litequery.FilterGreaterEq, value, fallbackType, nil
	case context.CONTAINS() != nil:
		return litequery.FilterContains, value, fallbackType, nil
	default:
		return "", litequery.Value{}, "", unsupported("filter operator")
	}
}

func parseList(context interface {
	ValueList() grammar.IValueListContext
	Value() grammar.IValueContext
}) (litequery.Value, litequery.ValueType, error) {
	var values []grammar.IValueContext
	if list := context.ValueList(); list != nil {
		values = list.AllValue()
	} else if value := context.Value(); value != nil {
		values = []grammar.IValueContext{value}
	}
	if len(values) == 0 {
		return litequery.Value{}, "", unsupported("empty IN filter")
	}
	result := litequery.Value{}
	var resultType litequery.ValueType
	for _, value := range values {
		parsed, currentType, err := parseValue(value)
		if err != nil {
			return litequery.Value{}, "", err
		}
		if resultType == "" {
			resultType = currentType
		} else if currentType != resultType {
			return litequery.Value{}, "", unsupported("mixed-type IN filter")
		}
		switch currentType {
		case litequery.ValueTypeString:
			result.Kind = litequery.ValueStringList
			result.Strings = append(result.Strings, parsed.String)
		case litequery.ValueTypeNumber:
			result.Kind = litequery.ValueNumberList
			result.Numbers = append(result.Numbers, parsed.Number)
		case litequery.ValueTypeBool:
			result.Kind = litequery.ValueBoolList
			result.Bools = append(result.Bools, parsed.Bool)
		}
	}
	return result, resultType, nil
}

func parseValue(context grammar.IValueContext) (litequery.Value, litequery.ValueType, error) {
	if token := context.QUOTED_TEXT(); token != nil {
		raw := token.GetText()
		var value string
		if len(raw) >= 2 && raw[0] == '\'' && raw[len(raw)-1] == '\'' {
			value = strings.ReplaceAll(raw[1:len(raw)-1], "\\'", "'")
		} else {
			var err error
			value, err = strconv.Unquote(raw)
			if err != nil {
				return litequery.Value{}, "", unsupported("invalid quoted filter value")
			}
		}
		return litequery.Value{Kind: litequery.ValueString, String: value}, litequery.ValueTypeString, nil
	}
	if token := context.NUMBER(); token != nil {
		value, err := strconv.ParseFloat(token.GetText(), 64)
		if err != nil {
			return litequery.Value{}, "", unsupported("invalid numeric filter value")
		}
		return litequery.Value{Kind: litequery.ValueNumber, Number: value}, litequery.ValueTypeNumber, nil
	}
	if token := context.BOOL(); token != nil {
		return litequery.Value{Kind: litequery.ValueBool, Bool: strings.EqualFold(token.GetText(), "true")}, litequery.ValueTypeBool, nil
	}
	return litequery.Value{}, "", unsupported("variable filter value")
}

func textFieldToLite(text string, signal litequery.Signal, fallback litequery.ValueType, metadata MetricMetadata) (litequery.FieldRef, error) {
	key := telemetrytypes.GetFieldKeyFromKeyText(text)
	return fieldToLite(key, signal, fallback, metadata)
}

func fieldToLite(key telemetrytypes.TelemetryFieldKey, signal litequery.Signal, fallback litequery.ValueType, metadata MetricMetadata) (litequery.FieldRef, error) {
	key.Normalize()
	originalContext := key.FieldContext
	// Intrinsic fields belong to the static signal schema, so they must be
	// resolved before consulting telemetry metadata. Saved V5 filters may carry
	// an explicit data type (for example name:string); that type annotation
	// should validate the intrinsic rather than make it look like a dynamic
	// attribute that must have been observed in the metadata registry.
	if context, err := fieldContext(key.FieldContext, signal); err == nil {
		if _, isTraceScope := litequery.TraceScopeForName(signal, context, key.Name); isTraceScope {
			if key.FieldDataType != telemetrytypes.FieldDataTypeUnspecified {
				declaredType, typeErr := fieldType(key.FieldDataType, fallback)
				if typeErr != nil {
					return litequery.FieldRef{}, typeErr
				}
				if declaredType != litequery.ValueTypeBool {
					return litequery.FieldRef{}, unsupported("trace scope field " + strconv.Quote(key.Name) + " has type bool, not " + string(declaredType))
				}
			}
			return litequery.FieldRef{Name: key.Name, Context: context, Type: litequery.ValueTypeBool}, nil
		}
		if intrinsicType, ok := litequery.IntrinsicFieldType(signal, context, key.Name); ok {
			if key.FieldDataType != telemetrytypes.FieldDataTypeUnspecified {
				declaredType, typeErr := fieldType(key.FieldDataType, fallback)
				if typeErr != nil {
					return litequery.FieldRef{}, typeErr
				}
				if declaredType != intrinsicType {
					return litequery.FieldRef{}, unsupported("intrinsic field " + strconv.Quote(key.Name) + " has type " + string(intrinsicType) + ", not " + string(declaredType))
				}
			}
			return litequery.FieldRef{Name: key.Name, Context: context, Type: intrinsicType}, nil
		}
	}
	resolvedFromMetadata := false
	inferredFromSchema := false
	if resolved, ok := resolveFieldMetadata(key, signal, fallback, metadata.FieldKeys[key.Name]); ok {
		key = resolved
		resolvedFromMetadata = true
	}
	// Collector stores resource values in a string-only map, while number and
	// bool values are stored only in record attributes. This makes an omitted
	// context deterministic even when a key registry has not observed the field.
	if key.FieldContext == telemetrytypes.FieldContextUnspecified && (fallback == litequery.ValueTypeNumber || fallback == litequery.ValueTypeBool) {
		key.FieldContext = telemetrytypes.FieldContextAttribute
		inferredFromSchema = true
	}
	context, err := fieldContext(key.FieldContext, signal)
	if err != nil {
		return litequery.FieldRef{}, err
	}
	if originalContext == telemetrytypes.FieldContextUnspecified && metadata.FieldKeys != nil && !resolvedFromMetadata && !inferredFromSchema {
		return litequery.FieldRef{}, unsupported("unqualified " + string(signal) + " field " + strconv.Quote(key.Name) + " was not found in telemetry metadata; qualify it as resource or attribute")
	}
	fieldType, err := fieldType(key.FieldDataType, fallback)
	if err != nil {
		return litequery.FieldRef{}, err
	}
	return litequery.FieldRef{Name: key.Name, Context: context, Type: fieldType}, nil
}

func resolveFieldMetadata(key telemetrytypes.TelemetryFieldKey, signal litequery.Signal, fallback litequery.ValueType, candidates []*telemetrytypes.TelemetryFieldKey) (telemetrytypes.TelemetryFieldKey, bool) {
	matches := make([]telemetrytypes.TelemetryFieldKey, 0, len(candidates))
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		if candidate == nil || (candidate.Signal != telemetrytypes.SignalUnspecified && candidate.Signal.StringValue() != string(signal)) {
			continue
		}
		current := *candidate
		current.Normalize()
		if key.FieldContext != telemetrytypes.FieldContextUnspecified && current.FieldContext != key.FieldContext {
			continue
		}
		if key.FieldDataType != telemetrytypes.FieldDataTypeUnspecified {
			requestedType, requestedErr := fieldType(key.FieldDataType, fallback)
			candidateType, candidateErr := fieldType(current.FieldDataType, fallback)
			if requestedErr != nil || candidateErr != nil || requestedType != candidateType {
				continue
			}
		}
		identity := current.FieldContext.StringValue() + ";" + current.FieldDataType.StringValue()
		if _, ok := seen[identity]; ok {
			continue
		}
		seen[identity] = struct{}{}
		matches = append(matches, current)
	}
	if len(matches) == 0 {
		return telemetrytypes.TelemetryFieldKey{}, false
	}

	if key.FieldDataType == telemetrytypes.FieldDataTypeUnspecified {
		typeMatches := matches[:0]
		for _, candidate := range matches {
			candidateType, err := fieldType(candidate.FieldDataType, fallback)
			if err == nil && candidateType == fallback {
				typeMatches = append(typeMatches, candidate)
			}
		}
		if len(typeMatches) != 0 {
			matches = typeMatches
		}
	}
	// Preserve the established V5 resource preference only after applying the
	// operator's value type. A numeric comparison cannot target the Collector's
	// string-only resource map when a numeric attribute with the same name is
	// available.
	if key.FieldContext == telemetrytypes.FieldContextUnspecified {
		resourceMatches := matches[:0]
		for _, candidate := range matches {
			if candidate.FieldContext == telemetrytypes.FieldContextResource {
				resourceMatches = append(resourceMatches, candidate)
			}
		}
		if len(resourceMatches) != 0 {
			matches = resourceMatches
		}
	}
	if len(matches) != 1 {
		return telemetrytypes.TelemetryFieldKey{}, false
	}
	key.OverrideMetadataFrom(&matches[0])
	return key, true
}

func fieldContext(context telemetrytypes.FieldContext, signal litequery.Signal) (litequery.FieldContext, error) {
	if context == telemetrytypes.FieldContextUnspecified {
		switch signal {
		case litequery.SignalLogs:
			return litequery.FieldContextLog, nil
		case litequery.SignalTraces:
			return litequery.FieldContextSpan, nil
		case litequery.SignalMetrics, litequery.SignalMeter:
			return litequery.FieldContextLabel, nil
		}
	}
	switch context {
	case telemetrytypes.FieldContextResource:
		return litequery.FieldContextResource, nil
	case telemetrytypes.FieldContextAttribute:
		return litequery.FieldContextAttribute, nil
	case telemetrytypes.FieldContextSpan, telemetrytypes.FieldContextTrace:
		return litequery.FieldContextSpan, nil
	case telemetrytypes.FieldContextLog:
		return litequery.FieldContextLog, nil
	case telemetrytypes.FieldContextBody:
		return litequery.FieldContextBody, nil
	case telemetrytypes.FieldContextScope:
		return litequery.FieldContextScope, nil
	case telemetrytypes.FieldContextMetric:
		return litequery.FieldContextMetric, nil
	default:
		return "", unsupported("field context " + context.StringValue())
	}
}

func fieldType(dataType telemetrytypes.FieldDataType, fallback litequery.ValueType) (litequery.ValueType, error) {
	switch dataType {
	case telemetrytypes.FieldDataTypeUnspecified:
		return fallback, nil
	case telemetrytypes.FieldDataTypeString:
		return litequery.ValueTypeString, nil
	case telemetrytypes.FieldDataTypeBool:
		return litequery.ValueTypeBool, nil
	case telemetrytypes.FieldDataTypeNumber, telemetrytypes.FieldDataTypeInt64, telemetrytypes.FieldDataTypeFloat64:
		return litequery.ValueTypeNumber, nil
	default:
		return "", unsupported("field type " + dataType.StringValue())
	}
}
