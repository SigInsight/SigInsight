package contextlinks

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"

	"github.com/SigNoz/signoz/pkg/query-service/model/querytypes"
)

func PrepareLinksToTracesV5(start, end time.Time, whereClause string) string {

	// Traces list view expects time in nanoseconds
	tr := URLShareableTimeRange{
		Start:    start.UnixNano(),
		End:      end.UnixNano(),
		PageSize: 100,
	}

	options := URLShareableOptions{}

	period, _ := json.Marshal(tr)
	urlEncodedTimeRange := url.QueryEscape(string(period))

	linkQuery := LinkQuery{
		BuilderQuery: querytypes.BuilderQuery{
			DataSource:         querytypes.DataSourceTraces,
			QueryName:          "A",
			AggregateOperator:  querytypes.AggregateOperatorNoOp,
			AggregateAttribute: querytypes.AttributeKey{},
			Expression:         "A",
			Disabled:           false,
			Having:             []querytypes.Having{},
			StepInterval:       60,
		},
		Filter: &FilterExpression{Expression: whereClause},
	}

	urlData := URLShareableCompositeQuery{
		QueryType: string(querytypes.QueryTypeBuilder),
		Builder: URLShareableBuilderQuery{
			QueryData: []LinkQuery{
				linkQuery,
			},
			QueryFormulas: make([]string, 0),
		},
	}

	data, _ := json.Marshal(urlData)
	compositeQuery := url.QueryEscape(url.QueryEscape(string(data)))

	optionsData, _ := json.Marshal(options)
	urlEncodedOptions := url.QueryEscape(string(optionsData))

	return fmt.Sprintf("compositeQuery=%s&timeRange=%s&startTime=%d&endTime=%d&options=%s", compositeQuery, urlEncodedTimeRange, tr.Start, tr.End, urlEncodedOptions)
}

func PrepareLinksToLogsV5(start, end time.Time, whereClause string) string {

	// Logs list view expects time in milliseconds
	tr := URLShareableTimeRange{
		Start:    start.UnixMilli(),
		End:      end.UnixMilli(),
		PageSize: 100,
	}

	options := URLShareableOptions{}

	period, _ := json.Marshal(tr)
	urlEncodedTimeRange := url.QueryEscape(string(period))

	linkQuery := LinkQuery{
		BuilderQuery: querytypes.BuilderQuery{
			DataSource:         querytypes.DataSourceLogs,
			QueryName:          "A",
			AggregateOperator:  querytypes.AggregateOperatorNoOp,
			AggregateAttribute: querytypes.AttributeKey{},
			Expression:         "A",
			Disabled:           false,
			Having:             []querytypes.Having{},
			StepInterval:       60,
		},
		Filter: &FilterExpression{Expression: whereClause},
	}

	urlData := URLShareableCompositeQuery{
		QueryType: string(querytypes.QueryTypeBuilder),
		Builder: URLShareableBuilderQuery{
			QueryData: []LinkQuery{
				linkQuery,
			},
			QueryFormulas: make([]string, 0),
		},
	}

	data, _ := json.Marshal(urlData)
	compositeQuery := url.QueryEscape(url.QueryEscape(string(data)))

	optionsData, _ := json.Marshal(options)
	urlEncodedOptions := url.QueryEscape(string(optionsData))

	return fmt.Sprintf("compositeQuery=%s&timeRange=%s&startTime=%d&endTime=%d&options=%s", compositeQuery, urlEncodedTimeRange, tr.Start, tr.End, urlEncodedOptions)
}
