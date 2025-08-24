package bytesize

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
)

type ByteSize int64

var (
	ErrCharsAfterUnit = errors.New("characters after the unit are not allowed")
	ErrMultipleUnits  = errors.New("multiple units found in string")
	ErrUnknownUnit    = errors.New("unknown unit")
	ErrEmptyString    = errors.New("empty string")
	ErrInvalidFormat  = errors.New("invalid format")
)

const (
	UnitB int64 = 1
	UnitK int64 = 1024
	UnitM int64 = 1024 * 1024
	UnitG int64 = 1024 * 1024 * 1024
	UnitT int64 = 1024 * 1024 * 1024 * 1024
)

var unitRuneMap = map[rune]int64{
	'B': UnitB,
	'K': UnitK,
	'M': UnitM,
	'G': UnitG,
	'T': UnitT,
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

func Parse(s string) (ByteSize, error) {
	if s == "" {
		return 0, ErrEmptyString
	}

	num := int64(0)
	multiplier := int64(1)
	foundUnit := false

	for _, r := range s {
		if isDigit(r) {
			if foundUnit {
				return 0, fmt.Errorf("%w in: %s", ErrCharsAfterUnit, s)
			}

			digit := int64(r - '0')
			num = num*10 + digit
		} else {
			if foundUnit {
				return 0, fmt.Errorf("%w in: %s", ErrMultipleUnits, s)
			}

			unit, exists := unitRuneMap[r]
			if !exists {
				return 0, fmt.Errorf("%w: %c in: %s", ErrUnknownUnit, r, s)
			}

			multiplier = unit
			foundUnit = true
			break
		}
	}

	return ByteSize(num * multiplier), nil
}

func (b ByteSize) Convert(unit int64) int64 {
	return int64(b) / unit
}

func (b ByteSize) Bytes() int64 {
	return b.Convert(UnitB)
}

func (b ByteSize) KiloBytes() int64 {
	return b.Convert(UnitK)
}

func (b ByteSize) MegaBytes() int64 {
	return b.Convert(UnitM)
}

func (b ByteSize) GigaBytes() int64 {
	return b.Convert(UnitG)
}

func (b ByteSize) TeraBytes() int64 {
	return b.Convert(UnitT)
}

func ParseUnchecked(s string) ByteSize {
	result, err := Parse(s)
	if err != nil {
		slog.Error("Failed to parse byte size", "input", s, "error", err)
		panic(err)
	}
	return result
}

func (b ByteSize) ToString(unitRune rune) (string, error) {
	unit, exists := unitRuneMap[unitRune]
	if !exists {
		return "", fmt.Errorf("%w: %c", ErrUnknownUnit, unitRune)
	}
	size := int64(b) / unit
	return fmt.Sprintf("%d%c", size, unitRune), nil
}

func (b ByteSize) FindLargestFittingUnit() rune {
	largestUnitSize := int64(1)
	largestUnitRune := 'B'

	for unitRune, unitSize := range unitRuneMap {
		if int64(b) < unitSize {
			continue
		}

		if unitSize < largestUnitSize {
			continue
		}

		largestUnitRune = unitRune
		largestUnitSize = unitSize
	}

	return largestUnitRune
}

func (b ByteSize) String() string {
	unitRune := b.FindLargestFittingUnit()
	result, _ := b.ToString(unitRune)
	return result
}

func (b ByteSize) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.String())
}

func (b *ByteSize) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	parsed, err := Parse(s)
	if err != nil {
		return err
	}
	*b = parsed

	return nil
}
