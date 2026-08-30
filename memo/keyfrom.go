package memo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
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
// Note: A part that cannot be JSON-encoded contributes a marker derived from its type rather than being dropped, so
// it still separates one key from another.
func KeyFrom(parts ...any) string {
	h := sha256.New()
	for _, p := range parts {
		b, err := json.Marshal(p)
		if err != nil {
			// Skipping the part outright - with its separator - would make KeyFrom("user", make(chan int))
			// equal to KeyFrom("user"), so two different computations would share one cache entry. Fold in a
			// type-tagged marker instead, so the part still contributes to the key.
			b = fmt.Appendf(nil, "!unmarshalable:%T", p)
		}

		h.Write(b)
		h.Write([]byte{0}) // separator to preserve parameter-order sensitivity
	}

	return hex.EncodeToString(h.Sum(nil))
}
