package telemetrytraces

import (
	"context"
	"testing"
	"time"

	"github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/instrumentation/instrumentationtest"
	"github.com/SigNoz/signoz/pkg/querybuilder"
	"github.com/SigNoz/signoz/pkg/querybuilder/resourcefilter"
	qbtypes "github.com/SigNoz/signoz/pkg/types/querybuildertypes/querybuildertypesv5"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes/telemetrytypestest"
	"github.com/stretchr/testify/require"
)

func resourceFilterStmtBuilder() qbtypes.StatementBuilder[qbtypes.TraceAggregation] {
	fm := resourcefilter.NewFieldMapper()
	cb := resourcefilter.NewConditionBuilder(fm)
	mockMetadataStore := telemetrytypestest.NewMockMetadataStore()
	mockMetadataStore.KeysMap = buildCompleteFieldKeyMap()

	return resourcefilter.NewTraceResourceFilterStatementBuilder(
		instrumentationtest.New().ToProviderSettings(),
		fm,
		cb,
		mockMetadataStore,
	)
}

func TestStatementBuilder(t *testing.T) {
	cases := []struct {
		name        string
		requestType qbtypes.RequestType
		query       qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]
		expected    qbtypes.Statement
		expectedErr error
	}{
		{
			name:        "test",
			requestType: qbtypes.RequestTypeTimeSeries,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal:       telemetrytypes.SignalTraces,
				StepInterval: qbtypes.Step{Duration: 30 * time.Second},
				Aggregations: []qbtypes.TraceAggregation{
					{
						Expression: "count()",
					},
				},
				Filter: &qbtypes.Filter{
					Expression: "service.name = 'redis-manual'",
				},
				Limit: 10,
				GroupBy: []qbtypes.GroupByKey{
					{
						TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
							Name: "service.name",
						},
					},
				},
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE (simpleJSONExtractString(labels, 'service.name') = ? AND labels LIKE ? AND labels LIKE ?) AND seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?), __limit_cte AS (SELECT toString(multiIf(multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) IS NOT NULL, multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL), NULL)) AS `service.name`, count() AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND true AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? GROUP BY `service.name` ORDER BY __result_0 DESC LIMIT ?) SELECT toStartOfInterval(timestamp, INTERVAL 30 SECOND) AS ts, toString(multiIf(multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) IS NOT NULL, multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL), NULL)) AS `service.name`, count() AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND true AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? AND (`service.name`) GLOBAL IN (SELECT `service.name` FROM __limit_cte) GROUP BY ts, `service.name`",
				Args:  []any{"redis-manual", "%service.name%", "%service.name\":\"redis-manual%", uint64(1747945619), uint64(1747983448), "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10, "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448)},
			},
			expectedErr: nil,
		},
		{
			name:        "OR b/w resource attr and attribute",
			requestType: qbtypes.RequestTypeTimeSeries,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal:       telemetrytypes.SignalTraces,
				StepInterval: qbtypes.Step{Duration: 30 * time.Second},
				Aggregations: []qbtypes.TraceAggregation{
					{
						Expression: "count()",
					},
				},
				Filter: &qbtypes.Filter{
					Expression: "service.name = 'redis-manual' OR http.request.method = 'GET'",
				},
				Limit: 10,
				GroupBy: []qbtypes.GroupByKey{
					{
						TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
							Name: "service.name",
						},
					},
				},
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE ((simpleJSONExtractString(labels, 'service.name') = ? AND labels LIKE ? AND labels LIKE ?) OR true) AND seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?), __limit_cte AS (SELECT toString(multiIf(multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) IS NOT NULL, multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL), NULL)) AS `service.name`, count() AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND ((multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) = ? AND multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) IS NOT NULL) OR (attributes_string['http.request.method'] = ? AND mapContains(attributes_string, 'http.request.method') = ?)) AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? GROUP BY `service.name` ORDER BY __result_0 DESC LIMIT ?) SELECT toStartOfInterval(timestamp, INTERVAL 30 SECOND) AS ts, toString(multiIf(multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) IS NOT NULL, multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL), NULL)) AS `service.name`, count() AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND ((multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) = ? AND multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) IS NOT NULL) OR (attributes_string['http.request.method'] = ? AND mapContains(attributes_string, 'http.request.method') = ?)) AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? AND (`service.name`) GLOBAL IN (SELECT `service.name` FROM __limit_cte) GROUP BY ts, `service.name`",
				Args:  []any{"redis-manual", "%service.name%", "%service.name\":\"redis-manual%", uint64(1747945619), uint64(1747983448), "redis-manual", "GET", true, "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10, "redis-manual", "GET", true, "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448)},
			},
			expectedErr: nil,
		},
		{
			name:        "OR with regexp",
			requestType: qbtypes.RequestTypeTimeSeries,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal:       telemetrytypes.SignalTraces,
				StepInterval: qbtypes.Step{Duration: 30 * time.Second},
				Aggregations: []qbtypes.TraceAggregation{
					{
						Expression: "count()",
					},
				},
				Filter: &qbtypes.Filter{
					Expression: "materialized.key.name REGEXP 'redis-manual' OR service.name = 'redis-manual'",
				},
				Limit: 10,
				GroupBy: []qbtypes.GroupByKey{
					{
						TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
							Name: "service.name",
						},
					},
				},
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE (true OR (simpleJSONExtractString(labels, 'service.name') = ? AND labels LIKE ? AND labels LIKE ?)) AND seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?), __limit_cte AS (SELECT toString(multiIf(multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) IS NOT NULL, multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL), NULL)) AS `service.name`, count() AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND ((match(`attribute_string_materialized$$key$$name`, ?) AND `attribute_string_materialized$$key$$name_exists` = ?) OR (multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) = ? AND multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) IS NOT NULL)) AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? GROUP BY `service.name` ORDER BY __result_0 DESC LIMIT ?) SELECT toStartOfInterval(timestamp, INTERVAL 30 SECOND) AS ts, toString(multiIf(multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) IS NOT NULL, multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL), NULL)) AS `service.name`, count() AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND ((match(`attribute_string_materialized$$key$$name`, ?) AND `attribute_string_materialized$$key$$name_exists` = ?) OR (multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) = ? AND multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) IS NOT NULL)) AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? AND (`service.name`) GLOBAL IN (SELECT `service.name` FROM __limit_cte) GROUP BY ts, `service.name`",
				Args:  []any{"redis-manual", "%service.name%", "%service.name\":\"redis-manual%", uint64(1747945619), uint64(1747983448), "redis-manual", true, "redis-manual", "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10, "redis-manual", true, "redis-manual", "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448)},
			},
			expectedErr: nil,
		},
		{
			name:        "legacy http.route in group by",
			requestType: qbtypes.RequestTypeTimeSeries,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal:       telemetrytypes.SignalTraces,
				StepInterval: qbtypes.Step{Duration: 30 * time.Second},
				Aggregations: []qbtypes.TraceAggregation{
					{
						Expression: "count()",
					},
				},
				Filter: &qbtypes.Filter{
					Expression: "service.name = 'redis-manual'",
				},
				Limit: 10,
				GroupBy: []qbtypes.GroupByKey{
					{
						TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
							Name:          "http.route",
							FieldDataType: telemetrytypes.FieldDataTypeString,
							FieldContext:  telemetrytypes.FieldContextAttribute,
						},
					},
				},
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE (simpleJSONExtractString(labels, 'service.name') = ? AND labels LIKE ? AND labels LIKE ?) AND seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?), __limit_cte AS (SELECT toString(multiIf(attribute_string_http$$route <> ?, attribute_string_http$$route, NULL)) AS `http.route`, count() AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND true AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? GROUP BY `http.route` ORDER BY __result_0 DESC LIMIT ?) SELECT toStartOfInterval(timestamp, INTERVAL 30 SECOND) AS ts, toString(multiIf(attribute_string_http$$route <> ?, attribute_string_http$$route, NULL)) AS `http.route`, count() AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND true AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? AND (`http.route`) GLOBAL IN (SELECT `http.route` FROM __limit_cte) GROUP BY ts, `http.route`",
				Args:  []any{"redis-manual", "%service.name%", "%service.name\":\"redis-manual%", uint64(1747945619), uint64(1747983448), "", "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10, "", "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448)},
			},
			expectedErr: nil,
		},
		{
			name:        "context as key prefix test",
			requestType: qbtypes.RequestTypeTimeSeries,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal:       telemetrytypes.SignalTraces,
				StepInterval: qbtypes.Step{Duration: 30 * time.Second},
				Aggregations: []qbtypes.TraceAggregation{
					{
						Expression: "sum(metric.max_count)",
						Alias:      "metric.max_count",
					},
				},
				Filter: &qbtypes.Filter{
					Expression: "service.name = 'redis-manual'",
				},
				Limit: 10,
				GroupBy: []qbtypes.GroupByKey{
					{
						TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
							Name: "service.name",
						},
					},
				},
				Order: []qbtypes.OrderBy{
					{
						Key: qbtypes.OrderByKey{
							TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
								Name:         "max_count",
								FieldContext: telemetrytypes.FieldContextMetric,
							},
						},
						Direction: qbtypes.OrderDirectionDesc,
					},
				},
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE (simpleJSONExtractString(labels, 'service.name') = ? AND labels LIKE ? AND labels LIKE ?) AND seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?), __limit_cte AS (SELECT toString(multiIf(multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) IS NOT NULL, multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL), NULL)) AS `service.name`, sum(multiIf(mapContains(attributes_number, 'metric.max_count') = ?, toFloat64(attributes_number['metric.max_count']), NULL)) AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND true AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? GROUP BY `service.name` ORDER BY __result_0 desc LIMIT ?) SELECT toStartOfInterval(timestamp, INTERVAL 30 SECOND) AS ts, toString(multiIf(multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) IS NOT NULL, multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL), NULL)) AS `service.name`, sum(multiIf(mapContains(attributes_number, 'metric.max_count') = ?, toFloat64(attributes_number['metric.max_count']), NULL)) AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND true AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? AND (`service.name`) GLOBAL IN (SELECT `service.name` FROM __limit_cte) GROUP BY ts, `service.name` ORDER BY ts desc",
				Args:  []any{"redis-manual", "%service.name%", "%service.name\":\"redis-manual%", uint64(1747945619), uint64(1747983448), true, "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10, true, "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448)},
			},
			expectedErr: nil,
		},
		{
			name:        "mat number key in aggregation test",
			requestType: qbtypes.RequestTypeTimeSeries,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal:       telemetrytypes.SignalTraces,
				StepInterval: qbtypes.Step{Duration: 30 * time.Second},
				Aggregations: []qbtypes.TraceAggregation{
					{
						Expression: "sum(cart.items_count)",
					},
				},
				Filter: &qbtypes.Filter{
					Expression: "service.name = 'redis-manual'",
				},
				Limit: 10,
				GroupBy: []qbtypes.GroupByKey{
					{
						TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
							Name: "service.name",
						},
					},
				},
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE (simpleJSONExtractString(labels, 'service.name') = ? AND labels LIKE ? AND labels LIKE ?) AND seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?), __limit_cte AS (SELECT toString(multiIf(multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) IS NOT NULL, multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL), NULL)) AS `service.name`, sum(multiIf(`attribute_number_cart$$items_count_exists` = ?, toFloat64(`attribute_number_cart$$items_count`), NULL)) AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND true AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? GROUP BY `service.name` ORDER BY __result_0 DESC LIMIT ?) SELECT toStartOfInterval(timestamp, INTERVAL 30 SECOND) AS ts, toString(multiIf(multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) IS NOT NULL, multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL), NULL)) AS `service.name`, sum(multiIf(`attribute_number_cart$$items_count_exists` = ?, toFloat64(`attribute_number_cart$$items_count`), NULL)) AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND true AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? AND (`service.name`) GLOBAL IN (SELECT `service.name` FROM __limit_cte) GROUP BY ts, `service.name`",
				Args:  []any{"redis-manual", "%service.name%", "%service.name\":\"redis-manual%", uint64(1747945619), uint64(1747983448), true, "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10, true, "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448)},
			},
			expectedErr: nil,
		},
		{
			name:        "mat number key in aggregation test with order by service",
			requestType: qbtypes.RequestTypeTimeSeries,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal:       telemetrytypes.SignalTraces,
				StepInterval: qbtypes.Step{Duration: 30 * time.Second},
				Aggregations: []qbtypes.TraceAggregation{
					{
						Expression: "sum(cart.items_count)",
					},
				},
				Filter: &qbtypes.Filter{
					Expression: "service.name = 'redis-manual'",
				},
				Limit: 10,
				GroupBy: []qbtypes.GroupByKey{
					{
						TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
							Name: "service.name",
						},
					},
				},
				Order: []qbtypes.OrderBy{
					{
						Key: qbtypes.OrderByKey{
							TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
								Name: "service.name",
							},
						},
						Direction: qbtypes.OrderDirectionDesc,
					},
				},
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE (simpleJSONExtractString(labels, 'service.name') = ? AND labels LIKE ? AND labels LIKE ?) AND seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?), __limit_cte AS (SELECT toString(multiIf(multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) IS NOT NULL, multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL), NULL)) AS `service.name`, sum(multiIf(`attribute_number_cart$$items_count_exists` = ?, toFloat64(`attribute_number_cart$$items_count`), NULL)) AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND true AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? GROUP BY `service.name` ORDER BY `service.name` desc LIMIT ?) SELECT toStartOfInterval(timestamp, INTERVAL 30 SECOND) AS ts, toString(multiIf(multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) IS NOT NULL, multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL), NULL)) AS `service.name`, sum(multiIf(`attribute_number_cart$$items_count_exists` = ?, toFloat64(`attribute_number_cart$$items_count`), NULL)) AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND true AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? AND (`service.name`) GLOBAL IN (SELECT `service.name` FROM __limit_cte) GROUP BY ts, `service.name` ORDER BY `service.name` desc, ts desc",
				Args:  []any{"redis-manual", "%service.name%", "%service.name\":\"redis-manual%", uint64(1747945619), uint64(1747983448), true, "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10, true, "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448)},
			},
			expectedErr: nil,
		},
		{
			name:        "Legacy column with incorrect field context test",
			requestType: qbtypes.RequestTypeTimeSeries,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal:       telemetrytypes.SignalTraces,
				StepInterval: qbtypes.Step{Duration: 30 * time.Second},
				Aggregations: []qbtypes.TraceAggregation{
					{
						Expression: "count()",
					},
				},
				Filter: &qbtypes.Filter{
					Expression: "service.name = 'redis-manual'",
				},
				Limit: 10,
				GroupBy: []qbtypes.GroupByKey{
					{
						TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
							Name:          "response_status_code",
							FieldContext:  telemetrytypes.FieldContextAttribute,
							FieldDataType: telemetrytypes.FieldDataTypeString,
						},
					},
				},
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE (simpleJSONExtractString(labels, 'service.name') = ? AND labels LIKE ? AND labels LIKE ?) AND seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?), __limit_cte AS (SELECT toString(multiIf(response_status_code <> ?, response_status_code, NULL)) AS `response_status_code`, count() AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND true AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? GROUP BY `response_status_code` ORDER BY __result_0 DESC LIMIT ?) SELECT toStartOfInterval(timestamp, INTERVAL 30 SECOND) AS ts, toString(multiIf(response_status_code <> ?, response_status_code, NULL)) AS `response_status_code`, count() AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND true AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? AND (`response_status_code`) GLOBAL IN (SELECT `response_status_code` FROM __limit_cte) GROUP BY ts, `response_status_code`",
				Args:  []any{"redis-manual", "%service.name%", "%service.name\":\"redis-manual%", uint64(1747945619), uint64(1747983448), "", "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10, "", "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448)},
			},
			expectedErr: nil,
		},
		{
			name:        "Legacy column in aggregation and incorrect field context test",
			requestType: qbtypes.RequestTypeTimeSeries,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal:       telemetrytypes.SignalTraces,
				StepInterval: qbtypes.Step{Duration: 30 * time.Second},
				Aggregations: []qbtypes.TraceAggregation{
					{
						Expression: "p90(duration_nano)",
					},
				},
				Filter: &qbtypes.Filter{
					Expression: "service.name = 'redis-manual'",
				},
				Limit: 10,
				GroupBy: []qbtypes.GroupByKey{
					{
						TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
							Name:          "response_status_code",
							FieldContext:  telemetrytypes.FieldContextAttribute,
							FieldDataType: telemetrytypes.FieldDataTypeString,
						},
					},
				},
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE (simpleJSONExtractString(labels, 'service.name') = ? AND labels LIKE ? AND labels LIKE ?) AND seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?), __limit_cte AS (SELECT toString(multiIf(response_status_code <> ?, response_status_code, NULL)) AS `response_status_code`, quantile(0.90)(multiIf(duration_nano <> ?, duration_nano, NULL)) AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND true AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? GROUP BY `response_status_code` ORDER BY __result_0 DESC LIMIT ?) SELECT toStartOfInterval(timestamp, INTERVAL 30 SECOND) AS ts, toString(multiIf(response_status_code <> ?, response_status_code, NULL)) AS `response_status_code`, quantile(0.90)(multiIf(duration_nano <> ?, duration_nano, NULL)) AS __result_0 FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND true AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? AND (`response_status_code`) GLOBAL IN (SELECT `response_status_code` FROM __limit_cte) GROUP BY ts, `response_status_code`",
				Args:  []any{"redis-manual", "%service.name%", "%service.name\":\"redis-manual%", uint64(1747945619), uint64(1747983448), "", 0, "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10, "", 0, "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448)},
			},
			expectedErr: nil,
		},
	}

	fm := NewFieldMapper()
	cb := NewConditionBuilder(fm)
	mockMetadataStore := telemetrytypestest.NewMockMetadataStore()
	mockMetadataStore.KeysMap = buildCompleteFieldKeyMap()
	aggExprRewriter := querybuilder.NewAggExprRewriter(instrumentationtest.New().ToProviderSettings(), nil, fm, cb, nil)

	resourceFilterStmtBuilder := resourceFilterStmtBuilder()

	statementBuilder := NewTraceQueryStatementBuilder(
		instrumentationtest.New().ToProviderSettings(),
		mockMetadataStore,
		fm,
		cb,
		resourceFilterStmtBuilder,
		aggExprRewriter,
		nil,
	)

	vars := map[string]qbtypes.VariableItem{
		"service.name": {
			Value: "redis-manual",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {

			q, err := statementBuilder.Build(context.Background(), 1747947419000, 1747983448000, c.requestType, c.query, vars)

			if c.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), c.expectedErr.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, c.expected.Query, q.Query)
				require.Equal(t, c.expected.Args, q.Args)
				require.Equal(t, c.expected.Warnings, q.Warnings)
			}
		})
	}
}

func TestStatementBuilderListQuery(t *testing.T) {
	cases := []struct {
		name        string
		requestType qbtypes.RequestType
		query       qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]
		expected    qbtypes.Statement
		expectedErr error
	}{
		{
			name:        "List query with mat selected fields",
			requestType: qbtypes.RequestTypeRaw,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal:       telemetrytypes.SignalTraces,
				StepInterval: qbtypes.Step{Duration: 30 * time.Second},
				Filter: &qbtypes.Filter{
					Expression: "service.name = 'redis-manual'",
				},
				Limit: 10,
				SelectFields: []telemetrytypes.TelemetryFieldKey{
					{
						Name:          "name",
						Signal:        telemetrytypes.SignalTraces,
						FieldContext:  telemetrytypes.FieldContextSpan,
						FieldDataType: telemetrytypes.FieldDataTypeString,
					},
					{
						Name:          "service.name",
						Signal:        telemetrytypes.SignalTraces,
						FieldContext:  telemetrytypes.FieldContextResource,
						FieldDataType: telemetrytypes.FieldDataTypeString,
						Materialized:  true,
					},
					{
						Name:          "duration_nano",
						Signal:        telemetrytypes.SignalTraces,
						FieldContext:  telemetrytypes.FieldContextSpan,
						FieldDataType: telemetrytypes.FieldDataTypeNumber,
					},
					{
						Name:          "cart.items_count",
						FieldContext:  telemetrytypes.FieldContextAttribute,
						FieldDataType: telemetrytypes.FieldDataTypeFloat64,
					},
				},
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE (simpleJSONExtractString(labels, 'service.name') = ? AND labels LIKE ? AND labels LIKE ?) AND seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?) SELECT name AS `name`, multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) AS `service.name`, duration_nano AS `duration_nano`, `attribute_number_cart$$items_count` AS `cart.items_count`, timestamp AS `timestamp`, span_id AS `span_id`, trace_id AS `trace_id` FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND true AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? LIMIT ?",
				Args:  []any{"redis-manual", "%service.name%", "%service.name\":\"redis-manual%", uint64(1747945619), uint64(1747983448), "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10},
			},
			expectedErr: nil,
		},
		{
			name:        "List query with default fields and attribute order by",
			requestType: qbtypes.RequestTypeRaw,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal:       telemetrytypes.SignalTraces,
				StepInterval: qbtypes.Step{Duration: 30 * time.Second},
				Filter: &qbtypes.Filter{
					Expression: "service.name = 'redis-manual'",
				},
				Order: []qbtypes.OrderBy{
					{
						Key: qbtypes.OrderByKey{
							TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
								Name:          "user.id",
								FieldContext:  telemetrytypes.FieldContextAttribute,
								FieldDataType: telemetrytypes.FieldDataTypeString,
							},
						},
						Direction: qbtypes.OrderDirectionDesc,
					},
				},
				Limit: 10,
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE (simpleJSONExtractString(labels, 'service.name') = ? AND labels LIKE ? AND labels LIKE ?) AND seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?) SELECT duration_nano AS `duration_nano`, name AS `name`, response_status_code AS `response_status_code`, multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) AS `service.name`, span_id AS `span_id`, timestamp AS `timestamp`, trace_id AS `trace_id` FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND true AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? ORDER BY attributes_string['user.id'] AS `user.id` desc LIMIT ?",
				Args:  []any{"redis-manual", "%service.name%", "%service.name\":\"redis-manual%", uint64(1747945619), uint64(1747983448), "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10},
			},
			expectedErr: nil,
		},
	}

	fm := NewFieldMapper()
	cb := NewConditionBuilder(fm)
	mockMetadataStore := telemetrytypestest.NewMockMetadataStore()
	mockMetadataStore.KeysMap = buildCompleteFieldKeyMap()
	aggExprRewriter := querybuilder.NewAggExprRewriter(instrumentationtest.New().ToProviderSettings(), nil, fm, cb, nil)

	resourceFilterStmtBuilder := resourceFilterStmtBuilder()

	statementBuilder := NewTraceQueryStatementBuilder(
		instrumentationtest.New().ToProviderSettings(),
		mockMetadataStore,
		fm,
		cb,
		resourceFilterStmtBuilder,
		aggExprRewriter,
		nil,
	)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {

			q, err := statementBuilder.Build(context.Background(), 1747947419000, 1747983448000, c.requestType, c.query, nil)

			if c.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), c.expectedErr.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, c.expected.Query, q.Query)
				require.Equal(t, c.expected.Args, q.Args)
				require.Equal(t, c.expected.Warnings, q.Warnings)
			}
		})
	}
}

func TestStatementBuilderListQueryWithCorruptData(t *testing.T) {
	cases := []struct {
		name        string
		requestType qbtypes.RequestType
		query       qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]
		keysMap     map[string][]*telemetrytypes.TelemetryFieldKey
		expected    qbtypes.Statement
		expectedErr error
	}{
		{
			name:        "List query with empty select fields",
			requestType: qbtypes.RequestTypeRaw,
			keysMap: map[string][]*telemetrytypes.TelemetryFieldKey{
				"timestamp": {
					{
						Name:          "timestamp",
						Signal:        telemetrytypes.SignalTraces,
						FieldContext:  telemetrytypes.FieldContextAttribute,
						FieldDataType: telemetrytypes.FieldDataTypeString,
					},
				},
			},
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal:       telemetrytypes.SignalTraces,
				StepInterval: qbtypes.Step{Duration: 30 * time.Second},
				Filter:       &qbtypes.Filter{},
				Limit:        10,
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?) SELECT duration_nano AS `duration_nano`, name AS `name`, response_status_code AS `response_status_code`, multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) AS `service.name`, span_id AS `span_id`, timestamp AS `timestamp`, trace_id AS `trace_id` FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? LIMIT ?",
				Args:  []any{uint64(1747945619), uint64(1747983448), "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10},
			},
			expectedErr: nil,
		},
		{
			name:        "List query with empty select fields and order by timestamp",
			requestType: qbtypes.RequestTypeRaw,
			keysMap: map[string][]*telemetrytypes.TelemetryFieldKey{
				"timestamp": {
					{
						Name:          "timestamp",
						Signal:        telemetrytypes.SignalTraces,
						FieldContext:  telemetrytypes.FieldContextAttribute,
						FieldDataType: telemetrytypes.FieldDataTypeString,
					},
				},
			},
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal:       telemetrytypes.SignalTraces,
				StepInterval: qbtypes.Step{Duration: 30 * time.Second},
				Filter:       &qbtypes.Filter{},
				Limit:        10,
				Order: []qbtypes.OrderBy{{
					Key: qbtypes.OrderByKey{
						TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
							Name: "timestamp",
						},
					},
					Direction: qbtypes.OrderDirectionAsc,
				}},
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?) SELECT duration_nano AS `duration_nano`, name AS `name`, response_status_code AS `response_status_code`, multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) AS `service.name`, span_id AS `span_id`, timestamp AS `timestamp`, trace_id AS `trace_id` FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? ORDER BY timestamp AS `timestamp` asc LIMIT ?",
				Args:  []any{uint64(1747945619), uint64(1747983448), "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10},
			},
			expectedErr: nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fm := NewFieldMapper()
			cb := NewConditionBuilder(fm)
			mockMetadataStore := telemetrytypestest.NewMockMetadataStore()
			mockMetadataStore.KeysMap = c.keysMap
			if mockMetadataStore.KeysMap == nil {
				mockMetadataStore.KeysMap = buildCompleteFieldKeyMap()
			}
			aggExprRewriter := querybuilder.NewAggExprRewriter(instrumentationtest.New().ToProviderSettings(), nil, fm, cb, nil)

			resourceFilterStmtBuilder := resourceFilterStmtBuilder()

			statementBuilder := NewTraceQueryStatementBuilder(
				instrumentationtest.New().ToProviderSettings(),
				mockMetadataStore,
				fm,
				cb,
				resourceFilterStmtBuilder,
				aggExprRewriter,
				nil,
			)

			q, err := statementBuilder.Build(context.Background(), 1747947419000, 1747983448000, c.requestType, c.query, nil)

			if c.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), c.expectedErr.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, c.expected.Query, q.Query)
				require.Equal(t, c.expected.Args, q.Args)
				require.Equal(t, c.expected.Warnings, q.Warnings)
			}
		})
	}
}

func TestStatementBuilderTraceQuery(t *testing.T) {
	cases := []struct {
		name        string
		requestType qbtypes.RequestType
		query       qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]
		expected    qbtypes.Statement
		expectedErr error
	}{
		{
			name:        "List query with non-mat selected fields",
			requestType: qbtypes.RequestTypeTrace,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal: telemetrytypes.SignalTraces,
				Filter: &qbtypes.Filter{
					Expression: "service.name = 'redis-manual'",
				},
				Limit: 10,
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE (simpleJSONExtractString(labels, 'service.name') = ? AND labels LIKE ? AND labels LIKE ?) AND seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?), __toe AS (SELECT trace_id FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND true AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ?), __toe_duration_sorted AS (SELECT trace_id, duration_nano, resource_string_service$$name as `service.name`, name FROM signoz_traces.signoz_index_v3 WHERE parent_span_id = '' AND trace_id GLOBAL IN __toe AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? ORDER BY duration_nano DESC LIMIT 1 BY trace_id) SELECT __toe_duration_sorted.`service.name` AS `service.name`, __toe_duration_sorted.name AS `name`, count() AS span_count, __toe_duration_sorted.duration_nano AS `duration_nano`, __toe_duration_sorted.trace_id AS `trace_id` FROM __toe INNER JOIN __toe_duration_sorted ON __toe.trace_id = __toe_duration_sorted.trace_id GROUP BY trace_id, duration_nano, name, `service.name` ORDER BY duration_nano DESC LIMIT 1 BY trace_id LIMIT ? SETTINGS max_memory_usage=10000000000",
				Args:  []any{"redis-manual", "%service.name%", "%service.name\":\"redis-manual%", uint64(1747945619), uint64(1747983448), "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10},
			},
			expectedErr: nil,
		},
		{
			name:        "List query with mat selected fields",
			requestType: qbtypes.RequestTypeTrace,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal: telemetrytypes.SignalTraces,
				Filter: &qbtypes.Filter{
					Expression: "materialized.key.name = 'redis-manual'",
				},
				Limit: 10,
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE true AND seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?), __toe AS (SELECT trace_id FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND (`attribute_string_materialized$$key$$name` = ? AND `attribute_string_materialized$$key$$name_exists` = ?) AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ?), __toe_duration_sorted AS (SELECT trace_id, duration_nano, resource_string_service$$name as `service.name`, name FROM signoz_traces.signoz_index_v3 WHERE parent_span_id = '' AND trace_id GLOBAL IN __toe AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? ORDER BY duration_nano DESC LIMIT 1 BY trace_id) SELECT __toe_duration_sorted.`service.name` AS `service.name`, __toe_duration_sorted.name AS `name`, count() AS span_count, __toe_duration_sorted.duration_nano AS `duration_nano`, __toe_duration_sorted.trace_id AS `trace_id` FROM __toe INNER JOIN __toe_duration_sorted ON __toe.trace_id = __toe_duration_sorted.trace_id GROUP BY trace_id, duration_nano, name, `service.name` ORDER BY duration_nano DESC LIMIT 1 BY trace_id LIMIT ? SETTINGS max_memory_usage=10000000000",
				Args:  []any{uint64(1747945619), uint64(1747983448), "redis-manual", true, "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10},
			},
			expectedErr: nil,
		},
		{
			name:        "List query with mat selected fields using or and regexp",
			requestType: qbtypes.RequestTypeTrace,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal: telemetrytypes.SignalTraces,
				Filter: &qbtypes.Filter{
					Expression: "materialized.key.name REGEXP 'redis-manual' OR service.name = 'redis-manual'",
				},
				Limit: 10,
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE (true OR (simpleJSONExtractString(labels, 'service.name') = ? AND labels LIKE ? AND labels LIKE ?)) AND seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?), __toe AS (SELECT trace_id FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND ((match(`attribute_string_materialized$$key$$name`, ?) AND `attribute_string_materialized$$key$$name_exists` = ?) OR (multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) = ? AND multiIf(resource.`service.name` IS NOT NULL, resource.`service.name`::String, mapContains(resources_string, 'service.name'), resources_string['service.name'], NULL) IS NOT NULL)) AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ?), __toe_duration_sorted AS (SELECT trace_id, duration_nano, resource_string_service$$name as `service.name`, name FROM signoz_traces.signoz_index_v3 WHERE parent_span_id = '' AND trace_id GLOBAL IN __toe AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? ORDER BY duration_nano DESC LIMIT 1 BY trace_id) SELECT __toe_duration_sorted.`service.name` AS `service.name`, __toe_duration_sorted.name AS `name`, count() AS span_count, __toe_duration_sorted.duration_nano AS `duration_nano`, __toe_duration_sorted.trace_id AS `trace_id` FROM __toe INNER JOIN __toe_duration_sorted ON __toe.trace_id = __toe_duration_sorted.trace_id GROUP BY trace_id, duration_nano, name, `service.name` ORDER BY duration_nano DESC LIMIT 1 BY trace_id LIMIT ? SETTINGS max_memory_usage=10000000000",
				Args:  []any{"redis-manual", "%service.name%", "%service.name\":\"redis-manual%", uint64(1747945619), uint64(1747983448), "redis-manual", true, "redis-manual", "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10},
			},
			expectedErr: nil,
		},
		{
			name:        "List query without any filter",
			requestType: qbtypes.RequestTypeTrace,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal: telemetrytypes.SignalTraces,
				Filter: &qbtypes.Filter{},
				Limit:  10,
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?), __toe AS (SELECT trace_id FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ?), __toe_duration_sorted AS (SELECT trace_id, duration_nano, resource_string_service$$name as `service.name`, name FROM signoz_traces.signoz_index_v3 WHERE parent_span_id = '' AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? ORDER BY duration_nano DESC LIMIT 1 BY trace_id) SELECT __toe_duration_sorted.`service.name` AS `service.name`, __toe_duration_sorted.name AS `name`, count() AS span_count, __toe_duration_sorted.duration_nano AS `duration_nano`, __toe_duration_sorted.trace_id AS `trace_id` FROM __toe INNER JOIN __toe_duration_sorted ON __toe.trace_id = __toe_duration_sorted.trace_id GROUP BY trace_id, duration_nano, name, `service.name` ORDER BY duration_nano DESC LIMIT 1 BY trace_id LIMIT ? SETTINGS max_memory_usage=10000000000",
				Args:  []any{uint64(1747945619), uint64(1747983448), "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10},
			},
			expectedErr: nil,
		},
		{
			name:        "list query with span scope filter",
			requestType: qbtypes.RequestTypeTrace,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal: telemetrytypes.SignalTraces,
				Filter: &qbtypes.Filter{
					Expression: "isentrypoint = true",
				},
				Limit: 10,
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE true AND seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?), __toe AS (SELECT trace_id FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND ((name, resource_string_service$$name) GLOBAL IN (SELECT DISTINCT name, serviceName from signoz_traces.top_level_operations WHERE time >= toDateTime(1747947419))) AND parent_span_id != '' AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ?), __toe_duration_sorted AS (SELECT trace_id, duration_nano, resource_string_service$$name as `service.name`, name FROM signoz_traces.signoz_index_v3 WHERE parent_span_id = '' AND trace_id GLOBAL IN __toe AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? ORDER BY duration_nano DESC LIMIT 1 BY trace_id) SELECT __toe_duration_sorted.`service.name` AS `service.name`, __toe_duration_sorted.name AS `name`, count() AS span_count, __toe_duration_sorted.duration_nano AS `duration_nano`, __toe_duration_sorted.trace_id AS `trace_id` FROM __toe INNER JOIN __toe_duration_sorted ON __toe.trace_id = __toe_duration_sorted.trace_id GROUP BY trace_id, duration_nano, name, `service.name` ORDER BY duration_nano DESC LIMIT 1 BY trace_id LIMIT ? SETTINGS max_memory_usage=10000000000",
				Args:  []any{uint64(1747945619), uint64(1747983448), "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10},
			},
			expectedErr: nil,
		},
		{
			name:        "list query with span scope filter Or mat field",
			requestType: qbtypes.RequestTypeTrace,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal: telemetrytypes.SignalTraces,
				Filter: &qbtypes.Filter{
					Expression: "isentrypoint = true or materialized.key.name = 'redis-manual'",
				},
				Limit: 10,
			},
			expected: qbtypes.Statement{
				Query: "WITH __resource_filter AS (SELECT fingerprint FROM signoz_traces.traces_v3_resource WHERE (true OR true) AND seen_at_ts_bucket_start >= ? AND seen_at_ts_bucket_start <= ?), __toe AS (SELECT trace_id FROM signoz_traces.signoz_index_v3 WHERE resource_fingerprint GLOBAL IN (SELECT fingerprint FROM __resource_filter) AND (((name, resource_string_service$$name) GLOBAL IN (SELECT DISTINCT name, serviceName from signoz_traces.top_level_operations WHERE time >= toDateTime(1747947419))) AND parent_span_id != '' OR (`attribute_string_materialized$$key$$name` = ? AND `attribute_string_materialized$$key$$name_exists` = ?)) AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ?), __toe_duration_sorted AS (SELECT trace_id, duration_nano, resource_string_service$$name as `service.name`, name FROM signoz_traces.signoz_index_v3 WHERE parent_span_id = '' AND trace_id GLOBAL IN __toe AND timestamp >= ? AND timestamp < ? AND ts_bucket_start >= ? AND ts_bucket_start <= ? ORDER BY duration_nano DESC LIMIT 1 BY trace_id) SELECT __toe_duration_sorted.`service.name` AS `service.name`, __toe_duration_sorted.name AS `name`, count() AS span_count, __toe_duration_sorted.duration_nano AS `duration_nano`, __toe_duration_sorted.trace_id AS `trace_id` FROM __toe INNER JOIN __toe_duration_sorted ON __toe.trace_id = __toe_duration_sorted.trace_id GROUP BY trace_id, duration_nano, name, `service.name` ORDER BY duration_nano DESC LIMIT 1 BY trace_id LIMIT ? SETTINGS max_memory_usage=10000000000",
				Args:  []any{uint64(1747945619), uint64(1747983448), "redis-manual", true, "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), "1747947419000000000", "1747983448000000000", uint64(1747945619), uint64(1747983448), 10},
			},
			expectedErr: nil,
		},
		{
			name:        "reject deprecated filter field",
			requestType: qbtypes.RequestTypeTrace,
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Signal: telemetrytypes.SignalTraces,
				Filter: &qbtypes.Filter{
					Expression: "kind = 2 or kind_string = 'Server'",
				},
				Limit: 10,
			},
			expectedErr: errors.NewInvalidInputf(errors.CodeInvalidInput, "Found 1 errors while parsing the search expression."),
		},
	}

	fm := NewFieldMapper()
	cb := NewConditionBuilder(fm)
	mockMetadataStore := telemetrytypestest.NewMockMetadataStore()
	mockMetadataStore.KeysMap = buildCompleteFieldKeyMap()
	aggExprRewriter := querybuilder.NewAggExprRewriter(instrumentationtest.New().ToProviderSettings(), nil, fm, cb, nil)

	resourceFilterStmtBuilder := resourceFilterStmtBuilder()

	statementBuilder := NewTraceQueryStatementBuilder(
		instrumentationtest.New().ToProviderSettings(),
		mockMetadataStore,
		fm,
		cb,
		resourceFilterStmtBuilder,
		aggExprRewriter,
		nil,
	)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {

			q, err := statementBuilder.Build(context.Background(), 1747947419000, 1747983448000, c.requestType, c.query, nil)

			if c.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), c.expectedErr.Error())
			} else {
				require.NoError(t, err)
				require.Equal(t, c.expected.Query, q.Query)
				require.Equal(t, c.expected.Args, q.Args)
				require.Equal(t, c.expected.Warnings, q.Warnings)
			}
		})
	}
}

func TestAdjustKey(t *testing.T) {
	cases := []struct {
		name        string
		inputKey    telemetrytypes.TelemetryFieldKey
		keysMap     map[string][]*telemetrytypes.TelemetryFieldKey
		expectedKey telemetrytypes.TelemetryFieldKey
	}{
		{
			name: "intrinsic field with no metadata match - override",
			inputKey: telemetrytypes.TelemetryFieldKey{
				Name:          "trace_id",
				FieldContext:  telemetrytypes.FieldContextUnspecified,
				FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
			},
			keysMap:     buildCompleteFieldKeyMap(),
			expectedKey: IntrinsicFields["trace_id"],
		},
		{
			name: "intrinsic field with metadata match with incorrect context - override",
			inputKey: telemetrytypes.TelemetryFieldKey{
				Name:          "duration_nano",
				FieldContext:  telemetrytypes.FieldContextBody, // incorrect context
				FieldDataType: telemetrytypes.FieldDataTypeInt64,
			},
			keysMap: buildCompleteFieldKeyMap(),
			expectedKey: telemetrytypes.TelemetryFieldKey{
				Name:          "duration_nano",
				FieldContext:  telemetrytypes.FieldContextSpan,    // should be corrected
				FieldDataType: telemetrytypes.FieldDataTypeNumber, // modified
			},
		},
		{
			name: "intrinsic field with metadata match with correct context - no override",
			inputKey: telemetrytypes.TelemetryFieldKey{
				Name:          "duration_nano",
				FieldContext:  telemetrytypes.FieldContextSpan, // correct context
				FieldDataType: telemetrytypes.FieldDataTypeInt64,
			},
			keysMap: buildCompleteFieldKeyMap(),
			expectedKey: telemetrytypes.TelemetryFieldKey{
				Name:          "duration_nano",
				FieldContext:  telemetrytypes.FieldContextSpan,   // should be corrected
				FieldDataType: telemetrytypes.FieldDataTypeInt64, // not modified
			},
		},
		{
			name: "single matching key in metadata - override",
			inputKey: telemetrytypes.TelemetryFieldKey{
				Name:          "service.name",
				FieldContext:  telemetrytypes.FieldContextUnspecified,
				FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
			},
			keysMap:     buildCompleteFieldKeyMap(),
			expectedKey: *buildCompleteFieldKeyMap()["service.name"][0],
		},
		{
			name: "single matching key with context specified - override",
			inputKey: telemetrytypes.TelemetryFieldKey{
				Name:          "cart.items_count",
				FieldContext:  telemetrytypes.FieldContextAttribute,
				FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
			},
			keysMap:     buildCompleteFieldKeyMap(),
			expectedKey: *buildCompleteFieldKeyMap()["cart.items_count"][0],
		},
		{
			name: "multiple matching keys - all materialized",
			inputKey: telemetrytypes.TelemetryFieldKey{
				Name:          "multi.mat.key",
				FieldContext:  telemetrytypes.FieldContextUnspecified,
				FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
			},
			keysMap: map[string][]*telemetrytypes.TelemetryFieldKey{
				"multi.mat.key": {
					{
						Name:          "multi.mat.key",
						FieldContext:  telemetrytypes.FieldContextAttribute,
						FieldDataType: telemetrytypes.FieldDataTypeString,
						Materialized:  true,
					},
					{
						Name:          "multi.mat.key",
						FieldContext:  telemetrytypes.FieldContextResource,
						FieldDataType: telemetrytypes.FieldDataTypeNumber,
						Materialized:  true,
					},
				},
			},
			expectedKey: telemetrytypes.TelemetryFieldKey{
				Name:         "multi.mat.key",
				Materialized: true,
			},
		},
		{
			name: "multiple matching keys - mixed materialization",
			inputKey: telemetrytypes.TelemetryFieldKey{
				Name:          "mixed.materialization.key",
				FieldContext:  telemetrytypes.FieldContextUnspecified,
				FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
			},
			keysMap: buildCompleteFieldKeyMap(),
			expectedKey: telemetrytypes.TelemetryFieldKey{
				Name:          "mixed.materialization.key",
				FieldDataType: telemetrytypes.FieldDataTypeString,
				Materialized:  false,
			},
		},
		{
			name: "multiple matching keys with context specified",
			inputKey: telemetrytypes.TelemetryFieldKey{
				Name:          "mixed.materialization.key",
				FieldContext:  telemetrytypes.FieldContextAttribute,
				FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
			},
			keysMap: buildCompleteFieldKeyMap(),
			expectedKey: telemetrytypes.TelemetryFieldKey{
				Name:          "mixed.materialization.key",
				FieldContext:  telemetrytypes.FieldContextAttribute,
				FieldDataType: telemetrytypes.FieldDataTypeString,
				Materialized:  true,
			},
		},
		{
			name: "no matching keys - unknown field",
			inputKey: telemetrytypes.TelemetryFieldKey{
				Name:          "unknown.field",
				FieldContext:  telemetrytypes.FieldContextUnspecified,
				FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
			},
			keysMap: buildCompleteFieldKeyMap(),
			expectedKey: telemetrytypes.TelemetryFieldKey{
				Name:         "unknown.field",
				Materialized: false,
			},
		},
		{
			name: "no matching keys with context filter",
			inputKey: telemetrytypes.TelemetryFieldKey{
				Name:          "service.name",
				FieldContext:  telemetrytypes.FieldContextAttribute,
				FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
			},
			keysMap: buildCompleteFieldKeyMap(),
			expectedKey: telemetrytypes.TelemetryFieldKey{
				Name:          "service.name",
				FieldContext:  telemetrytypes.FieldContextAttribute,
				FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
				Materialized:  false,
			},
		},
		{
			name: "materialized field",
			inputKey: telemetrytypes.TelemetryFieldKey{
				Name:          "cart.items_count",
				FieldContext:  telemetrytypes.FieldContextUnspecified,
				FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
			},
			keysMap: buildCompleteFieldKeyMap(),
			expectedKey: telemetrytypes.TelemetryFieldKey{
				Name:          "cart.items_count",
				FieldContext:  telemetrytypes.FieldContextAttribute,
				FieldDataType: telemetrytypes.FieldDataTypeFloat64,
				Materialized:  true,
			},
		},
		{
			name: "non-materialized field",
			inputKey: telemetrytypes.TelemetryFieldKey{
				Name:          "user.id",
				FieldContext:  telemetrytypes.FieldContextUnspecified,
				FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
			},
			keysMap: buildCompleteFieldKeyMap(),
			expectedKey: telemetrytypes.TelemetryFieldKey{
				Name:          "user.id",
				FieldContext:  telemetrytypes.FieldContextAttribute,
				FieldDataType: telemetrytypes.FieldDataTypeString,
				Materialized:  false,
			},
		},
	}

	fm := NewFieldMapper()
	cb := NewConditionBuilder(fm)
	mockMetadataStore := telemetrytypestest.NewMockMetadataStore()
	aggExprRewriter := querybuilder.NewAggExprRewriter(instrumentationtest.New().ToProviderSettings(), nil, fm, cb, nil)
	resourceFilterStmtBuilder := resourceFilterStmtBuilder()

	statementBuilder := NewTraceQueryStatementBuilder(
		instrumentationtest.New().ToProviderSettings(),
		mockMetadataStore,
		fm,
		cb,
		resourceFilterStmtBuilder,
		aggExprRewriter,
		nil,
	)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Create a copy of the input key to avoid modifying the original
			key := c.inputKey

			// Call adjustKey
			statementBuilder.adjustKey(&key, c.keysMap)

			// Verify the key was adjusted as expected
			require.Equal(t, c.expectedKey.Name, key.Name, "key name should match")
			require.Equal(t, c.expectedKey.FieldContext, key.FieldContext, "field context should match")
			require.Equal(t, c.expectedKey.FieldDataType, key.FieldDataType, "field data type should match")
			require.Equal(t, c.expectedKey.Materialized, key.Materialized, "materialized should match")
		})
	}
}

func TestAdjustKeys(t *testing.T) {
	cases := []struct {
		name                 string
		query                qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]
		keysMap              map[string][]*telemetrytypes.TelemetryFieldKey
		expectedSelectFields []telemetrytypes.TelemetryFieldKey
		expectedGroupBy      []qbtypes.GroupByKey
		expectedOrder        []qbtypes.OrderBy
	}{
		{
			name: "adjust select fields",
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				SelectFields: []telemetrytypes.TelemetryFieldKey{
					{
						Name:          "service.name",
						FieldContext:  telemetrytypes.FieldContextUnspecified,
						FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
					},
					{
						Name:          "cart.items_count",
						FieldContext:  telemetrytypes.FieldContextUnspecified,
						FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
					},
				},
			},
			keysMap: buildCompleteFieldKeyMap(),
			expectedSelectFields: []telemetrytypes.TelemetryFieldKey{
				{
					Name:          "service.name",
					FieldContext:  telemetrytypes.FieldContextResource,
					FieldDataType: telemetrytypes.FieldDataTypeString,
					Materialized:  false,
				},
				{
					Name:          "cart.items_count",
					FieldContext:  telemetrytypes.FieldContextAttribute,
					FieldDataType: telemetrytypes.FieldDataTypeFloat64,
					Materialized:  true,
				},
			},
		},
		{
			name: "adjust group by fields",
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				GroupBy: []qbtypes.GroupByKey{
					{
						TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
							Name:          "service.name",
							FieldContext:  telemetrytypes.FieldContextUnspecified,
							FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
						},
					},
					{
						TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
							Name:          "user.id",
							FieldContext:  telemetrytypes.FieldContextUnspecified,
							FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
						},
					},
				},
			},
			keysMap: buildCompleteFieldKeyMap(),
			expectedGroupBy: []qbtypes.GroupByKey{
				{
					TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
						Name:          "service.name",
						FieldContext:  telemetrytypes.FieldContextResource,
						FieldDataType: telemetrytypes.FieldDataTypeString,
						Materialized:  false,
					},
				},
				{
					TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
						Name:          "user.id",
						FieldContext:  telemetrytypes.FieldContextAttribute,
						FieldDataType: telemetrytypes.FieldDataTypeString,
						Materialized:  false,
					},
				},
			},
		},
		{
			name: "adjust order by fields",
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Order: []qbtypes.OrderBy{
					{
						Key: qbtypes.OrderByKey{
							TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
								Name:          "service.name",
								FieldContext:  telemetrytypes.FieldContextUnspecified,
								FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
							},
						},
						Direction: qbtypes.OrderDirectionAsc,
					},
					{
						Key: qbtypes.OrderByKey{
							TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
								Name:          "cart.items_count",
								FieldContext:  telemetrytypes.FieldContextUnspecified,
								FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
							},
						},
						Direction: qbtypes.OrderDirectionDesc,
					},
				},
			},
			keysMap: buildCompleteFieldKeyMap(),
			expectedOrder: []qbtypes.OrderBy{
				{
					Key: qbtypes.OrderByKey{
						TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
							Name:          "service.name",
							FieldContext:  telemetrytypes.FieldContextResource,
							FieldDataType: telemetrytypes.FieldDataTypeString,
							Materialized:  false,
						},
					},
					Direction: qbtypes.OrderDirectionAsc,
				},
				{
					Key: qbtypes.OrderByKey{
						TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
							Name:          "cart.items_count",
							FieldContext:  telemetrytypes.FieldContextAttribute,
							FieldDataType: telemetrytypes.FieldDataTypeFloat64,
							Materialized:  true,
						},
					},
					Direction: qbtypes.OrderDirectionDesc,
				},
			},
		},
		{
			name: "adjust all field types together",
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				SelectFields: []telemetrytypes.TelemetryFieldKey{
					{
						Name:          "trace_id",
						FieldContext:  telemetrytypes.FieldContextUnspecified,
						FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
					},
				},
				GroupBy: []qbtypes.GroupByKey{
					{
						TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
							Name:          "service.name",
							FieldContext:  telemetrytypes.FieldContextUnspecified,
							FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
						},
					},
				},
				Order: []qbtypes.OrderBy{
					{
						Key: qbtypes.OrderByKey{
							TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
								Name:          "user.id",
								FieldContext:  telemetrytypes.FieldContextUnspecified,
								FieldDataType: telemetrytypes.FieldDataTypeUnspecified,
							},
						},
						Direction: qbtypes.OrderDirectionDesc,
					},
				},
			},
			keysMap: buildCompleteFieldKeyMap(),
			expectedSelectFields: []telemetrytypes.TelemetryFieldKey{
				{
					Name:          "trace_id",
					FieldContext:  telemetrytypes.FieldContextSpan,
					FieldDataType: telemetrytypes.FieldDataTypeString,
					Materialized:  false,
				},
			},
			expectedGroupBy: []qbtypes.GroupByKey{
				{
					TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
						Name:          "service.name",
						FieldContext:  telemetrytypes.FieldContextResource,
						FieldDataType: telemetrytypes.FieldDataTypeString,
						Materialized:  false,
					},
				},
			},
			expectedOrder: []qbtypes.OrderBy{
				{
					Key: qbtypes.OrderByKey{
						TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
							Name:          "user.id",
							FieldContext:  telemetrytypes.FieldContextAttribute,
							FieldDataType: telemetrytypes.FieldDataTypeString,
							Materialized:  false,
						},
					},
					Direction: qbtypes.OrderDirectionDesc,
				},
			},
		},
		{
			name: "adjust keys for alias expressions in aggregations - order by",
			query: qbtypes.QueryBuilderQuery[qbtypes.TraceAggregation]{
				Aggregations: []qbtypes.TraceAggregation{
					{
						Expression: "sum(span.duration)",
						Alias:      "span.duration",
					},
				},
				Order: []qbtypes.OrderBy{
					{
						Key: qbtypes.OrderByKey{
							TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
								Name:         "duration",
								FieldContext: telemetrytypes.FieldContextSpan,
							},
						},
						Direction: qbtypes.OrderDirectionDesc,
					},
				},
			},
			keysMap: buildCompleteFieldKeyMap(),
			// After alias adjustment, name becomes "span.duration" with FieldContextUnspecified
			// "span.duration" is not in keysMap, so context stays unspecified
			expectedOrder: []qbtypes.OrderBy{
				{
					Key: qbtypes.OrderByKey{
						TelemetryFieldKey: telemetrytypes.TelemetryFieldKey{
							Name:         "span.duration",
							FieldContext: telemetrytypes.FieldContextUnspecified,
						},
					},
					Direction: qbtypes.OrderDirectionDesc,
				},
			},
		},
	}

	fm := NewFieldMapper()
	cb := NewConditionBuilder(fm)
	mockMetadataStore := telemetrytypestest.NewMockMetadataStore()
	aggExprRewriter := querybuilder.NewAggExprRewriter(instrumentationtest.New().ToProviderSettings(), nil, fm, cb, nil)
	resourceFilterStmtBuilder := resourceFilterStmtBuilder()

	statementBuilder := NewTraceQueryStatementBuilder(
		instrumentationtest.New().ToProviderSettings(),
		mockMetadataStore,
		fm,
		cb,
		resourceFilterStmtBuilder,
		aggExprRewriter,
		nil,
	)

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Create a deep copy of the keys map to avoid modifying the original
			keysMapCopy := make(map[string][]*telemetrytypes.TelemetryFieldKey)
			for k, v := range c.keysMap {
				keysMapCopy[k] = make([]*telemetrytypes.TelemetryFieldKey, len(v))
				copy(keysMapCopy[k], v)
			}

			// Call adjustKeys
			c.query = statementBuilder.adjustKeys(context.Background(), keysMapCopy, c.query, qbtypes.RequestTypeScalar)

			// Verify select fields were adjusted
			if c.expectedSelectFields != nil {
				require.Len(t, c.query.SelectFields, len(c.expectedSelectFields))
				for i, expected := range c.expectedSelectFields {
					require.Equal(t, expected.Name, c.query.SelectFields[i].Name, "select field %d name should match", i)
					require.Equal(t, expected.FieldContext, c.query.SelectFields[i].FieldContext, "select field %d context should match", i)
					require.Equal(t, expected.FieldDataType, c.query.SelectFields[i].FieldDataType, "select field %d data type should match", i)
					require.Equal(t, expected.Materialized, c.query.SelectFields[i].Materialized, "select field %d materialized should match", i)
				}
			}

			// Verify group by fields were adjusted
			if c.expectedGroupBy != nil {
				require.Len(t, c.query.GroupBy, len(c.expectedGroupBy))
				for i, expected := range c.expectedGroupBy {
					require.Equal(t, expected.TelemetryFieldKey.Name, c.query.GroupBy[i].TelemetryFieldKey.Name, "group by field %d name should match", i)
					require.Equal(t, expected.TelemetryFieldKey.FieldContext, c.query.GroupBy[i].TelemetryFieldKey.FieldContext, "group by field %d context should match", i)
					require.Equal(t, expected.TelemetryFieldKey.FieldDataType, c.query.GroupBy[i].TelemetryFieldKey.FieldDataType, "group by field %d data type should match", i)
					require.Equal(t, expected.TelemetryFieldKey.Materialized, c.query.GroupBy[i].TelemetryFieldKey.Materialized, "group by field %d materialized should match", i)
				}
			}

			// Verify order by fields were adjusted
			if c.expectedOrder != nil {
				require.Len(t, c.query.Order, len(c.expectedOrder))
				for i, expected := range c.expectedOrder {
					require.Equal(t, expected.Key.TelemetryFieldKey.Name, c.query.Order[i].Key.TelemetryFieldKey.Name, "order field %d name should match", i)
					require.Equal(t, expected.Key.TelemetryFieldKey.FieldContext, c.query.Order[i].Key.TelemetryFieldKey.FieldContext, "order field %d context should match", i)
					require.Equal(t, expected.Key.TelemetryFieldKey.FieldDataType, c.query.Order[i].Key.TelemetryFieldKey.FieldDataType, "order field %d data type should match", i)
					require.Equal(t, expected.Key.TelemetryFieldKey.Materialized, c.query.Order[i].Key.TelemetryFieldKey.Materialized, "order field %d materialized should match", i)
					require.Equal(t, expected.Direction, c.query.Order[i].Direction, "order field %d direction should match", i)
				}
			}

		})
	}
}
