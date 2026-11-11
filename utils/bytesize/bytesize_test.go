package bytesize

import "testing"

func TestStringKeepsEveryByte(t *testing.T) {
	// String is what gets saved to the config file, so it must parse back to the same size.
	for input, want := range map[string]string{
		"1500M": "1500M",
		"2560K": "2560K",
		"1024M": "1G",
		"10G":   "10G",
		"1023B": "1023B",
		"0B":    "0B",
	} {
		got := ParseUnchecked(input).String()
		if got != want {
			t.Errorf("%s was written back as %s, want %s", input, got, want)
		}
		if ParseUnchecked(got) != ParseUnchecked(input) {
			t.Errorf("%s changed size when written back as %s", input, got)
		}
	}
}
