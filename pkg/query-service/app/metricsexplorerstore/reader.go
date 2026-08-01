package metricsexplorerstore

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	errorsV2 "github.com/SigNoz/signoz/pkg/errors"
	"github.com/SigNoz/signoz/pkg/query-service/common"
	"github.com/SigNoz/signoz/pkg/query-service/constants"
	"github.com/SigNoz/signoz/pkg/query-service/interfaces"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/query-service/model/metrics_explorer"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/query-service/utils"
	"github.com/SigNoz/signoz/pkg/types/ctxtypes"
	"github.com/SigNoz/signoz/pkg/types/instrumentationtypes"
	"github.com/SigNoz/signoz/pkg/types/telemetrytypes"
	"github.com/SigNoz/signoz/pkg/types/timeseriestypes"
	"github.com/SigNoz/signoz/pkg/valuer"
)

const (
	siginsightMetricDBName   = "siginsight_metrics"
	siginsightMetadataDBName = "siginsight_metadata"

	siginsightTSTableNameV4      = "time_series_v4"
	siginsightTSTableNameV41Day  = "time_series_v4_1day"
	siginsightTSTableNameV41Week = "time_series_v4_1week"
)

type metadataReader interface {
	GetUpdatedMetricsMetadata(context.Context, valuer.UUID, ...string) (map[string]*model.UpdateMetricsMetadata, error)
}

func metricColumn(key string) (string, bool) {
	switch key {
	case "metric_name":
		return "metric_name", true
	case "metric_unit":
		return "unit", true
	case "metric_type":
		return "type", true
	default:
		return "", false
	}
}

func buildMetricFilterConditions(filters *querytypes.FilterSet, skipKey string) ([]string, []any, error) {
	if filters == nil || len(filters.Items) == 0 {
		return nil, nil, nil
	}

	conditions := make([]string, 0, len(filters.Items))
	args := make([]any, 0, len(filters.Items)*2)
	for _, item := range filters.Items {
		if item.Key.Key == skipKey {
			continue
		}

		operator := querytypes.FilterOperator(strings.ToLower(strings.TrimSpace(string(item.Operator))))
		if operator == querytypes.FilterOperatorExists || operator == querytypes.FilterOperatorNotExists {
			prefix := ""
			if operator == querytypes.FilterOperatorNotExists {
				prefix = "not "
			}
			conditions = append(conditions, fmt.Sprintf("%shas(JSONExtractKeys(labels), ?)", prefix))
			args = append(args, item.Key.Key)
			continue
		}

		keyExpression, isColumn := metricColumn(item.Key.Key)
		if !isColumn {
			keyExpression = "JSONExtractString(labels, ?)"
			args = append(args, item.Key.Key)
		}

		value := item.Value
		if operator == querytypes.FilterOperatorContains || operator == querytypes.FilterOperatorNotContains {
			stringValue, ok := value.(string)
			if !ok {
				return nil, nil, fmt.Errorf("%s filter value must be a string", operator)
			}
			value = fmt.Sprintf("%%%s%%", stringValue)
		}

		var condition string
		switch operator {
		case querytypes.FilterOperatorEqual:
			condition = fmt.Sprintf("%s = ?", keyExpression)
		case querytypes.FilterOperatorNotEqual:
			condition = fmt.Sprintf("%s != ?", keyExpression)
		case querytypes.FilterOperatorIn:
			condition = fmt.Sprintf("%s IN ?", keyExpression)
		case querytypes.FilterOperatorNotIn:
			condition = fmt.Sprintf("%s NOT IN ?", keyExpression)
		case querytypes.FilterOperatorLike, querytypes.FilterOperatorContains:
			condition = fmt.Sprintf("like(%s, ?)", keyExpression)
		case querytypes.FilterOperatorNotLike, querytypes.FilterOperatorNotContains:
			condition = fmt.Sprintf("notLike(%s, ?)", keyExpression)
		case querytypes.FilterOperatorRegex:
			condition = fmt.Sprintf("match(%s, ?)", keyExpression)
		case querytypes.FilterOperatorNotRegex:
			condition = fmt.Sprintf("not match(%s, ?)", keyExpression)
		case querytypes.FilterOperatorGreaterThan:
			condition = fmt.Sprintf("%s > ?", keyExpression)
		case querytypes.FilterOperatorGreaterThanOrEq:
			condition = fmt.Sprintf("%s >= ?", keyExpression)
		case querytypes.FilterOperatorLessThan:
			condition = fmt.Sprintf("%s < ?", keyExpression)
		case querytypes.FilterOperatorLessThanOrEq:
			condition = fmt.Sprintf("%s <= ?", keyExpression)
		default:
			return nil, nil, fmt.Errorf("unsupported filter operator: %s", operator)
		}

		conditions = append(conditions, condition)
		args = append(args, value)
	}
	return conditions, args, nil
}

type Reader struct {
	db       clickhouse.Conn
	logger   *slog.Logger
	metadata metadataReader
}

var _ interfaces.MetricsExplorerReader = (*Reader)(nil)

func New(logger *slog.Logger, db clickhouse.Conn, metadata metadataReader) *Reader {
	return &Reader{db: db, logger: logger, metadata: metadata}
}

func (r *Reader) GetMetricAggregateAttributes(ctx context.Context, orgID valuer.UUID, req *querytypes.AggregateAttributeRequest, skipSignozMetrics bool) (*querytypes.AggregateAttributeResponse, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetMetricAggregateAttributes",
	})
	var response querytypes.AggregateAttributeResponse
	normalized := true
	if constants.IsDotMetricsEnabled {
		normalized = false
	}

	// Query all relevant metric names from time_series_v4, but leave metadata retrieval to cache/db
	query := fmt.Sprintf(
		`SELECT DISTINCT metric_name
		 FROM %s.%s
		 WHERE metric_name ILIKE $1 AND __normalized = $2`,
		siginsightMetricDBName, siginsightTSTableNameV41Day)

	if req.Limit != 0 {
		query = query + fmt.Sprintf(" LIMIT %d;", req.Limit)
	}

	rows, err := r.db.Query(ctx, query, fmt.Sprintf("%%%s%%", req.SearchText), normalized)
	if err != nil {
		r.logger.Error("Error while querying metric names", errorsV2.Attr(err))
		return nil, fmt.Errorf("error while executing metric name query: %s", err.Error())
	}
	defer rows.Close()

	var metricNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("error while scanning metric name: %s", err.Error())
		}
		if skipSignozMetrics && strings.HasPrefix(name, "signoz") {
			continue
		}
		metricNames = append(metricNames, name)
	}

	if len(metricNames) == 0 {
		return &response, nil
	}

	// Get all metadata in one shot
	metadataMap, err := r.metadata.GetUpdatedMetricsMetadata(ctx, orgID, metricNames...)
	if err != nil {
		return &response, fmt.Errorf("error getting updated metrics metadata: %w", err)
	}

	seen := make(map[string]struct{})
	for _, name := range metricNames {
		metadata, ok := metadataMap[name]
		if !ok || metadata == nil {
			return &response, fmt.Errorf("metric metadata not found: %s", name)
		}

		typ := string(metadata.MetricType)
		temporality := string(metadata.Temporality)
		isMonotonic := metadata.IsMonotonic

		// Non-monotonic cumulative sums are treated as gauges
		if typ == "Sum" && !isMonotonic && temporality == string(querytypes.Cumulative) {
			typ = "Gauge"
		}

		// unlike traces/logs `tag`/`resource` type, the `Type` will be metric type
		key := querytypes.AttributeKey{
			Key:      name,
			DataType: querytypes.AttributeKeyDataTypeFloat64,
			Type:     querytypes.AttributeKeyType(typ),
			IsColumn: true,
		}

		if _, ok := seen[name+typ]; ok {
			continue
		}
		seen[name+typ] = struct{}{}
		response.AttributeKeys = append(response.AttributeKeys, key)
	}

	return &response, nil
}

func (r *Reader) GetAllMetricFilterAttributeKeys(ctx context.Context, req *metrics_explorer.FilterKeyRequest) (*[]querytypes.AttributeKey, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetAllMetricFilterAttributeKeys",
	})
	var rows driver.Rows
	var response []querytypes.AttributeKey
	normalized := true
	if constants.IsDotMetricsEnabled {
		normalized = false
	}
	query := fmt.Sprintf("SELECT arrayJoin(tagKeys) AS distinctTagKey FROM (SELECT JSONExtractKeys(labels) AS tagKeys FROM %s.%s WHERE unix_milli >= $1 and __normalized = $2 GROUP BY tagKeys) WHERE distinctTagKey ILIKE $3 AND distinctTagKey NOT LIKE '\\_\\_%%' GROUP BY distinctTagKey", siginsightMetricDBName, siginsightTSTableNameV41Day)
	if req.Limit != 0 {
		query = query + fmt.Sprintf(" LIMIT %d;", req.Limit)
	}
	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err := r.db.Query(valueCtx, query, common.PastDayRoundOff(), normalized, fmt.Sprintf("%%%s%%", req.SearchText)) //only showing past day data
	if err != nil {
		r.logger.Error("Error while executing query", errorsV2.Attr(err))
		return nil, fmt.Errorf("query metric filter attribute keys: %w", err)
	}

	var attributeKey string
	for rows.Next() {
		if err := rows.Scan(&attributeKey); err != nil {
			return nil, fmt.Errorf("scan metric filter attribute key: %w", err)
		}
		key := querytypes.AttributeKey{
			Key:      attributeKey,
			DataType: querytypes.AttributeKeyDataTypeString, // https://github.com/OpenObservability/OpenMetrics/blob/main/proto/openmetrics_data_model.proto#L64-L72.
			Type:     querytypes.AttributeKeyTypeTag,
			IsColumn: false,
		}
		response = append(response, key)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metric filter attribute keys: %w", err)
	}
	return &response, nil
}

func (r *Reader) GetAllMetricFilterAttributeValues(ctx context.Context, req *metrics_explorer.FilterValueRequest) ([]string, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetAllMetricFilterAttributeValues",
	})
	var query string
	var err error
	var rows driver.Rows
	var attributeValues []string
	normalized := true
	if constants.IsDotMetricsEnabled {
		normalized = false
	}

	query = fmt.Sprintf("SELECT JSONExtractString(labels, $1) AS tagValue FROM %s.%s WHERE JSONExtractString(labels, $2) ILIKE $3 AND unix_milli >= $4 AND __normalized = $5 GROUP BY tagValue", siginsightMetricDBName, siginsightTSTableNameV41Day)
	if req.Limit != 0 {
		query = query + fmt.Sprintf(" LIMIT %d;", req.Limit)
	}
	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err = r.db.Query(valueCtx, query, req.FilterKey, req.FilterKey, fmt.Sprintf("%%%s%%", req.SearchText), common.PastDayRoundOff(), normalized) //only showing past day data

	if err != nil {
		r.logger.Error("Error while executing query", errorsV2.Attr(err))
		return nil, fmt.Errorf("query metric filter attribute values: %w", err)
	}
	defer rows.Close()

	var atrributeValue string
	for rows.Next() {
		if err := rows.Scan(&atrributeValue); err != nil {
			return nil, fmt.Errorf("scan metric filter attribute value: %w", err)
		}
		attributeValues = append(attributeValues, atrributeValue)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metric filter attribute values: %w", err)
	}
	return attributeValues, nil
}

func (r *Reader) GetAttributesForMetricName(ctx context.Context, metricName string, start, end *int64, filters *querytypes.FilterSet) (*[]metrics_explorer.Attribute, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetAttributesForMetricName",
	})
	whereClause := ""
	var filterArgs []any
	if filters != nil {
		conditions, args, err := buildMetricFilterConditions(filters, "t")
		if err != nil {
			return nil, err
		}
		if conditions != nil {
			whereClause = "AND " + strings.Join(conditions, " AND ")
		}
		filterArgs = args
	}
	normalized := true
	if constants.IsDotMetricsEnabled {
		normalized = false
	}

	const baseQueryTemplate = `
SELECT
    kv.1 AS key,
    arrayMap(x -> trim(BOTH '"' FROM x), groupUniqArray(1000)(kv.2)) AS values,
    length(groupUniqArray(10000)(kv.2)) AS valueCount
FROM %s.%s
ARRAY JOIN arrayFilter(x -> NOT startsWith(x.1, '__'), JSONExtractKeysAndValuesRaw(labels)) AS kv
WHERE metric_name = ? AND __normalized=? %s`

	var args []interface{}
	args = append(args, metricName)
	tableName := siginsightTSTableNameV41Week

	args = append(args, normalized)
	args = append(args, filterArgs...)

	if start != nil && end != nil {
		st, en, tsTable, _ := utils.WhichTSTableToUse(*start, *end)
		*start, *end, tableName = st, en, tsTable
		args = append(args, *start, *end)
	} else if start == nil && end == nil {
		tableName = siginsightTSTableNameV41Week
	}

	query := fmt.Sprintf(baseQueryTemplate, siginsightMetricDBName, tableName, whereClause)

	if start != nil && end != nil {
		query += " AND unix_milli BETWEEN ? AND ?"
	}

	query += "\nGROUP BY kv.1\nORDER BY valueCount DESC;"

	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err := r.db.Query(valueCtx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query metric attributes: %w", err)
	}
	defer rows.Close()

	var attributesList []metrics_explorer.Attribute
	for rows.Next() {
		var attr metrics_explorer.Attribute
		if err := rows.Scan(&attr.Key, &attr.Value, &attr.ValueCount); err != nil {
			return nil, fmt.Errorf("scan metric attribute: %w", err)
		}
		attributesList = append(attributesList, attr)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metric attributes: %w", err)
	}

	return &attributesList, nil
}

func (r *Reader) GetNameSimilarity(ctx context.Context, req *metrics_explorer.RelatedMetricsRequest) (map[string]metrics_explorer.RelatedMetricsScore, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetNameSimilarity",
	})
	start, end, tsTable, _ := utils.WhichTSTableToUse(req.Start, req.End)

	normalized := true
	if constants.IsDotMetricsEnabled {
		normalized = false
	}

	query := fmt.Sprintf(`
		SELECT
			metric_name,
			any(type) as type,
		    any(temporality) as temporality,
		    any(is_monotonic) as monotonic,
			1 - (levenshteinDistance(?, metric_name) / greatest(NULLIF(length(?), 0), NULLIF(length(metric_name), 0))) AS name_similarity
		FROM %s.%s
		WHERE metric_name != ?
		  AND unix_milli BETWEEN ? AND ?
		 AND NOT startsWith(metric_name, 'signoz')
		AND __normalized = ?
		GROUP BY metric_name
		ORDER BY name_similarity DESC
		LIMIT 30;`,
		siginsightMetricDBName, tsTable)

	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err := r.db.Query(valueCtx, query, req.CurrentMetricName, req.CurrentMetricName, req.CurrentMetricName, start, end, normalized)
	if err != nil {
		return nil, fmt.Errorf("query metric name similarity: %w", err)
	}
	defer rows.Close()

	result := make(map[string]metrics_explorer.RelatedMetricsScore)
	for rows.Next() {
		var metric string
		var sim float64
		var metricType querytypes.MetricType
		var temporality querytypes.Temporality
		var isMonotonic bool
		if err := rows.Scan(&metric, &metricType, &temporality, &isMonotonic, &sim); err != nil {
			return nil, fmt.Errorf("scan metric name similarity: %w", err)
		}
		result[metric] = metrics_explorer.RelatedMetricsScore{
			NameSimilarity: sim,
			MetricType:     metricType,
			Temporality:    temporality,
			IsMonotonic:    isMonotonic,
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metric name similarity: %w", err)
	}

	return result, nil
}

func (r *Reader) GetAttributeSimilarity(ctx context.Context, req *metrics_explorer.RelatedMetricsRequest) (map[string]metrics_explorer.RelatedMetricsScore, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetAttributeSimilarity",
	})
	start, end, tsTable, _ := utils.WhichTSTableToUse(req.Start, req.End)

	normalized := true
	if constants.IsDotMetricsEnabled {
		normalized = false
	}

	// Get target labels
	extractedLabelsQuery := fmt.Sprintf(`
		SELECT
			kv.1 AS label_key,
			topK(10)(JSONExtractString(kv.2)) AS label_values
		FROM %s.%s
		ARRAY JOIN JSONExtractKeysAndValuesRaw(labels) AS kv
		WHERE metric_name = ?
		  AND unix_milli between ? and ?
		  AND NOT startsWith(kv.1, '__')
		AND NOT startsWith(metric_name, 'signoz_')
		AND __normalized = ?
		GROUP BY label_key
		LIMIT 50`, siginsightMetricDBName, tsTable)

	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err := r.db.Query(valueCtx, extractedLabelsQuery, req.CurrentMetricName, start, end, normalized)
	if err != nil {
		return nil, fmt.Errorf("query metric attribute labels: %w", err)
	}
	defer rows.Close()

	var targetKeys []string
	var targetValues []string
	for rows.Next() {
		var key string
		var value []string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan metric attribute labels: %w", err)
		}
		targetKeys = append(targetKeys, key)
		targetValues = append(targetValues, value...)
	}

	priorityPairs := make(clickhouse.ArraySet, 0, len(req.Filters.Items))
	for _, f := range req.Filters.Items {
		if f.Operator == querytypes.FilterOperatorEqual {
			priorityPairs = append(priorityPairs, clickhouse.GroupSet{Value: []any{f.Key.Key, f.Value}})
		}
	}

	candidateLabelsQuery := fmt.Sprintf(`
		WITH
			arrayDistinct(?) AS filter_keys,
			arrayDistinct(?) AS filter_values,
			? AS priority_pairs_input,
			? AS priority_multiplier
		SELECT
			metric_name,
			any(type) as type,
			any(temporality) as temporality,
			any(is_monotonic) as monotonic,
			SUM(
				arraySum(
					kv -> if(has(filter_keys, kv.1) AND has(filter_values, kv.2), 1, 0),
					JSONExtractKeysAndValues(labels, 'String')
				)
			)::UInt64 AS raw_match_count,
			SUM(
				arraySum(
					kv ->
						if(
							arrayExists(pr -> pr.1 = kv.1 AND pr.2 = kv.2, priority_pairs_input),
							priority_multiplier,
							0
						),
					JSONExtractKeysAndValues(labels, 'String')
				)
			)::UInt64 AS weighted_match_count,
		toJSONString(
			arrayDistinct(
				arrayFlatten(
					groupArray(
						arrayFilter(
							kv -> arrayExists(pr -> pr.1 = kv.1 AND pr.2 = kv.2, priority_pairs_input),
							JSONExtractKeysAndValues(labels, 'String')
						)
					)
				)
			)
		) AS priority_pairs
		FROM %s.%s
		WHERE rand() %% 100 < 10
		AND unix_milli between ? and ?
		AND NOT startsWith(metric_name, 'signoz_')
		AND __normalized = ?
		GROUP BY metric_name
		ORDER BY weighted_match_count DESC, raw_match_count DESC
		LIMIT 30
		`,
		siginsightMetricDBName, tsTable)

	rows, err = r.db.Query(valueCtx, candidateLabelsQuery, targetKeys, targetValues, priorityPairs, 2, start, end, normalized)
	if err != nil {
		return nil, fmt.Errorf("query metric attribute similarity: %w", err)
	}
	defer rows.Close()

	result := make(map[string]metrics_explorer.RelatedMetricsScore)
	attributeMap := make(map[string]uint64)

	for rows.Next() {
		var metric string
		var metricType querytypes.MetricType
		var temporality querytypes.Temporality
		var isMonotonic bool
		var weightedMatchCount, rawMatchCount uint64
		var priorityPairsJSON string

		if err := rows.Scan(&metric, &metricType, &temporality, &isMonotonic, &rawMatchCount, &weightedMatchCount, &priorityPairsJSON); err != nil {
			return nil, fmt.Errorf("scan metric attribute similarity: %w", err)
		}

		attributeMap[metric] = weightedMatchCount + (rawMatchCount)/10
		var priorityPairs [][]string
		if err := json.Unmarshal([]byte(priorityPairsJSON), &priorityPairs); err != nil {
			priorityPairs = [][]string{}
		}

		result[metric] = metrics_explorer.RelatedMetricsScore{
			AttributeSimilarity: float64(attributeMap[metric]), // Will be normalized later
			Filters:             priorityPairs,
			MetricType:          metricType,
			Temporality:         temporality,
			IsMonotonic:         isMonotonic,
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metric attribute similarity: %w", err)
	}

	// Normalize the attribute similarity scores
	normalizeMap := utils.NormalizeMap(attributeMap)
	for metric := range result {
		if score, exists := normalizeMap[metric]; exists {
			metricScore := result[metric]
			metricScore.AttributeSimilarity = score
			result[metric] = metricScore
		}
	}

	return result, nil
}

func (r *Reader) GetMetricsAllResourceAttributes(ctx context.Context, start int64, end int64) (map[string]uint64, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetMetricsAllResourceAttributes",
	})
	start, end, attTable, _ := utils.WhichAttributesTableToUse(start, end)
	query := fmt.Sprintf(`SELECT
    key,
    count(distinct value) AS distinct_value_count
FROM (
    SELECT key, value
    FROM %s.%s
    ARRAY JOIN
        arrayConcat(mapKeys(resource_attributes)) AS key,
        arrayConcat(mapValues(resource_attributes)) AS value
    WHERE unix_milli between ? and ?
)
GROUP BY key
ORDER BY distinct_value_count DESC;`, siginsightMetadataDBName, attTable)
	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err := r.db.Query(valueCtx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("query metric resource attributes: %w", err)
	}
	attributes := make(map[string]uint64)
	for rows.Next() {
		var attrs string
		var uniqCount uint64

		if err := rows.Scan(&attrs, &uniqCount); err != nil {
			return nil, fmt.Errorf("scan metric resource attribute: %w", err)
		}
		attributes[attrs] = uniqCount
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate metric resource attributes: %w", err)
	}
	return attributes, nil
}

func (r *Reader) GetInspectMetrics(ctx context.Context, req *metrics_explorer.InspectMetricsRequest, fingerprints []string) (*metrics_explorer.InspectMetricsResponse, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetInspectMetrics",
	})
	start, end, _, localTsTable := utils.WhichTSTableToUse(req.Start, req.End)
	fingerprintsString := strings.Join(fingerprints, ",")
	query := fmt.Sprintf(`SELECT
                fingerprint,
                labels,
                unix_milli,
                value as per_series_value
        FROM
                siginsight_metrics.samples_v4
        INNER JOIN (
                SELECT DISTINCT
                        fingerprint,
                        labels
                FROM
                        %s.%s
                WHERE
                        fingerprint in (%s)
                        AND unix_milli >= ?
                        AND unix_milli < ?) as filtered_time_series
                USING fingerprint
        WHERE
                metric_name  = ?
                AND unix_milli >= ?
                AND unix_milli < ?
                ORDER BY fingerprint DESC, unix_milli DESC`, siginsightMetricDBName, localTsTable, fingerprintsString)
	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	rows, err := r.db.Query(valueCtx, query, start, end, req.MetricName, start, end)
	if err != nil {
		return nil, fmt.Errorf("query inspect metrics: %w", err)
	}
	defer rows.Close()

	seriesMap := make(map[uint64]*timeseriestypes.Series)

	for rows.Next() {
		var fingerprint uint64
		var labelsJSON string
		var unixMilli int64
		var perSeriesValue float64

		if err := rows.Scan(&fingerprint, &labelsJSON, &unixMilli, &perSeriesValue); err != nil {
			return nil, fmt.Errorf("scan inspect metrics: %w", err)
		}

		var labelsMap map[string]string
		if err := json.Unmarshal([]byte(labelsJSON), &labelsMap); err != nil {
			return nil, fmt.Errorf("decode inspect metric labels: %w", err)
		}

		// Filter out keys starting with "__"
		filteredLabelsMap := make(map[string]string)
		for k, v := range labelsMap {
			if !strings.HasPrefix(k, "__") {
				filteredLabelsMap[k] = v
			}
		}

		var labelsArray []map[string]string
		for k, v := range filteredLabelsMap {
			labelsArray = append(labelsArray, map[string]string{k: v})
		}

		// Check if we already have a Series for this fingerprint.
		series, exists := seriesMap[fingerprint]
		if !exists {
			series = &timeseriestypes.Series{
				Labels:      filteredLabelsMap,
				LabelsArray: labelsArray,
				Points:      []timeseriestypes.Point{},
			}
			seriesMap[fingerprint] = series
		}

		series.Points = append(series.Points, timeseriestypes.Point{
			Timestamp: unixMilli,
			Value:     perSeriesValue,
		})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inspect metrics: %w", err)
	}

	var seriesList []timeseriestypes.Series
	for _, s := range seriesMap {
		seriesList = append(seriesList, *s)
	}

	return &metrics_explorer.InspectMetricsResponse{
		Series: &seriesList,
	}, nil
}

func (r *Reader) GetInspectMetricsFingerprints(ctx context.Context, attributes []string, req *metrics_explorer.InspectMetricsRequest) ([]string, error) {
	ctx = ctxtypes.NewContextWithCommentVals(ctx, map[string]string{
		instrumentationtypes.TelemetrySignal:  telemetrytypes.SignalMetrics.StringValue(),
		instrumentationtypes.CodeNamespace:    "clickhouse-reader",
		instrumentationtypes.CodeFunctionName: "GetInspectMetricsFingerprints",
	})
	// Build dynamic key selections and JSON extracts
	var jsonExtracts []string
	var groupBys []string

	attributeArgs := make([]any, 0, len(attributes))
	for i, attr := range attributes {
		keyAlias := fmt.Sprintf("key%d", i+1)
		jsonExtracts = append(jsonExtracts, fmt.Sprintf("JSONExtractString(labels, ?) AS %s", keyAlias))
		groupBys = append(groupBys, keyAlias)
		attributeArgs = append(attributeArgs, attr)
	}

	conditions, filterArgs, err := buildMetricFilterConditions(&req.Filters, "")
	if err != nil {
		return nil, err
	}
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "AND " + strings.Join(conditions, " AND ")
	}

	start, end, tsTable, _ := utils.WhichTSTableToUse(req.Start, req.End)
	query := fmt.Sprintf(`
        SELECT
    arrayDistinct(groupArray(toString(fingerprint))) AS fingerprints
FROM
(
    SELECT
        metric_name, labels, fingerprint,
        %s
    FROM %s.%s
    WHERE metric_name = ?
      AND unix_milli BETWEEN ? AND ?
    %s
)
GROUP BY %s
ORDER BY length(fingerprints) DESC, rand()
LIMIT 40`, // added rand to get diff value every time we run this query
		strings.Join(jsonExtracts, ", "),
		siginsightMetricDBName, tsTable,
		whereClause,
		strings.Join(groupBys, ", "))
	valueCtx := context.WithValue(ctx, "clickhouse_max_threads", constants.MetricsExplorerClickhouseThreads)
	queryArgs := append(attributeArgs,
		req.MetricName,
		start,
		end,
	)
	queryArgs = append(queryArgs, filterArgs...)
	rows, err := r.db.Query(valueCtx, query, queryArgs...)
	if err != nil {
		return nil, fmt.Errorf("query inspect metric fingerprints: %w", err)
	}
	defer rows.Close()

	var fingerprints []string
	for rows.Next() {
		// Create dynamic scanning based on number of attributes
		var batch []string

		if err := rows.Scan(&batch); err != nil {
			return nil, fmt.Errorf("scan inspect metric fingerprints: %w", err)
		}

		remaining := 40 - len(fingerprints)
		if remaining <= 0 {
			break
		}

		// if this batch would overshoot, only take as many as we need
		if len(batch) > remaining {
			fingerprints = append(fingerprints, batch[:remaining]...)
			break
		}

		// otherwise take the whole batch and keep going
		fingerprints = append(fingerprints, batch...)

	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate inspect metric fingerprints: %w", err)
	}

	return fingerprints, nil
}
