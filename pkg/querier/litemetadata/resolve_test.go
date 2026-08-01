package litemetadata

import (
	"context"
	"reflect"
	"testing"

	"github.com/SigNoz/signoz/pkg/types/metrictypes"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
)

func TestUnresolvedMetricNamesDeduplicatesOnlyRequiredLookups(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{
		{Type: qbtypes.QueryTypeBuilder, Spec: qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]{Aggregations: []qbtypes.MetricAggregation{
			{MetricName: "requests", Type: metrictypes.UnspecifiedType, Temporality: metrictypes.Unknown},
			{MetricName: "requests", Type: metrictypes.UnspecifiedType, Temporality: metrictypes.Unknown},
			{MetricName: "cpu", Type: metrictypes.GaugeType, Temporality: metrictypes.Unknown},
			{MetricName: "complete", Type: metrictypes.GaugeType, Temporality: metrictypes.Unspecified},
		}}},
	}}}
	want := []string{"requests"}
	if got := unresolvedMetricNames(request); !reflect.DeepEqual(got, want) {
		t.Fatalf("unresolvedMetricNames() = %#v, want %#v", got, want)
	}
}

func TestUnresolvedMetricNamesOnlyNeedsTemporalityForSums(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{
		{Type: qbtypes.QueryTypeBuilder, Spec: qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]{Aggregations: []qbtypes.MetricAggregation{
			{MetricName: "gauge", Type: metrictypes.GaugeType, Temporality: metrictypes.Unknown},
			{MetricName: "histogram", Type: metrictypes.HistogramType, Temporality: metrictypes.Unknown},
			{MetricName: "sum", Type: metrictypes.SumType, Temporality: metrictypes.Unknown},
		}}},
	}}}
	if got, want := unresolvedMetricNames(request), []string{"histogram", "sum"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("unresolvedMetricNames() = %#v, want %#v", got, want)
	}
}

func TestUnresolvedMetricNamesSkipsDisabledQueries(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{CompositeQuery: qbtypes.CompositeQuery{Queries: []qbtypes.QueryEnvelope{
		{Type: qbtypes.QueryTypeBuilder, Spec: qbtypes.QueryBuilderQuery[qbtypes.MetricAggregation]{
			Disabled: true,
			Aggregations: []qbtypes.MetricAggregation{{
				MetricName: "retired_metric", Type: metrictypes.SumType, Temporality: metrictypes.Unknown,
			}},
		}},
	}}}
	if got := unresolvedMetricNames(request); len(got) != 0 {
		t.Fatalf("unresolvedMetricNames() = %#v, want disabled query ignored", got)
	}
}

func TestResolveDoesNotRequireStoreWhenRequestNeedsNoMetadata(t *testing.T) {
	request := &qbtypes.QueryRangeRequest{Start: 1, End: 2}
	metadata, err := Resolve(context.Background(), nil, request)
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if metadata.Temporality != nil || metadata.Types != nil || metadata.FieldKeys != nil {
		t.Fatalf("Resolve() metadata = %#v, want empty", metadata)
	}
}
