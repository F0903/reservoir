package phc

import (
	"encoding/base64"
	"fmt"
	"strings"
	"testing"
)

func TestPHC_RoundTripAndVerify(t *testing.T) {
	password := "p@ssw0rd!"
	ph := GenerateArgon2id(password)

	s := ph.String()
	if !strings.HasPrefix(s, "$argon2id$") {
		t.Fatalf("unexpected PHC prefix: %s", s)
	}

	parsed, err := ParsePHC(s)
	if err != nil {
		t.Fatalf("ParsePHC error: %v\ninput: %s", err, s)
	}

	if !parsed.VerifyArgon2id(password) {
		t.Fatalf("VerifyArgon2id failed for correct password")
	}
	if parsed.VerifyArgon2id("wrong-password") {
		t.Fatalf("VerifyArgon2id unexpectedly succeeded for wrong password")
	}

	// Ensure deterministic String() round-trip
	if got := parsed.String(); got != s {
		t.Fatalf("String() round-trip mismatch\nwant: %s\n got: %s", s, got)
	}

	// Basic field sanity
	if parsed.version != ph.version || parsed.id != ph.id {
		t.Fatalf("parsed id/version mismatch")
	}
	if parsed.memory == 0 || parsed.time == 0 || parsed.threads == 0 {
		t.Fatalf("parsed parameters should be non-zero: m=%d t=%d p=%d", parsed.memory, parsed.time, parsed.threads)
	}
	if len(parsed.hash) == 0 || parsed.keyLen == 0 || parsed.keyLen != uint32(len(parsed.hash)) {
		t.Fatalf("hash/keyLen mismatch or empty: keyLen=%d hashLen=%d", parsed.keyLen, len(parsed.hash))
	}
	if len(parsed.salt) != 16 {
		t.Fatalf("salt length mismatch: got %d", len(parsed.salt))
	}
}

func TestParsePHC_InvalidFormat(t *testing.T) {
	valid := GenerateArgon2id("secret").String()

	withPart := func(index int, value func(string) string) string {
		t.Helper()

		parts := strings.Split(strings.TrimPrefix(valid, "$"), "$")
		if len(parts) != 5 {
			t.Fatalf("unexpected parts count: %d", len(parts))
		}
		parts[index] = value(parts[index])
		return "$" + strings.Join(parts, "$")
	}

	shortSalt := make([]byte, 8)
	for i := range shortSalt {
		shortSalt[i] = byte(i)
	}

	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{name: "empty input", input: "", wantErr: true},
		{name: "valid without leading dollar", input: strings.TrimPrefix(valid, "$")},
		{name: "missing hash", input: "$argon2id$v=19$m=65536,t=1,p=4,l=32$onlysalt", wantErr: true},
		{name: "invalid version", input: "$argon2id$v=x$m=65536,t=1,p=4,l=32$abc$def", wantErr: true},
		{name: "unsupported id", input: strings.Replace(valid, "$argon2id$", "$scrypt$", 1), wantErr: true},
		{
			name: "short salt",
			input: withPart(3, func(string) string {
				return base64.RawStdEncoding.EncodeToString(shortSalt)
			}),
			wantErr: true,
		},
		{
			name: "hash length mismatch",
			input: withPart(2, func(params string) string {
				return replaceLValue(params, 999) // Guaranteed to differ from the actual hash length.
			}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParsePHC(tt.input); (err != nil) != tt.wantErr {
				t.Fatalf("ParsePHC(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
		})
	}
}

// replaceLValue replaces the l=... parameter in a comma-separated params string.
func replaceLValue(params string, newVal int) string {
	segs := strings.Split(params, ",")
	for i, seg := range segs {
		if strings.HasPrefix(seg, "l=") {
			segs[i] = fmt.Sprintf("l=%d", newVal)
			return strings.Join(segs, ",")
		}
	}
	// If l= not present, append it
	return params + fmt.Sprintf(",l=%d", newVal)
}
