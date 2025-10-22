package crypto

import (
	"crypto/sha256"
	"fmt"
	"io"
)

// Sha256String computes the SHA-256 hash of the input string and returns it as a hexadecimal string.
// It returns an error if the hashing process fails.
//
// # Parameters:
//   - str: the string to be hashed
//
// # Returns:
//   - A hexadecimal string representation of the SHA-256 hash
//   - An error if the write operation fails, wrapped with context
//
// # Example:
//
//	hash, err := Sha256String("hello world")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(hash) // Output: b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9
func Sha256String(str string) (string, error) {
	h := sha256.New()
	_, err := io.WriteString(h, str)
	if err != nil {
		return "", fmt.Errorf("failed to write string to hasher: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
