package version

import (
	"strings"
	"testing"
)

func TestBannerUsesSigInsightBrandWithoutLegacyArtwork(t *testing.T) {
	banner := (Build{version: "2.1.0", variant: "community"}).banner(2026)

	for _, expected := range []string{
		"SigInsight\n",
		"Version: 2.1.0 (community)\n",
		"Copyright 2026 SigInsight. All rights reserved.\n",
	} {
		if !strings.Contains(banner, expected) {
			t.Fatalf("banner = %q, want %q", banner, expected)
		}
	}
	if strings.Contains(banner, "SigNoz") || strings.Contains(banner, "********") {
		t.Fatalf("banner contains legacy branding or artwork: %q", banner)
	}
}
