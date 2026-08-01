package liteadapter

import (
	"strings"

	grammar "github.com/SigNoz/signoz/pkg/parser/grammar"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/antlr4-go/antlr/v4"
)

// FieldKeySelectors returns exact metadata lookups needed to resolve fields
// whose context or data type was omitted from the public V5 request.
func FieldKeySelectors(request *qbtypes.QueryRangeRequest) []*telemetrytypes.FieldKeySelector {
	if request == nil {
		return nil
	}
	selectors := make([]*telemetrytypes.FieldKeySelector, 0)
	for _, envelope := range request.CompositeQuery.Queries {
		if envelope.Type != qbtypes.QueryTypeBuilder {
			continue
		}
		switch query := envelope.Spec.(type) {
		case qbtypes.QueryBuilderQuery[qbtypes.LogAggregation]:
			selectors = appendBuilderSelectors(selectors, query, telemetrytypes.SignalLogs, request.Start, request.End)
			for _, aggregation := range query.Aggregations {
				if name, ok := logAggregationField(aggregation.Expression); ok {
					selectors = appendSelector(selectors, telemetrytypes.TelemetryFieldKey{Name: name}, telemetrytypes.SignalLogs, request.Start, request.End)
				}
			}
		case qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]:
			selectors = appendBuilderSelectors(selectors, query, telemetrytypes.SignalTraces, request.Start, request.End)
		}
	}
	return deduplicateSelectors(selectors)
}

// FilterFieldKeySelectors returns the exact metadata lookups needed to resolve
// an independently supplied filter expression, such as a live-log filter.
func FilterFieldKeySelectors(expression string, signal telemetrytypes.Signal, start, end uint64) []*telemetrytypes.FieldKeySelector {
	return deduplicateSelectors(filterFieldSelectors(expression, signal, start, end))
}

func appendBuilderSelectors[T any](selectors []*telemetrytypes.FieldKeySelector, query qbtypes.QueryBuilderQuery[T], signal telemetrytypes.Signal, start, end uint64) []*telemetrytypes.FieldKeySelector {
	if query.Filter != nil {
		selectors = append(selectors, filterFieldSelectors(query.Filter.Expression, signal, start, end)...)
	}
	for _, key := range query.SelectFields {
		selectors = appendSelector(selectors, key, signal, start, end)
	}
	for _, group := range query.GroupBy {
		selectors = appendSelector(selectors, group.TelemetryFieldKey, signal, start, end)
	}
	for _, order := range query.Order {
		selectors = appendSelector(selectors, order.Key.TelemetryFieldKey, signal, start, end)
	}
	return selectors
}

func filterFieldSelectors(expression string, signal telemetrytypes.Signal, start, end uint64) []*telemetrytypes.FieldKeySelector {
	lexer := grammar.NewFilterQueryLexer(antlr.NewInputStream(expression))
	selectors := make([]*telemetrytypes.FieldKeySelector, 0)
	for token := lexer.NextToken(); token.GetTokenType() != antlr.TokenEOF; token = lexer.NextToken() {
		if token.GetTokenType() != grammar.FilterQueryLexerKEY {
			continue
		}
		selectors = appendSelector(selectors, telemetrytypes.GetFieldKeyFromKeyText(token.GetText()), signal, start, end)
	}
	return selectors
}

func appendSelector(selectors []*telemetrytypes.FieldKeySelector, key telemetrytypes.TelemetryFieldKey, signal telemetrytypes.Signal, start, end uint64) []*telemetrytypes.FieldKeySelector {
	key.Normalize()
	if key.Name == "" || (key.FieldContext != telemetrytypes.FieldContextUnspecified && key.FieldDataType != telemetrytypes.FieldDataTypeUnspecified) {
		return selectors
	}
	return append(selectors, &telemetrytypes.FieldKeySelector{
		StartUnixMilli:    int64(start),
		EndUnixMilli:      int64(end),
		Signal:            signal,
		FieldContext:      key.FieldContext,
		FieldDataType:     key.FieldDataType,
		Name:              key.Name,
		SelectorMatchType: telemetrytypes.FieldSelectorMatchTypeExact,
		Limit:             20,
	})
}

func deduplicateSelectors(selectors []*telemetrytypes.FieldKeySelector) []*telemetrytypes.FieldKeySelector {
	result := make([]*telemetrytypes.FieldKeySelector, 0, len(selectors))
	seen := map[string]struct{}{}
	for _, selector := range selectors {
		identity := selector.Signal.StringValue() + ";" + selector.FieldContext.StringValue() + ";" + selector.FieldDataType.StringValue() + ";" + selector.Name
		if _, ok := seen[identity]; ok {
			continue
		}
		seen[identity] = struct{}{}
		result = append(result, selector)
	}
	return result
}

func logAggregationField(expression string) (string, bool) {
	expression = strings.TrimSpace(expression)
	lower := strings.ToLower(expression)
	for _, prefix := range []string{"sum(", "avg(", "min(", "max("} {
		if strings.HasPrefix(lower, prefix) && strings.HasSuffix(expression, ")") {
			return strings.TrimSpace(expression[len(prefix) : len(expression)-1]), true
		}
	}
	return "", false
}
