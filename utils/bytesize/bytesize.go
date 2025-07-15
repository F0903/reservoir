package bytesize

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
)

type ByteSize int64

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
	num := int64(0)
	multiplier := int64(1)
	foundUnit := false
	for _, r := range s {
		if isDigit(r) {
			if foundUnit {
				return 0, errors.New("characters after the unit are not allowed")
			}

			digit := int64(r - '0')
			num = num*10 + digit
		} else {
			if foundUnit {
				return 0, fmt.Errorf("multiple units found in string: %s", s)
			}

			unit, exists := unitRuneMap[r]
			if !exists {
				return 0, fmt.Errorf("unknown unit: %c", r)
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
	return int64(b)
}

func ParseUnchecked(s string) ByteSize {
	result, err := Parse(s)
	if err != nil {
		log.Fatalf("failed to parse byte size: %v", err)
	}
	return result
}

func (b ByteSize) ToString(unitRune rune) string {
	unit, exists := unitRuneMap[unitRune]
	if !exists {
		return fmt.Sprintf("unknown unit: %c", unitRune)
	}
	size := int64(b) / unit
	return fmt.Sprintf("%d%c", size, unitRune)
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
	return b.ToString(unitRune)
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
