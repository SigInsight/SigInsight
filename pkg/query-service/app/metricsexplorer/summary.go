package metricsexplorer

import (
	"context"
	"errors"
	"sort"

	"strings"
	"time"

	"log/slog"

	"golang.org/x/sync/errgroup"

	"github.com/SigNoz/signoz/pkg/query-service/interfaces"
	"github.com/SigNoz/signoz/pkg/query-service/model"
	"github.com/SigNoz/signoz/pkg/query-service/model/metrics_explorer"
	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
	"github.com/SigNoz/signoz/pkg/query-service/rules"
	"github.com/SigNoz/signoz/pkg/valuer"
)

type SummaryService struct {
	reader       interfaces.Reader
	rulesManager *rules.Manager
}

func NewSummaryService(reader interfaces.Reader, alertManager *rules.Manager) *SummaryService {
	return &SummaryService{reader: reader, rulesManager: alertManager}
}

func (receiver *SummaryService) FilterKeys(ctx context.Context, params *metrics_explorer.FilterKeyRequest) (*metrics_explorer.FilterKeyResponse, *model.ApiError) {
	var response metrics_explorer.FilterKeyResponse
	keys, apiError := receiver.reader.GetAllMetricFilterAttributeKeys(ctx, params)
	if apiError != nil {
		return nil, apiError
	}
	response.AttributeKeys = *keys
	var availableColumnFilter []string
	for key := range metrics_explorer.AvailableColumnFilterMap {
		availableColumnFilter = append(availableColumnFilter, key)
	}
	response.MetricColumns = availableColumnFilter
	return &response, nil
}

func (receiver *SummaryService) FilterValues(ctx context.Context, orgID valuer.UUID, params *metrics_explorer.FilterValueRequest) (*metrics_explorer.FilterValueResponse, *model.ApiError) {
	var response metrics_explorer.FilterValueResponse
	switch params.FilterKey {
	case "metric_name":
		var filterValues []string
		request := querytypes.AggregateAttributeRequest{DataSource: querytypes.DataSourceMetrics, SearchText: params.SearchText, Limit: params.Limit}
		attributes, err := receiver.reader.GetMetricAggregateAttributes(ctx, orgID, &request, true)
		if err != nil {
			return nil, model.InternalError(err)
		}
		for _, item := range attributes.AttributeKeys {
			filterValues = append(filterValues, item.Key)
		}
		response.FilterValues = filterValues
		return &response, nil
	case "metric_unit":
		attributes, apiErr := receiver.reader.GetAllMetricFilterUnits(ctx, params)
		if apiErr != nil {
			return nil, apiErr
		}
		response.FilterValues = attributes
		return &response, nil
	case "metric_type":
		attributes, apiErr := receiver.reader.GetAllMetricFilterTypes(ctx, params)
		if apiErr != nil {
			return nil, apiErr
		}
		response.FilterValues = attributes
		return &response, nil
	default:
		attributes, apiErr := receiver.reader.GetAllMetricFilterAttributeValues(ctx, params)
		if apiErr != nil {
			return nil, apiErr
		}
		response.FilterValues = attributes
		return &response, nil
	}
}

func (receiver *SummaryService) GetRelatedMetrics(ctx context.Context, params *metrics_explorer.RelatedMetricsRequest) (*metrics_explorer.RelatedMetricsResponse, *model.ApiError) {
	// Get name similarity scores
	nameSimilarityScores, err := receiver.reader.GetNameSimilarity(ctx, params)
	if err != nil {
		return nil, err
	}

	attrCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	attrSimilarityScores, err := receiver.reader.GetAttributeSimilarity(attrCtx, params)
	if err != nil {
		// If we hit a deadline exceeded error, proceed with only name similarity
		if errors.Is(err.Err, context.DeadlineExceeded) {
			slog.Warn("attribute similarity calculation timed out, proceeding with name similarity only")
			attrSimilarityScores = make(map[string]metrics_explorer.RelatedMetricsScore)
		} else {
			return nil, err
		}
	}

	// Combine scores and compute final scores
	finalScores := make(map[string]float64)
	relatedMetricsMap := make(map[string]metrics_explorer.RelatedMetricsScore)

	// Merge name and attribute similarity scores
	for metric, nameScore := range nameSimilarityScores {
		attrScore, exists := attrSimilarityScores[metric]
		if exists {
			relatedMetricsMap[metric] = metrics_explorer.RelatedMetricsScore{
				NameSimilarity:      nameScore.NameSimilarity,
				AttributeSimilarity: attrScore.AttributeSimilarity,
				Filters:             attrScore.Filters,
				MetricType:          attrScore.MetricType,
				Temporality:         attrScore.Temporality,
				IsMonotonic:         attrScore.IsMonotonic,
			}
		} else {
			relatedMetricsMap[metric] = nameScore
		}
		finalScores[metric] = nameScore.NameSimilarity*0.7 + relatedMetricsMap[metric].AttributeSimilarity*0.3
	}

	// Handle metrics that are only present in attribute similarity scores
	for metric, attrScore := range attrSimilarityScores {
		if _, exists := nameSimilarityScores[metric]; !exists {
			relatedMetricsMap[metric] = metrics_explorer.RelatedMetricsScore{
				AttributeSimilarity: attrScore.AttributeSimilarity,
				Filters:             attrScore.Filters,
				MetricType:          attrScore.MetricType,
				Temporality:         attrScore.Temporality,
				IsMonotonic:         attrScore.IsMonotonic,
			}
			finalScores[metric] = attrScore.AttributeSimilarity * 0.3
		}
	}

	type metricScore struct {
		Name  string
		Score float64
	}
	var sortedScores []metricScore
	for metric, score := range finalScores {
		sortedScores = append(sortedScores, metricScore{
			Name:  metric,
			Score: score,
		})
	}

	sort.Slice(sortedScores, func(i, j int) bool {
		return sortedScores[i].Score > sortedScores[j].Score
	})

	metricNames := make([]string, len(sortedScores))
	for i, ms := range sortedScores {
		metricNames[i] = ms.Name
	}

	// Fetch alerts for related metrics.
	g, ctx := errgroup.WithContext(ctx)

	alertsRelatedData := make(map[string][]metrics_explorer.Alert)

	g.Go(func() error {
		rulesData, apiError := receiver.rulesManager.GetAlertDetailsForMetricNames(ctx, metricNames)
		if apiError != nil {
			return apiError
		}
		for s, gettableRules := range rulesData {
			var metricsRules []metrics_explorer.Alert
			for _, rule := range gettableRules {
				metricsRules = append(metricsRules, metrics_explorer.Alert{AlertID: rule.Id, AlertName: rule.AlertName})
			}
			alertsRelatedData[s] = metricsRules
		}
		return nil
	})

	// Check for context cancellation before waiting
	if ctx.Err() != nil {
		return nil, &model.ApiError{Typ: "ContextCanceled", Err: ctx.Err()}
	}

	if err := g.Wait(); err != nil {
		var apiErr *model.ApiError
		if errors.As(err, &apiErr) {
			return nil, apiErr
		}
		return nil, &model.ApiError{Typ: "InternalError", Err: err}
	}

	// Build response
	var response metrics_explorer.RelatedMetricsResponse
	for _, ms := range sortedScores {
		relatedMetric := metrics_explorer.RelatedMetrics{
			Name:  ms.Name,
			Query: getQueryRangeForRelateMetricsList(ms.Name, relatedMetricsMap[ms.Name]),
		}
		if alerts, ok := alertsRelatedData[ms.Name]; ok {
			relatedMetric.Alerts = alerts
		}
		response.RelatedMetrics = append(response.RelatedMetrics, relatedMetric)
	}

	return &response, nil
}

func getQueryRangeForRelateMetricsList(metricName string, scores metrics_explorer.RelatedMetricsScore) *querytypes.BuilderQuery {
	var filterItems []querytypes.FilterItem
	for _, pair := range scores.Filters {
		if len(pair) < 2 {
			continue // Skip invalid filter pairs.
		}
		filterItem := querytypes.FilterItem{
			Key: querytypes.AttributeKey{
				Key:      pair[0], // Default type, or you can use querytypes.AttributeKeyTypeUnspecified.
				IsColumn: false,
				IsJSON:   false,
			},
			Value:    pair[1],
			Operator: querytypes.FilterOperatorEqual, // Using "=" as the operator.
		}
		filterItems = append(filterItems, filterItem)
	}

	// If there are any filters, combine them with an "AND" operator.
	var filters *querytypes.FilterSet
	if len(filterItems) > 0 {
		filters = &querytypes.FilterSet{
			Operator: "AND",
			Items:    filterItems,
		}
	}

	// Create the BuilderQuery. Here we set the QueryName to the metric name.
	query := querytypes.BuilderQuery{
		QueryName:  metricName,
		DataSource: querytypes.DataSourceMetrics,
		Expression: metricName, // Using metric name as expression
		Filters:    filters,
	}

	if scores.MetricType == querytypes.MetricTypeSum && !scores.IsMonotonic && scores.Temporality == querytypes.Cumulative {
		scores.MetricType = querytypes.MetricTypeGauge
	}

	switch scores.MetricType {
	case querytypes.MetricTypeGauge:
		query.TimeAggregation = querytypes.TimeAggregationAvg
		query.SpaceAggregation = querytypes.SpaceAggregationAvg
	case querytypes.MetricTypeSum:
		query.TimeAggregation = querytypes.TimeAggregationRate
		query.SpaceAggregation = querytypes.SpaceAggregationSum
	case querytypes.MetricTypeHistogram:
		query.SpaceAggregation = querytypes.SpaceAggregationPercentile95
	}

	query.AggregateAttribute = querytypes.AttributeKey{
		Key:  metricName,
		Type: querytypes.AttributeKeyType(scores.MetricType),
	}

	query.StepInterval = 60

	return &query
}

func (receiver *SummaryService) GetInspectMetrics(ctx context.Context, params *metrics_explorer.InspectMetricsRequest) (*metrics_explorer.InspectMetricsResponse, *model.ApiError) {
	// Capture the original context.
	parentCtx := ctx

	// Create an errgroup using the original context.
	g, egCtx := errgroup.WithContext(ctx)

	var attributes []metrics_explorer.Attribute
	var resourceAttrs map[string]uint64

	// Run the two queries concurrently using the derived context.
	g.Go(func() error {
		attrs, apiErr := receiver.reader.GetAttributesForMetricName(egCtx, params.MetricName, &params.Start, &params.End, &params.Filters)
		if apiErr != nil {
			return apiErr
		}
		if attrs != nil {
			attributes = *attrs
		}
		return nil
	})

	g.Go(func() error {
		resAttrs, apiErr := receiver.reader.GetMetricsAllResourceAttributes(egCtx, params.Start, params.End)
		if apiErr != nil {
			return apiErr
		}
		if resAttrs != nil {
			resourceAttrs = resAttrs
		}
		return nil
	})

	// Wait for the concurrent operations to complete.
	if err := g.Wait(); err != nil {
		return nil, &model.ApiError{Typ: "InternalError", Err: err}
	}

	// Use the parentCtx (or create a new context from it) for the rest of the calls.
	if parentCtx.Err() != nil {
		return nil, &model.ApiError{Typ: "ContextCanceled", Err: parentCtx.Err()}
	}

	// Build a set of attribute keys for O(1) lookup.
	attributeKeys := make(map[string]struct{})
	for _, attr := range attributes {
		attributeKeys[attr.Key] = struct{}{}
	}

	// Filter resource attributes that are present in attributes.
	var validAttrs []string
	for attrName := range resourceAttrs {
		normalizedAttrName := strings.ReplaceAll(attrName, ".", "_")
		if _, ok := attributeKeys[normalizedAttrName]; ok {
			validAttrs = append(validAttrs, normalizedAttrName)
		}
	}

	// Get top 3 resource attributes (or use top attributes by valueCount if none match).
	if len(validAttrs) > 3 {
		validAttrs = validAttrs[:3]
	} else if len(validAttrs) == 0 {
		sort.Slice(attributes, func(i, j int) bool {
			return attributes[i].ValueCount > attributes[j].ValueCount
		})
		for i := 0; i < len(attributes) && i < 3; i++ {
			validAttrs = append(validAttrs, attributes[i].Key)
		}
	}
	fingerprints, apiError := receiver.reader.GetInspectMetricsFingerprints(parentCtx, validAttrs, params)
	if apiError != nil {
		return nil, apiError
	}

	baseResponse, apiErr := receiver.reader.GetInspectMetrics(parentCtx, params, fingerprints)
	if apiErr != nil {
		return nil, apiErr
	}

	return baseResponse, nil
}
