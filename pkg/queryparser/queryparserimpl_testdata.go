package queryparser

var (
	builderQueryWithGrouping = `
	[
	    {
	      "type":"builder_query",
	      "spec":{
	        "name":"A",
	        "signal":"metrics",
	        "stepInterval":null,
	        "disabled":false,
	        "filter":{"expression":""},
	        "groupBy":[
	          {"name":"service_name","fieldDataType":"","fieldContext":""},
	          {"name":"env","fieldDataType":"","fieldContext":""}
	        ],
	        "aggregations":[
	          {"metricName":"test_metric_cardinality","timeAggregation":"count","spaceAggregation":"sum"},
	          {"metricName":"cpu_usage_total","timeAggregation":"avg","spaceAggregation":"avg"}
	        ]
	      }
	    }
	  ]
	`

	builderQuerySingleGrouping = `
	[
	    {
	      "type":"builder_query",
	      "spec":{
	        "name":"B",
	        "signal":"metrics",
	        "stepInterval":null,
	        "disabled":false,
	        "groupBy":[
	          {"name":"namespace","fieldDataType":"","fieldContext":""}
	        ],
	        "aggregations":[
	          {"metricName":"latency_p50","timeAggregation":"avg","spaceAggregation":"max"}
	        ]
	      }
	    }
	  ]
	`

	builderQueryNoGrouping = `
	[
	    {
	      "type":"builder_query",
	      "spec":{
	        "name":"C",
	        "signal":"metrics",
	        "stepInterval":null,
	        "disabled":false,
	        "groupBy":[],
	        "aggregations":[
	          {"metricName":"disk_usage_total","timeAggregation":"sum","spaceAggregation":"sum"}
	        ]
	      }
	    }
	  ]
	`

	builderQueryWithFormula = `
	[
	    {
	      "type":"builder_query",
	      "spec":{
	        "name":"A",
	        "signal":"metrics",
	        "stepInterval":null,
	        "disabled":false,
	        "groupBy":[],
	        "aggregations":[
	          {"metricName":"cpu_usage","timeAggregation":"avg","spaceAggregation":"sum"}
	        ]
	      }
	    },
	    {
	      "type":"builder_query",
	      "spec":{
	        "name":"B",
	        "signal":"metrics",
	        "stepInterval":null,
	        "disabled":false,
	        "groupBy":[],
	        "aggregations":[
	          {"metricName":"mem_usage","timeAggregation":"avg","spaceAggregation":"sum"}
	        ]
	      }
	    },
	    {
	      "type":"builder_formula",
	      "spec":{
	        "name":"F1",
	        "expression":"A + B"
	      }
	    }
	  ]
	`

	builderQueryWithFormulaAndGroupBy = `
	[
	    {
	      "type":"builder_query",
	      "spec":{
	        "name":"A",
	        "signal":"metrics",
	        "stepInterval":null,
	        "disabled":false,
	        "groupBy":[
	          {"name":"host","fieldDataType":"","fieldContext":""},
			  {"name":"region","fieldDataType":"","fieldContext":""}
	        ],
	        "aggregations":[
	          {"metricName":"cpu","timeAggregation":"avg","spaceAggregation":"sum"}
	        ]
	      }
	    },
	    {
	      "type":"builder_query",
	      "spec":{
	        "name":"B",
	        "signal":"metrics",
	        "stepInterval":null,
	        "disabled":false,
	        "groupBy":[
	          {"name":"host","fieldDataType":"","fieldContext":""},
			  {"name":"instance","fieldDataType":"","fieldContext":""}
	        ],
	        "aggregations":[
	          {"metricName":"mem","timeAggregation":"avg","spaceAggregation":"sum"}
	        ]
	      }
	    },
	    {
	      "type":"builder_formula",
	      "spec":{
	        "name":"F1",
	        "expression":"A + B"
	      }
	    }
	  ]
	`

	builderQueryWithFormulaSameQuery = `
	[
	    {
	      "type":"builder_query",
	      "spec":{
	        "name":"A",
	        "signal":"metrics",
	        "stepInterval":null,
	        "disabled":false,
	        "groupBy":[],
	        "aggregations":[
	          {"metricName":"disk_used","timeAggregation":"sum","spaceAggregation":"sum"}
	        ]
	      }
	    },
	    {
	      "type":"builder_formula",
	      "spec":{
	        "name":"F1",
	        "expression":"A + A"
	      }
	    }
	  ]
	`
)
