package headers

import (
	"errors"
	"testing"
)

//TODO: Expand to more than Range testing

// errPanic is used to signal that ParseRange panicked so tests don't crash the run.
var errPanic = errors.New("panic in ParseRange")

// safeParse wraps ParseRange and converts a panic into an error so the suite continues.
func safeParse(rangeStr string, size int64) (start, end int64, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errPanic
		}
	}()
	rh, err := parseRangeHeader(rangeStr)
	if err != nil {
		return 0, 0, err
	}
	return rh.SliceSize(size)
}

func TestParseRange(t *testing.T) {
	const size int64 = 1000

	tests := []struct {
		name     string
		rangeStr string
		size     int64
		start    int64
		end      int64
		wantErr  error
	}{
		// Valid cases
		{name: "single-byte", rangeStr: "bytes=0-0", size: size, start: 0, end: 0, wantErr: nil},
		{name: "simple", rangeStr: "bytes=0-499", size: size, start: 0, end: 499, wantErr: nil},
		{name: "open-ended", rangeStr: "bytes=500-", size: size, start: 500, end: size - 1, wantErr: nil},
		{name: "suffix", rangeStr: "bytes=-500", size: size, start: size - 500, end: size - 1, wantErr: nil},
		{name: "last-byte", rangeStr: "bytes=999-999", size: size, start: 999, end: 999, wantErr: nil},
		{name: "spaced-numbers", rangeStr: "bytes= 999 - 999 ", size: size, start: 999, end: 999, wantErr: nil},

		// Invalid unit/format/value
		{name: "bad-unit", rangeStr: "items=0-10", size: size, wantErr: ErrInvalidRangeUnit},
		{name: "no-equals", rangeStr: "bytes0-10", size: size, wantErr: ErrInvalidRangeFormat},
		{name: "non-numeric", rangeStr: "bytes=a-b", size: size, wantErr: ErrInvalidRangeValue},
		{name: "missing-end-digit", rangeStr: "bytes=10-x", size: size, wantErr: ErrInvalidRangeValue},
		{name: "neg-start-format", rangeStr: "bytes=-1-10", size: size, wantErr: ErrInvalidRangeFormat},

		// Multiple ranges not supported
		{name: "multiple", rangeStr: "bytes=0-10,20-30", size: size, wantErr: ErrMultipleRangesNotSupported},

		// Out of bounds
		{name: "start-gt-end", rangeStr: "bytes=10-5", size: size, wantErr: ErrRangeValueOutOfBounds},
		{name: "start-eq-size", rangeStr: "bytes=1000-1000", size: size, wantErr: ErrRangeValueOutOfBounds},
		{name: "end-eq-size", rangeStr: "bytes=0-1000", size: size, wantErr: ErrRangeValueOutOfBounds},
		{name: "suffix-too-large", rangeStr: "bytes=-2000", size: size, wantErr: ErrRangeValueOutOfBounds},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStart, gotEnd, err := safeParse(tt.rangeStr, tt.size)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v (start=%d end=%d)", tt.wantErr, err, gotStart, gotEnd)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotStart != tt.start || gotEnd != tt.end {
				t.Fatalf("got (%d,%d), want (%d,%d)", gotStart, gotEnd, tt.start, tt.end)
			}
		})
	}
}
