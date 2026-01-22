package crypto

import (
	"fmt"

	"github.com/zeebo/xxh3"
)

// Xxh3Bytes computes the XXH3 hash of the input byte slice and returns it as a hexadecimal string.
// It returns an error if the hashing process fails.
//
// # Parameters:
//   - bytes: the byte slice to be hashed
//
// # Returns:
//   - A hexadecimal string representation of the XXH3 hash
//   - An error if the write operation fails, wrapped with context
//
// # Example:
//
//	hash, err := Xxh3Bytes([]byte("hello world"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(hash) // Output: d447b1ea40e6988b
func Xxh3Bytes(bytes []byte) (string, error) {
	h := xxh3.New()
	_, err := h.Write(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to write bytes to hasher: %w", err)
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// Xxh3String computes the XXH3 hash of the input string and returns it as a hexadecimal string.
// It returns an error if the hashing process fails.
//
// # Parameters:
//   - str: the string to be hashed
//
// # Returns:
//   - A hexadecimal string representation of the XXH3 hash
//   - An error if the write operation fails, wrapped with context
//
// # Example:
//
//	hash, err := Xxh3String("hello world")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(hash) // Output: d447b1ea40e6988b
func Xxh3String(str string) (string, error) {
	return Xxh3Bytes([]byte(str))
}
