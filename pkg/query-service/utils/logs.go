package utils

import "github.com/SigNoz/signoz/pkg/query-service/model/querytypes"

const HOUR_NANO = int64(3600000000000)

type LogsListTsRange struct {
	Start int64
	End   int64
}

func GetListTsRanges(start, end int64) []LogsListTsRange {
	startNano := GetEpochNanoSecs(start)
	endNano := GetEpochNanoSecs(end)
	result := []LogsListTsRange{}

	if endNano-startNano > HOUR_NANO {
		bucket := HOUR_NANO
		tStartNano := endNano - bucket

		complete := false
		for {
			result = append(result, LogsListTsRange{Start: tStartNano, End: endNano})
			if complete {
				break
			}

			bucket = bucket * 2
			endNano = tStartNano
			tStartNano = tStartNano - bucket

			// break condition
			if tStartNano <= startNano {
				complete = true
				tStartNano = startNano
			}
		}
	} else {
		result = append(result, LogsListTsRange{Start: start, End: end})
	}
	return result
}

// This tries to see all possible fields that it can fall back to if some meta is missing
// check Test_GenerateEnrichmentKeys for example
func GenerateEnrichmentKeys(field querytypes.AttributeKey) []string {
	names := []string{}
	if field.Type != querytypes.AttributeKeyTypeUnspecified && field.DataType != querytypes.AttributeKeyDataTypeUnspecified {
		names = append(names, field.Key+"##"+field.Type.String()+"##"+field.DataType.String())
		return names
	}

	types := []querytypes.AttributeKeyType{}
	dTypes := []querytypes.AttributeKeyDataType{}
	if field.Type != querytypes.AttributeKeyTypeUnspecified {
		types = append(types, field.Type)
	} else {
		types = append(types, querytypes.AttributeKeyTypeTag, querytypes.AttributeKeyTypeResource)
	}
	if field.DataType != querytypes.AttributeKeyDataTypeUnspecified {
		dTypes = append(dTypes, field.DataType)
	} else {
		dTypes = append(dTypes, querytypes.AttributeKeyDataTypeFloat64, querytypes.AttributeKeyDataTypeInt64, querytypes.AttributeKeyDataTypeString, querytypes.AttributeKeyDataTypeBool)
	}

	for _, t := range types {
		for _, d := range dTypes {
			names = append(names, field.Key+"##"+t.String()+"##"+d.String())
		}
	}

	return names
}
