package timeseriestypes

import (
	"encoding/json"
	"sort"
	"strconv"
	"time"
)

type Series struct {
	Labels      map[string]string   `json:"labels"`
	LabelsArray []map[string]string `json:"labelsArray"`
	Points      []Point             `json:"values"`
}

func (s *Series) SortPoints() {
	sort.Slice(s.Points, func(i, j int) bool {
		return s.Points[i].Timestamp < s.Points[j].Timestamp
	})
}

func (s *Series) RemoveDuplicatePoints() {
	if len(s.Points) == 0 {
		return
	}

	// Prioritize the last point when the same point is sent twice because it is
	// the most recent point adjusted for the flux interval.
	newPoints := make([]Point, 0)
	for i := len(s.Points) - 1; i >= 0; i-- {
		if len(newPoints) == 0 {
			newPoints = append(newPoints, s.Points[i])
			continue
		}
		if newPoints[len(newPoints)-1].Timestamp != s.Points[i].Timestamp {
			newPoints = append(newPoints, s.Points[i])
		}
	}

	for i := len(newPoints)/2 - 1; i >= 0; i-- {
		opposite := len(newPoints) - 1 - i
		newPoints[i], newPoints[opposite] = newPoints[opposite], newPoints[i]
	}

	s.Points = newPoints
}

type Row struct {
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

type Point struct {
	Timestamp int64
	Value     float64
	// BoolValue is present for typed boolean query results. It deliberately
	// remains separate from Value so alert conditions cannot mistake false for
	// a numeric zero or true for a numeric one.
	BoolValue *bool
}

// MarshalJSON implements json.Marshaler.
func (p *Point) MarshalJSON() ([]byte, error) {
	v := strconv.FormatFloat(p.Value, 'f', -1, 64)
	return json.Marshal(map[string]interface{}{"timestamp": p.Timestamp, "value": v})
}

// UnmarshalJSON implements json.Unmarshaler.
func (p *Point) UnmarshalJSON(data []byte) error {
	var v struct {
		Timestamp int64  `json:"timestamp"`
		Value     string `json:"value"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	p.Timestamp = v.Timestamp
	var err error
	p.Value, err = strconv.ParseFloat(v.Value, 64)
	return err
}
