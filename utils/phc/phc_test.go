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
	// empty input should error
	if _, err := ParsePHC(""); err == nil {
		t.Fatalf("expected error for empty input")
	}

	// Generate a valid PHC and remove the leading '$' — should still parse
	s := GenerateArgon2id("secret").String()
	sNoPrefix := strings.TrimPrefix(s, "$")
	if _, err := ParsePHC(sNoPrefix); err != nil {
		t.Fatalf("unexpected error for valid input without leading $: %v", err)
	}

	// Missing hash should error
	if _, err := ParsePHC("$argon2id$v=19$m=65536,t=1,p=4,l=32$onlysalt"); err == nil {
		t.Fatalf("expected error for missing hash part")
	}

	// Invalid version should error
	if _, err := ParsePHC("$argon2id$v=x$m=65536,t=1,p=4,l=32$abc$def"); err == nil {
		t.Fatalf("expected error for invalid version")
	}
}

func TestParsePHC_WrongID(t *testing.T) {
	ph := GenerateArgon2id("secret")
	s := ph.String()
	bad := strings.Replace(s, "$argon2id$", "$scrypt$", 1)
	if _, err := ParsePHC(bad); err == nil {
		t.Fatalf("expected error for unsupported id")
	}
}

func TestParsePHC_SaltLength(t *testing.T) {
	ph := GenerateArgon2id("secret")
	s := ph.String()

	parts := strings.Split(strings.TrimPrefix(s, "$"), "$")
	if len(parts) != 5 {
		t.Fatalf("unexpected parts count: %d", len(parts))
	}

	// Replace salt with 8 bytes
	shortSalt := make([]byte, 8)
	for i := range shortSalt {
		shortSalt[i] = byte(i)
	}
	parts[3] = base64.RawStdEncoding.EncodeToString(shortSalt)
	bad := "$" + strings.Join(parts, "$")

	if _, err := ParsePHC(bad); err == nil {
		t.Fatalf("expected error for short salt")
	}
}

func TestParsePHC_HashLengthMismatch(t *testing.T) {
	ph := GenerateArgon2id("secret")
	s := ph.String()

	parts := strings.Split(strings.TrimPrefix(s, "$"), "$")
	if len(parts) != 5 {
		t.Fatalf("unexpected parts count: %d", len(parts))
	}

	// Inflate l= to a mismatching value
	// parts[2] looks like: m=...,t=...,p=...,l=...
	// find existing l value and replace with a wrong one
	params := parts[2]
	// Set to a value guaranteed to differ from actual hash length
	params = replaceLValue(params, 999)
	parts[2] = params
	bad := "$" + strings.Join(parts, "$")

	if _, err := ParsePHC(bad); err == nil {
		t.Fatalf("expected error for hash length mismatch")
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
