package crypto

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// Sha256File computes the SHA-256 hash of the file at the given path.
//
// # Parameters:
//   - filePath: the path to the file to hash
//
// # Returns:
//   - string: the SHA-256 hash as a lowercase hexadecimal string
//   - error: any error that occurred during file operations or hashing
//
// # Example:
//
//	hash, err := Sha256File("/path/to/file.txt")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("SHA-256: %s\n", hash)
func Sha256File(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()

	// Copy the file content to the hash
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}

	// Calculate the final hash and return as hex string
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
