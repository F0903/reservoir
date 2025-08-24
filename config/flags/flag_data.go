package flags

import (
	"fmt"
	"log/slog"
	"reservoir/utils/bytesize"
	"strconv"
)

type FlagValue struct {
	raw string
}

func (v *FlagValue) AsString() string {
	return v.raw
}

func (v *FlagValue) AsInt() int {
	if intValue, err := strconv.Atoi(v.raw); err == nil {
		return intValue
	} else {
		slog.Error("Failed to parse int flag", "value", v.raw, "error", err)
		panic(fmt.Sprintf("Failed to parse int flag: %v", err))
	}
}

func (v *FlagValue) AsBool() bool {
	if boolValue, err := strconv.ParseBool(v.raw); err == nil {
		return boolValue
	} else {
		slog.Error("Failed to parse bool flag", "value", v.raw, "error", err)
		panic(fmt.Sprintf("Failed to parse bool flag: %v", err))
	}
}

func (v *FlagValue) AsBytesize() bytesize.ByteSize {
	if byteSizeValue, err := bytesize.Parse(v.raw); err == nil {
		return byteSizeValue
	} else {
		slog.Error("Failed to parse bytesize flag", "value", v.raw, "error", err)
		panic(fmt.Sprintf("Failed to parse bytesize flag: %v", err))
	}
}

type flagInfo struct {
	name  string
	onSet func(value FlagValue)
}

func (f *flagInfo) OnSet(onSet func(value FlagValue)) {
	f.onSet = onSet
}
