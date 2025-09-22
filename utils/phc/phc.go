package phc

import (
	"crypto/rand"
	"crypto/subtle"
	"database/sql/driver"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const defaultArgonMem uint32 = 64 * 1024
const defaultArgonTime uint32 = 1
const defaultArgonThreads uint8 = 2
const defaultArgonKeyLen uint32 = 32

type PHC struct {
	id      string
	version int
	memory  uint32
	time    uint32
	threads uint8
	keyLen  uint32
	salt    [16]byte
	hash    []byte
}

func GenerateArgon2id(password string) *PHC {
	var salt [16]byte
	rand.Read(salt[:])

	time := defaultArgonTime
	memory := defaultArgonMem
	threads := defaultArgonThreads
	keyLen := defaultArgonKeyLen
	hash := argon2.IDKey([]byte(password), salt[:], time, memory, threads, keyLen)
	return &PHC{
		id:      "argon2id",
		version: argon2.Version,
		memory:  memory,
		time:    time,
		threads: threads,
		keyLen:  keyLen,
		salt:    salt,
		hash:    hash,
	}
}

func ParsePHC(s string) (*PHC, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("empty PHC string")
	}

	// Strip a single leading '$' if present to normalize splitting
	s = strings.TrimPrefix(s, "$")

	parts := strings.Split(s, "$")
	if len(parts) != 5 {
		return nil, fmt.Errorf("invalid PHC format: expected 5 parts, got %d", len(parts))
	}

	id := parts[0]
	if id != "argon2id" {
		return nil, fmt.Errorf("unsupported algorithm id: %s", id)
	}

	// version in the form v=19
	verPart := parts[1]
	if !strings.HasPrefix(verPart, "v=") {
		return nil, fmt.Errorf("invalid version part: %q", verPart)
	}
	verStr := strings.TrimPrefix(verPart, "v=")
	version, err := strconv.Atoi(verStr)
	if err != nil {
		return nil, fmt.Errorf("invalid version: %w", err)
	}

	// params in the form m=...,t=...,p=...,l=...
	paramsPart := parts[2]
	var (
		memory  uint32
		time    uint32
		threads uint8
		keyLen  uint32
	)
	if paramsPart == "" {
		return nil, fmt.Errorf("missing parameters section")
	}
	for seg := range strings.SplitSeq(paramsPart, ",") {
		if seg == "" {
			continue
		}
		kv := strings.SplitN(seg, "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid parameter: %q", seg)
		}
		k := kv[0]
		v := kv[1]
		switch k {
		case "m":
			n, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid m value: %w", err)
			}
			memory = uint32(n)
		case "t":
			n, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid t value: %w", err)
			}
			time = uint32(n)
		case "p":
			n, err := strconv.ParseUint(v, 10, 8)
			if err != nil {
				return nil, fmt.Errorf("invalid p value: %w", err)
			}
			threads = uint8(n)
		case "l":
			n, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("invalid l value: %w", err)
			}
			keyLen = uint32(n)
		default:
			// Ignore unknown parameters for forward compatibility
		}
	}
	if memory == 0 || time == 0 || threads == 0 {
		return nil, fmt.Errorf("missing required parameters m,t,p or zero values")
	}

	// Decode salt (expect 16 bytes to fit [16]byte)
	saltB64 := parts[3]
	var salt [16]byte
	n, err := base64.RawStdEncoding.Decode(salt[:], []byte(saltB64))
	if err != nil {
		return nil, fmt.Errorf("invalid salt: %w", err)
	}
	if n != 16 {
		return nil, fmt.Errorf("invalid salt length: got %d, want 16", n)
	}

	// Decode hash
	hashB64 := parts[4]
	hash, err := base64.RawStdEncoding.DecodeString(hashB64)
	if err != nil {
		return nil, fmt.Errorf("invalid hash: %w", err)
	}
	if len(hash) == 0 {
		return nil, fmt.Errorf("empty hash")
	}
	// If l (keyLen) provided, ensure it matches the decoded hash length
	if keyLen != 0 && keyLen != uint32(len(hash)) {
		return nil, fmt.Errorf("hash length mismatch: l=%d, actual=%d", keyLen, len(hash))
	}
	if keyLen == 0 {
		keyLen = uint32(len(hash))
	}

	return &PHC{
		id:      id,
		version: version,
		memory:  memory,
		time:    time,
		threads: threads,
		keyLen:  keyLen,
		salt:    salt,
		hash:    hash,
	}, nil
}

func (p *PHC) VerifyArgon2id(password string) bool {
	hash := argon2.IDKey([]byte(password), p.salt[:], p.time, p.memory, p.threads, p.keyLen)
	return subtle.ConstantTimeCompare(hash, p.hash) == 1
}

func (p *PHC) String() string {
	return fmt.Sprintf("$%s$v=%d$m=%d,t=%d,p=%d,l=%d$%s$%s",
		p.id,
		p.version,
		p.memory,
		p.time,
		p.threads,
		p.keyLen,
		base64.RawStdEncoding.EncodeToString(p.salt[:]),
		base64.RawStdEncoding.EncodeToString(p.hash),
	)
}

// Implements the sql.Scanner interface to read a PHC from a database value.
func (p *PHC) Scan(src any) error {
	switch v := src.(type) {
	case string:
		parsed, err := ParsePHC(v)
		if err != nil {
			return err
		}
		*p = *parsed
	case []byte:
		parsed, err := ParsePHC(string(v))
		if err != nil {
			return err
		}
		*p = *parsed
	default:
		return fmt.Errorf("cannot scan PHC from %T", src)
	}
	return nil
}

// Implements the driver.Valuer interface to write a PHC to a database value.
func (p PHC) Value() (driver.Value, error) {
	return p.String(), nil
}
