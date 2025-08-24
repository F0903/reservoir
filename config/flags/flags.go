package flags

import (
	"flag"
)

type Flags struct {
	flags map[string]*flagInfo // name -> *flagInfo
}

func New() Flags {
	return Flags{
		flags: make(map[string]*flagInfo),
	}
}

func (fl *Flags) Parse() {
	flag.Parse()
	flag.Visit(func(setFlag *flag.Flag) {
		defFlag := fl.flags[setFlag.Name]
		value := setFlag.Value.String()
		defFlag.onSet(FlagValue{raw: value})
	})
}

func (fl *Flags) AddString(name string, value string, usage string) *flagInfo {
	flag.String(name, value, usage)
	f := &flagInfo{
		name: name,
	}
	fl.flags[name] = f
	return f
}

func (fl *Flags) AddInt(name string, value int, usage string) *flagInfo {
	flag.Int(name, value, usage)
	f := &flagInfo{
		name: name,
	}
	fl.flags[name] = f
	return f
}

func (fl *Flags) AddBool(name string, value bool, usage string) *flagInfo {
	flag.Bool(name, value, usage)
	f := &flagInfo{
		name: name,
	}
	fl.flags[name] = f
	return f
}
