package crypto

import (
	"fmt"
	"io"
	"os"

	"github.com/zeebo/xxh3"
)

// Xxh3Reader computes the XXH3 hash of a reader.
//
// # Parameters:
//   - reader: the reader to hash
//
// # Returns:
//   - string: the XXH3 hash as a lowercase hexadecimal string
//   - error: any error that occurred during hashing
//
// # Example:
//
//	hash, err := Xxh3Reader(fileReader)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("XXH3: %s\n", hash)
func Xxh3Reader(reader io.Reader) (string, error) {
	hash := xxh3.New()

	// Copy the reader content to the hash
	if _, err := io.Copy(hash, reader); err != nil {
		return "", err
	}

	// Calculate the final hash and return as hex string
	return fmt.Sprintf("%x", hash.Sum128().Bytes()), nil
}

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

	return Xxh3Reader(file)
}
