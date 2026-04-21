package memo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// KeyFrom generates a SHA-256 hash key from the provided parts. The function returns the hex-encoded string
// representation of the hash.
//
// This function is useful for creating consistent cache keys from multiple values of any type that can be JSON-encoded.
// Each part is marshalled independently and separated by a NUL byte so that parameter order is preserved. Maps with
// string keys are encoded in sorted key order, making the output fully deterministic across calls.
//
// # Parameters:
//   - parts: Variable number of arguments of any type to be hashed together
//
// # Returns:
//   - A hex-encoded SHA-256 hash string
//
// Note: If JSON encoding fails for any part, it is silently ignored (best-effort basis).
func KeyFrom(parts ...any) string {
	h := sha256.New()
	for _, p := range parts {
		b, err := json.Marshal(p)
		if err != nil {
			continue // best-effort
		}
		h.Write(b)
		h.Write([]byte{0}) // separator to preserve parameter-order sensitivity
	}

	return hex.EncodeToString(h.Sum(nil))
}
