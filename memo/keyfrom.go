package memo

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
)

func KeyFrom(parts ...any) string {
	h := sha256.New()
	enc := gob.NewEncoder(h)
	for _, p := range parts {
		_ = enc.Encode(p) // best-effort
	}

	return hex.EncodeToString(h.Sum(nil))
}
