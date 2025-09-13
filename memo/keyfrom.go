package memo

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
)

// KeyFrom generates a SHA-256 hash key from the provided parts. The function returns the hex-encoded string
// representation of the hash.
//
// This function is useful for creating consistent cache keys from multiple values of any type that gob can encode.
//
// # Parameters:
//   - parts: Variable number of arguments of any type to be hashed together
//
// # Returns:
//   - A hex-encoded SHA-256 hash string
//
// Note: If gob encoding fails for any part, it is silently ignored (best-effort basis).
func KeyFrom(parts ...any) string {
	h := sha256.New()
	enc := gob.NewEncoder(h)
	for _, p := range parts {
		_ = enc.Encode(p) // best-effort
	}

	return hex.EncodeToString(h.Sum(nil))
}
