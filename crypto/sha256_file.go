package crypto

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// Sha256Reader computes the SHA-256 hash of a reader.
//
// # Parameters:
//   - reader: the reader to hash
//
// # Returns:
//   - string: the SHA-256 hash as a lowercase hexadecimal string
//   - error: any error that occurred during hashing
//
// # Example:
//
//	hash, err := Sha256Reader(fileReader)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("SHA-256: %s\n", hash)
func Sha256Reader(reader io.Reader) (string, error) {
	hash := sha256.New()

	// Copy the reader content to the hash
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}

	// Calculate the final hash and return as hex string
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

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

	return Sha256Reader(file)
}
