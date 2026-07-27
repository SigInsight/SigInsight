package common

import (
	"math"
	"regexp"
	"time"
	"unicode"

	"github.com/SigNoz/signoz/pkg/query-service/constants"
)

func PastDayRoundOff() int64 {
	now := time.Now().UnixMilli()
	return int64(math.Floor(float64(now)/float64(time.Hour.Milliseconds()*24))) * time.Hour.Milliseconds() * 24
}

// start and end are in milliseconds
func MinAllowedStepInterval(start, end int64) int64 {
	step := (end - start) / constants.MaxAllowedPointsInTimeSeries / 1000
	if step < 60 {
		return 60
	}
	// return the nearest lower multiple of 60
	return step - step%60
}

func NormalizeLabelName(name string) string {
	// See https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels

	// Regular expression to match non-alphanumeric characters except underscores
	reg := regexp.MustCompile(`[^a-zA-Z0-9_]`)

	// Replace all non-alphanumeric characters except underscores with underscores
	normalized := reg.ReplaceAllString(name, "_")

	// If the first character is not a letter or an underscore, prepend an underscore
	if len(normalized) > 0 && !unicode.IsLetter(rune(normalized[0])) && normalized[0] != '_' {
		normalized = "_" + normalized
	}

	return normalized
}
