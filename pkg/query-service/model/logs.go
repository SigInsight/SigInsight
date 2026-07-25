package model

type QueryProgress struct {
	ReadRows uint64 `json:"read_rows"`

	ReadBytes uint64 `json:"read_bytes"`

	ElapsedMs uint64 `json:"elapsed_ms"`
}
