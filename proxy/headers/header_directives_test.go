package headers

import (
	"net/http"
	"testing"
)

func TestShouldCacheHonoursDirectives(t *testing.T) {
	tests := []struct {
		name         string
		cacheControl string
		ignore       bool
		want         bool
	}{
		{"public with no max-age", "public", false, true},
		{"max-age zero", "max-age=0", false, false},
		{"positive max-age", "max-age=60", false, true},
		{"no-store when strict", "no-store", false, false},
		{"no-store when ignoring", "no-store", true, true},
		{"private when strict", "private", false, false},
		{"private when ignoring", "private, max-age=60", true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hd := ParseHeaderDirective(http.Header{"Cache-Control": []string{tt.cacheControl}})
			if got := hd.ShouldCache(tt.ignore); got != tt.want {
				t.Fatalf("ShouldCache(%v) = %v, want %v", tt.ignore, got, tt.want)
			}
		})
	}
}
