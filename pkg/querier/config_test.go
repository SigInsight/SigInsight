package querier

import "testing"

func TestDefaultConfigKeepsTraceDetailFluxInterval(t *testing.T) {
	config, ok := newConfig().(Config)
	if !ok {
		t.Fatalf("newConfig() type = %T, want Config", newConfig())
	}
	if config.FluxInterval <= 0 {
		t.Fatal("default config must keep a positive trace-detail flux interval")
	}
}
