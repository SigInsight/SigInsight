package querier

import "testing"

func TestDefaultConfigEnablesLightweightEngine(t *testing.T) {
	config, ok := newConfig().(Config)
	if !ok {
		t.Fatalf("newConfig() type = %T, want Config", newConfig())
	}
	if !config.EnableLightweightEngine {
		t.Fatal("default config must route supported V5 queries through Lite")
	}
}
