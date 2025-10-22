package crypto

import (
	"crypto/sha256"
	"fmt"
)

// Sha256Bytes computes the SHA-256 hash of the input byte slice and returns it as a hexadecimal string.
// It returns an error if the hashing process fails.
//
// # Parameters:
//   - bytes: the byte slice to be hashed
//
// # Returns:
//   - A hexadecimal string representation of the SHA-256 hash
//   - An error if the write operation fails, wrapped with context
//
// # Example:
//
//	hash, err := Sha256Bytes([]byte("hello world"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(hash) // Output: b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9
func Sha256Bytes(bytes []byte) (string, error) {
	h := sha256.New()
	_, err := h.Write(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to write bytes to hasher: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
