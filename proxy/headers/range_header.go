package headers

import (
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
)

var (
	ErrInvalidRangeUnit           = errors.New("invalid range unit")
	ErrInvalidRangeValue          = errors.New("invalid range value")
	ErrInvalidRangeFormat         = errors.New("invalid range format")
	ErrMultipleRangesNotSupported = errors.New("multiple ranges not supported")
	ErrRangeValueOutOfBounds      = errors.New("range value out of bounds")
)

type rangeHeader struct {
	start int64 // -1 indicates no start
	end   int64 // -1 indicates no end
}

func parseRangeNumber(numStr string) (num int64, endIndex int64, ok bool) {
	if numStr == "" {
		return 0, 0, false
	}

	if numStr[0] == '-' {
		// Negative numbers are not allowed
		return 0, 0, false
	}

	var index int64 = 0
	for i, ch := range numStr {
		if ch == ' ' || ch == '\t' {
			index++
			continue
		}

		if ch < '0' || ch > '9' {
			if i == 0 {
				return 0, 0, false
			}
			return num, index, true
		}

		num = num*10 + int64(ch-'0')
		index++
	}

	return num, index, true
}

func validateRange(start, end, dataSize int64) error {
	if start < 0 || end < 0 || start >= dataSize || end >= dataSize || start > end {
		return ErrRangeValueOutOfBounds
	}
	return nil
}

func parseRangeHeader(rangeStr string) (rangeHeader, error) {
	slog.Debug("Parsing Range header", "raw", rangeStr)

	splitStr := strings.SplitN(rangeStr, "=", 2)
	if len(splitStr) != 2 {
		return rangeHeader{}, ErrInvalidRangeFormat
	}

	unit := splitStr[0]
	valuesStr := splitStr[1]

	if unit != "bytes" {
		return rangeHeader{}, ErrInvalidRangeUnit
	}

	firstCh := valuesStr[0]
	if firstCh == '-' {
		// Suffix range: last N bytes
		suffixLength, suffixTail, ok := parseRangeNumber(valuesStr[1:])
		if !ok {
			return rangeHeader{}, ErrInvalidRangeValue
		}
		suffixTail += 1 // To adjust for the firstCh offset

		isTailSmaller := suffixTail < int64(len(valuesStr))
		if isTailSmaller && valuesStr[suffixTail] == ',' {
			return rangeHeader{}, ErrMultipleRangesNotSupported
		} else if isTailSmaller && valuesStr[suffixTail] == '-' {
			// Invalid format: -N-...
			return rangeHeader{}, ErrInvalidRangeFormat
		}

		return rangeHeader{start: -1, end: suffixLength}, nil // Indicate suffix range
	}

	start, startTail, ok := parseRangeNumber(valuesStr)
	if !ok {
		return rangeHeader{}, ErrInvalidRangeValue
	}

	middleCh := valuesStr[startTail]
	if middleCh != '-' {
		return rangeHeader{}, ErrInvalidRangeFormat
	}

	if startTail+1 >= int64(len(valuesStr)) {
		// Unbounded range: start-
		return rangeHeader{start: start, end: -1}, nil
	}

	end, endTail, ok := parseRangeNumber(valuesStr[startTail+1:])
	if !ok {
		return rangeHeader{}, ErrInvalidRangeValue
	}
	endTail += startTail + 1 // To adjust for the middleCh offset

	if endTail < int64(len(valuesStr)) && valuesStr[endTail] == ',' {
		return rangeHeader{}, ErrMultipleRangesNotSupported
	}

	slog.Debug("Parsed Range header", "start", start, "end", end)
	return rangeHeader{start: start, end: end}, nil
}

func (r rangeHeader) SliceSize(dataSize int64) (start int64, end int64, err error) {
	if r.end == -1 && r.start == -1 {
		// This should be caught in ParseRange and not be able to happen.
		return 0, 0, ErrInvalidRangeFormat
	} else if r.start == -1 {
		// Suffix range
		start = dataSize - r.end
		end = dataSize - 1
	} else if r.start != -1 && r.end != -1 {
		start = r.start
		end = r.end
	} else if r.end == -1 {
		start = r.start
		end = dataSize - 1
	}

	return start, end, validateRange(start, end, dataSize)
}

func (r rangeHeader) String() string {
	if r.start == -1 {
		return "bytes=-" + strconv.FormatInt(r.end, 10)
	}
	return fmt.Sprintf("bytes=%d-%d", r.start, r.end)
}
