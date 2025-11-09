package atomics

import (
	"encoding/json"
	"sync/atomic"
)

// Uint64 is a wrapper around the Go 1.19+ atomic.Uint64 type,
// providing convenience methods and JSON serialization.
// The struct holds a pointer to the atomic value, making the wrapper itself copy-safe.
type Uint64 struct {
	val *atomic.Uint64
}

// NewUint64 creates a new Uint64 with an initial value.
func NewUint64(initialValue uint64) Uint64 {
	val := &atomic.Uint64{}
	val.Store(initialValue)
	return Uint64{val: val}
}

// Add atomically adds delta to the value.
func (u *Uint64) Add(delta uint64) {
	u.val.Add(delta)
}

// Sub atomically subtracts delta from the value.
func (u *Uint64) Sub(delta uint64) {
	u.val.Add(^uint64(delta - 1))
}

// Increment atomically increments the value by 1.
func (u *Uint64) Increment() {
	u.Add(1)
}

// Decrement atomically decrements the value by 1.
func (u *Uint64) Decrement() {
	u.Sub(1)
}

// Set atomically sets the value.
func (u *Uint64) Set(value uint64) {
	u.val.Store(value)
}

// Get atomically retrieves the value.
func (u *Uint64) Get() uint64 {
	return u.val.Load()
}

// MarshalJSON implements the json.Marshaler interface with a value receiver.
// This is safe because the struct itself is copy-safe (it only contains a pointer).
func (u Uint64) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.Get())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (u *Uint64) UnmarshalJSON(data []byte) error {
	var value uint64
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	u.Set(value)
	return nil
}
