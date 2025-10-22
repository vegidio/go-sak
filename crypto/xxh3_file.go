package crypto

import (
	"fmt"
	"io"
	"os"

	"github.com/zeebo/xxh3"
)

// Xxh3File computes the XXH3 hash of the file at the given path.
//
// # Parameters:
//   - filePath: the path to the file to hash
//
// # Returns:
//   - string: the XXH3 hash as a lowercase hexadecimal string
//   - error: any error that occurred during file operations or hashing
//
// # Example:
//
//	hash, err := Xxh3File("/path/to/file.txt")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("XXH3: %s\n", hash)
func Xxh3File(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := xxh3.New()

	// Copy the file content to the hash
	if _, err = io.Copy(hash, file); err != nil {
		return "", err
	}

	// Calculate the final hash and return as hex string
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
