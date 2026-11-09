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

func TestSliceSizeTrimsRangesToTheFile(t *testing.T) {
	const size = 36

	tests := []struct {
		name      string
		r         rangeHeader
		wantStart int64
		wantEnd   int64
		wantErr   bool
	}{
		{name: "inside the file", r: rangeHeader{start: 2, end: 5}, wantStart: 2, wantEnd: 5},
		{name: "end past the file", r: rangeHeader{start: 30, end: 100}, wantStart: 30, wantEnd: 35},
		{name: "open ended", r: rangeHeader{start: 30, end: -1}, wantStart: 30, wantEnd: 35},
		{name: "suffix longer than the file", r: rangeHeader{start: -1, end: 100}, wantStart: 0, wantEnd: 35},
		{name: "suffix inside the file", r: rangeHeader{start: -1, end: 6}, wantStart: 30, wantEnd: 35},
		{name: "start past the file", r: rangeHeader{start: 36, end: 40}, wantErr: true},
		{name: "empty suffix", r: rangeHeader{start: -1, end: 0}, wantErr: true},
		{name: "reversed", r: rangeHeader{start: 5, end: 2}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, err := tt.r.SliceSize(size)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %d-%d", start, end)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if start != tt.wantStart || end != tt.wantEnd {
				t.Fatalf("got %d-%d, want %d-%d", start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}
