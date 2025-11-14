# go-sak

A Swiss Army Knife collection of Go utilities providing commonly used functions across various domains including async operations, cryptography, file system operations, HTTP fetching, GitHub integration, memoization, and time utilities.

## ⚙️ Installation

```bash
go get github.com/vegidio/go-sak
```

## 🧰 Packages

### async

Concurrent processing utilities for channels and slices.

#### `SliceToChannel[T, R](items []T, concurrency int, fn func(T) R) <-chan R`

Processes items from an input slice concurrently using the specified number of worker goroutines. Returns a channel of results. Note that the order of results is not guaranteed due to concurrent processing.

#### `ConcurrentChannel[T, R](input <-chan T, concurrency int, fn func(T) R) <-chan R`

Processes items from an input channel concurrently using the specified number of worker goroutines. Returns a channel of results. Note that the order of results is not guaranteed due to concurrent processing.

---

### crypto

Hash computation utilities supporting SHA-256 and XXH3 algorithms.

#### `Sha256Bytes(bytes []byte) (string, error)`

Computes the SHA-256 hash of a byte slice and returns it as a hexadecimal string.

#### `Sha256File(filePath string) (string, error)`

Computes the SHA-256 hash of a file at the given path and returns it as a lowercase hexadecimal string.

#### `Sha256String(str string) (string, error)`

Computes the SHA-256 hash of a string and returns it as a hexadecimal string.

#### `Xxh3File(filePath string) (string, error)`

Computes the XXH3 hash of a file at the given path and returns it as a lowercase hexadecimal string. XXH3 is significantly faster than SHA-256 for large files.

---

### fetch

HTTP client utilities for downloading files and making REST API requests.

#### `New(headers map[string]string, retries int) *Fetch`

Creates a new Fetch instance with specified headers and retry settings. Automatically sets User-Agent and Content-Type headers if not provided.

#### `GetText(url string) (string, error)`

Performs a GET request to the specified URL and returns the response body as a string.

#### `GetResult(url string, headers map[string]string, result interface{}) (*resty.Response, error)`

Performs a GET request and unmarshals the JSON response body into the provided result interface.

#### `PostResult(url string, headers map[string]string, body interface{}, result interface{}) (*resty.Response, error)`

Performs a POST request with a JSON body and unmarshals the response into the provided result interface.

#### `NewRequest(url string, filePath string, headers map[string]string) (*Request, error)`

Creates a new download request with the specified URL, file path, and optional headers.

#### `DownloadFile(request *Request) *Response`

Downloads a single file based on the provided request. Supports resume capability, progress tracking, and automatic retries with exponential backoff. Uses BLAKE3 hashing for integrity verification.

#### `GetFileCookies(filePath string) ([]Cookie, error)`

Reads cookies from a Netscape-format cookie file and returns them as a slice of Cookie structs.

#### `GetBrowserCookies(domain string) []Cookie`

Retrieves cookies for a specific domain from installed browsers on the system.

#### `CookiesToHeader(cookies []Cookie) string`

Converts a slice of cookies into a properly formatted Cookie header string.

---

### fs

File system operations including temporary file/directory creation, user config management, and archive extraction.

#### `CopyFiles(sources []string, destDir string, flags CopyFlags, exts []string) error`

Copies files and/or directories to a destination directory with flexible options. The flags parameter controls copy behavior. The exts parameter filters files by extension. If nil or empty, no extension filtering is applied.

#### `MoveFiles(sources []string, destDir string, flags CmFlags, exts []string) error`

Moves files and/or directories to a destination directory with flexible options. The flags parameter controls move behavior (CmRecursive for subdirectories, CmPreserveStructure to maintain directory structure). The exts parameter filters files by extension. If nil or empty, no extension filtering is applied.

#### `FileExists(path string) bool`

Checks if a file exists at the specified path. Returns true if the path exists and is a file (not a directory). Returns false if the path does not exist or if it is a directory.

#### `ListPath(directory string, flags Flags, fileExt []string) ([]string, error)`

Traverses a directory and returns a list of paths based on flags (LpDir, LpFile, LpRecursive) and file extensions. Extensions are case-insensitive and should include the dot (e.g., ".txt").

#### `MkTempDir(pattern string) (string, func(), error)`

Creates a temporary directory with the given pattern prefix and returns the directory path along with a cleanup function that should be deferred.

#### `MkTempFile(directory string, pattern string) (*os.File, func(), error)`

Creates a temporary file in a directory with the given pattern and returns the file object along with a cleanup function.

#### `MkUserConfigDir(name string, parts ...string) (string, error)`

Creates a directory within the user's platform-specific configuration directory (e.g., ~/.config on Linux). Supports nested subdirectories through optional path parts.

#### `MkUserConfigFile(name string, parts ...string) (*os.File, error)`

Creates a file in the user's configuration directory with the specified application name and path components. Creates all necessary parent directories if they don't exist.

#### `Unzip(zipPath, targetDirectory string) error`

Extracts all files and directories from a ZIP archive to a target directory. Implements security measures to prevent Zip Slip attacks by validating paths and preventing traversal.

#### `UntarXz(tarXzPath, targetDirectory string) error`

Extracts all files and directories from a TAR.XZ archive to a target directory. Includes security measures against path traversal attacks and preserves file permissions.

---

### github

GitHub API utilities for release management.

#### `GetLatestRelease(owner, repo string) (*github.RepositoryRelease, error)`

Retrieves the latest published release for the specified GitHub repository, including tag name, body, assets, and other metadata.

#### `GetReleaseByName(owner, repo, tagName string) (*github.RepositoryRelease, error)`

Retrieves a specific release by its tag name for the specified GitHub repository.

#### `IsOutdatedRelease(owner, repo, version string) bool`

Checks if a given version is outdated compared to the latest release of a GitHub repository using semantic version comparison. Automatically handles version prefixes and returns false on errors.

---

### memo

Memoization utilities with support for memory-only, disk-only, and hybrid memory-disk caching strategies.

#### `NewMemoryOnly(opts CacheOpts) (*Memoizer, error)`

Creates a new Memoizer instance that uses only in-memory storage. Supports configuration of maximum entries and capacity.

#### `NewDiskOnly(directory string, opts CacheOpts) (*Memoizer, error)`

Creates a new Memoizer that uses disk-based storage (Badger database) in the specified directory. Persists cached values between runs.

#### `NewMemoryDisk(path string, opts CacheOpts, promoteTTL time.Duration) (*Memoizer, func() error, error)`

Creates a new Memoizer with a two-tier memory-disk composite store. Uses in-memory cache as L1 and disk-based cache as L2, with automatic promotion of disk hits to memory.

#### `NewMemoizer(store internal.Store) *Memoizer`

Creates a new Memoizer instance with a custom store implementation.

#### `Do[T any](m *Memoizer, ctx context.Context, key string, ttl time.Duration, compute func(context.Context) (T, error)) (T, error)`

Executes a memoized computation with the given key and TTL. Checks cache first, uses singleflight to deduplicate concurrent calls, executes the compute function on cache miss, and caches the result.

#### `KeyFrom(parts ...any) string`

Generates a SHA-256 hash key from the provided parts using gob encoding. Useful for creating consistent cache keys from multiple values.

#### `Close() error`

Closes the Memoizer and releases any resources held by the underlying store. Should be called when the Memoizer is no longer needed.

---

### time

Time and duration utilities for ETA calculation and custom time formats.

#### `CalculateEta(total, completed int, elapsed time.Duration) time.Duration`

Estimates the time remaining to complete a task based on progress made so far. Calculates average time per completed unit and extrapolates for remaining work.

#### `EpochTime`

A wrapper around time.Time that handles JSON unmarshalling of epoch time (Unix timestamps). Automatically converts numeric JSON values to time.Time objects.

#### `NotzTime`

A wrapper around time.Time that handles JSON unmarshalling of time strings without timezone information. Parses timestamps in the format "2006-01-02T15:04:05".

---

### types

Generic utility types.

#### `Result[T any]`

A generic struct that represents the result of an operation, containing both data of type T and an error. Provides an `IsSuccess()` method that returns true if no error occurred.

## 📝 License

**go-sak** is released under the MIT License. See [LICENSE](LICENSE) for details.

## 👨🏾‍💻 Author

Vinicius Egidio ([vinicius.io](http://vinicius.io))
