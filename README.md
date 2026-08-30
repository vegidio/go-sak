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

#### `ConcurrentChannelContext[T, R](ctx context.Context, input <-chan T, concurrency int, fn func(T) R) <-chan R`

As `ConcurrentChannel`, but stops when `ctx` is cancelled. A `concurrency` below 1 is clamped to a single worker.

---

### crypto

Hash computation utilities supporting SHA-256 and XXH3 algorithms.

#### `Sha256Bytes(bytes []byte) (string, error)`

Computes the SHA-256 hash of a byte slice and returns it as a hexadecimal string.

#### `Sha256String(str string) (string, error)`

Computes the SHA-256 hash of a string and returns it as a hexadecimal string.

#### `Sha256Reader(reader io.Reader) (string, error)`

Computes the SHA-256 hash of a reader and returns it as a lowercase hexadecimal string.

#### `Sha256File(filePath string) (string, error)`

Computes the SHA-256 hash of a file at the given path and returns it as a lowercase hexadecimal string.

#### `Xxh3Bytes(bytes []byte) (string, error)`

Computes the XXH3 hash of a byte slice and returns it as a hexadecimal string. XXH3 is significantly faster than SHA-256.

#### `Xxh3String(str string) (string, error)`

Computes the XXH3 hash of a string and returns it as a hexadecimal string. XXH3 is significantly faster than SHA-256.

#### `Xxh3Reader(reader io.Reader) (string, error)`

Computes the XXH3 hash of a reader and returns it as a lowercase hexadecimal string. XXH3 is significantly faster than SHA-256.

#### `Xxh3File(filePath string) (string, error)`

Computes the XXH3 hash of a file at the given path and returns it as a lowercase hexadecimal string. XXH3 is significantly faster than SHA-256 for large files.

---

### fetch

HTTP client utilities for downloading files and making REST API requests.

#### `New(headers map[string]string, retries int, disableHttp2 bool) *Fetch`

Creates a new Fetch instance with specified headers and retry settings. Automatically sets User-Agent and Content-Type headers if not provided. Redirects are capped at 5 and https→http downgrade is refused.

#### `GetText(ctx context.Context, url string) (string, error)`

Performs a GET request to the specified URL and returns the response body as a string. The supplied context controls cancellation and deadlines.

#### `GetResult(ctx context.Context, url string, headers map[string]string, result any) (*resty.Response, error)`

Performs a GET request and unmarshals the JSON response body into the provided result. The supplied context controls cancellation and deadlines.

#### `PostResult(ctx context.Context, url string, headers map[string]string, body any, result any) (*resty.Response, error)`

Performs a POST request with a JSON body and unmarshals the response into the provided result. The supplied context controls cancellation and deadlines.

#### `NewRequest(url string, filePath string, headers map[string]string) (*Request, error)`

Creates a new download request with the specified URL, file path, and optional headers.

#### `DownloadFile(request *Request) *Response`

Downloads a single file based on the provided request. Supports resume capability, progress tracking, and automatic retries with exponential backoff. Uses BLAKE3 hashing for integrity verification.

#### `DownloadFiles(requests []*Request, parallel int) (<-chan *Response, func())`

Downloads several files at once, at most `parallel` at a time. Returns a channel of responses and a function that cancels every download still in flight. It is safe to stop reading the channel and then cancel; the channel is still closed.

#### `Response.Track(callback func(completed, total int64, progress float64)) error`

Reports progress until the download ends, then fires the callback one final time with the terminal state — exactly once, including for downloads that fail without transferring anything. Returns the download's error.

#### `Response.BytesDownloaded() int64` / `TotalSize() int64` / `ProgressRatio() float64`

Read the download's progress. Unlike the `Downloaded`, `Size` and `Progress` fields, these are safe to call while the transfer is still running.

#### `Response.Error() error` / `IsComplete() bool` / `Cancel()` / `Bytes() ([]byte, error)`

`Error` blocks until the download finishes and returns its error. `IsComplete` reports whether it has finished without blocking. `Cancel` stops it. `Bytes` reads the downloaded file back from disk.

#### `GetFileCookies(filePath string) ([]Cookie, error)`

Reads cookies from a Netscape-format cookie file and returns them as a slice of Cookie structs.

#### `GetBrowserCookies(domain string) []Cookie`

Retrieves cookies for a specific domain from installed browsers on the system.

#### `CookiesToHeader(cookies []Cookie) string`

Converts a slice of cookies into a properly formatted Cookie header string.

#### `Request`

Represents a download request containing the URL and file path for a download operation. Created via `NewRequest()`.

#### `Response`

Represents the state and result of a download operation. Contains status information, progress tracking, and download metadata.

#### `Cookie`

Represents an HTTP cookie with Name and Value fields. Used for cookie management in download requests.

---

### fs

File system operations including temporary file/directory creation, user config management, and archive extraction.

#### `CopyFiles(sources []string, destDir string, flags CmFlags, exts []string) error`

Copies files and/or directories to a destination directory with flexible options. The flags parameter controls copy behavior. The exts parameter filters files by extension. If nil or empty, no extension filtering is applied.

#### `MoveFiles(sources []string, destDir string, flags CmFlags, exts []string) error`

Moves files and/or directories to a destination directory with flexible options. The flags parameter controls move behavior (CmRecursive for subdirectories, CmPreserveStructure to maintain directory structure). The exts parameter filters files by extension. If nil or empty, no extension filtering is applied.

#### `FileExists(path string) bool`

Checks if a file exists at the specified path. Returns true if the path exists and is a file (not a directory). Returns false if the path does not exist or if it is a directory.

#### `ListPath(directory string, flags ListFlags, fileExt []string) ([]string, error)`

Traverses a directory and returns a list of paths based on flags (LpDir, LpFile, LpRecursive) and file extensions. Extensions are case-insensitive and should include the dot (e.g., ".txt").

#### `MkTempDir(pattern string) (string, func(), error)`

Creates a temporary directory with the given pattern prefix and returns the directory path along with a cleanup function that should be deferred.

#### `MkTempFile(directory string, pattern string) (*os.File, func(), error)`

Creates a temporary file in a directory with the given pattern and returns the file object along with a cleanup function.

#### `MkUserConfigDir(name string, parts ...string) (string, error)`

Creates a directory within the user's platform-specific configuration directory (e.g., ~/.config on Linux). Supports nested subdirectories through optional path parts.

#### `MkUserConfigFile(name string, parts ...string) (*os.File, error)`

Creates a file in the user's configuration directory with the specified application name and path components. Creates all necessary parent directories if they don't exist.

#### `Unzip(zipPath, targetDirectory string, opts ...ExtractOption) error`

Extracts a ZIP archive into a target directory.

#### `Un7zip(sevenZipPath, targetDirectory string, opts ...ExtractOption) error`

Extracts a 7z archive into a target directory.

#### `UntarXz(tarXzPath, targetDirectory string, opts ...ExtractOption) error`

Extracts a TAR.XZ archive into a target directory.

All three share one extraction core. Extraction is confined to `targetDirectory` by an [`os.Root`](https://pkg.go.dev/os#Root), so neither a hostile entry name nor a chain of symbolic links planted by earlier entries in the archive can write outside it. Entry names that are absolute, that escape the root, or that name a reserved Windows device are rejected before anything is created. Extracted files keep the permission bits recorded in the archive, subject to the process umask; setuid, setgid and sticky bits are always stripped.

Errors wrap `ErrIllegalPath`, `ErrIllegalSymlink` or `ErrLimitExceeded`, so they can be told apart with `errors.Is`.

#### `ExtractOption`

Per-call limits and overrides for the three extractors:

| Option | Effect |
| --- | --- |
| `WithMaxTotalBytes(n int64)` | Caps the total uncompressed bytes one call may write |
| `WithMaxFileBytes(n int64)` | Caps the uncompressed size of any single entry |
| `WithMaxEntries(n int)` | Caps how many entries the archive may contain |
| `WithoutSymlinks()` | Rejects archives containing symbolic links outright |
| `WithFileMode(mode fs.FileMode)` | Applies `mode` to every extracted file instead of the archive's own |

Regardless of these, an entry is never allowed to produce more bytes than it declared.

---

### github

GitHub API utilities for release management.

#### `GetLatestRelease(ctx context.Context, owner, repo string) (*github.RepositoryRelease, error)`

Retrieves the latest published release for the specified GitHub repository, including tag name, body, assets, and other metadata. The supplied context controls cancellation and deadlines.

#### `GetReleaseByName(ctx context.Context, owner, repo, tagName string) (*github.RepositoryRelease, error)`

Retrieves a specific release by its tag name for the specified GitHub repository. The supplied context controls cancellation and deadlines.

#### `IsOutdatedRelease(ctx context.Context, owner, repo, version string) bool`

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

Generates a SHA-256 hash key from the provided parts using JSON encoding. Useful for creating consistent cache keys from multiple values. Parameter order is significant.

#### `CacheOpts`

Configuration options for cache stores. Contains `MaxEntries` (maximum number of cached entries) and `MaxCapacity` (maximum capacity in bytes). Used when creating memory or disk-based memoizers.

#### `Cleanup(ctx context.Context) error`

Reclaims the disk space still held by entries whose TTL has expired. Disk-backed memoizers run this automatically in the background when they are created; call it directly to reclaim space at a moment of your choosing. No-op for memory-only memoizers.

#### `Close() error`

Closes the Memoizer and releases any resources held by the underlying store. Should be called when the Memoizer is no longer needed.

See [memo/README.md](memo/README.md) for a fuller guide to this package.

---

### o11y

Observability utilities for logging with OpenTelemetry integration.

#### `NewTelemetry(endpoint, serviceName, version string, headers map[string]string, environment OtelEnvironment, enabled bool) (*Telemetry, error)`

Creates a `Telemetry` that ships log records to an OpenTelemetry collector over OTLP/HTTP, enriched with the application version, a machine identifier, the OS and architecture, a session id, and an approximate geolocation.

When `enabled` is false no exporter is installed and no record ever leaves the process. Nothing is sent over the network to build the enrichment either — in particular the geolocation lookup, which would disclose the caller's public IP to a third party, is skipped entirely.

The returned `*Telemetry` is never nil and is safe to use and to `Close` even when the error is non-nil.

#### `Telemetry.LogInfo(event string, fields map[string]any)`

Emits an informational record. `LogWarn` and `LogError(event, fields, err)` are the same at warning and error severity. None of them modifies the `fields` map you pass in.

#### `Telemetry.RenewSession()`

Assigns a new session id to every record emitted from then on. Safe to call concurrently with the `Log` methods.

#### `Telemetry.Close() error`

Flushes buffered records and shuts the exporter down.

#### `FetchGeolocation(baseURL ...string) (*Geolocation, error)`

Looks up the approximate location of the current public IP address via ipinfo.io.

#### `OtelEnvironment`

Specifies the OpenTelemetry environment configuration: `EnvDevelopment`, `EnvProduction`.

---

### os

Operating system utilities for environment management.

#### `AppendEnvPath(envvar string, path string)`

Appends a directory path to a PATH-like environment variable, using the OS-specific path list separator.

#### `ReExec(envVars ...string) error`

Replaces the current process with a fresh instance of itself, preserving the command-line arguments and adding the given `KEY=VALUE` environment entries. Use it only for variables that cannot be changed after the program starts, such as `LD_LIBRARY_PATH`; prefer `os.Setenv` otherwise.

`APP_REEXEC=1` is set automatically to stop the re-executed process from doing it again. On success this call never returns, because the process image has been replaced; it returns an error if the executable cannot be located, an entry is malformed, or the exec fails — including on Windows, which has no `execve`.

---

### string

String manipulation utilities.

#### `RightOf(s, sub string, useLast bool) string`

Returns everything to the right of the first or last occurrence of a substring. Returns an empty string if the substring is not found. When `useLast` is true, uses the last occurrence; otherwise uses the first occurrence.

---

### sysinfo

Hardware information about the host machine, on Linux, macOS and Windows.

#### `GetCPUInfo() (CPUInfo, error)`

Returns the CPU's model name and core count. Reads `/proc/cpuinfo` or `lscpu` on Linux, `sysctl` on macOS, and CIM via PowerShell on Windows.

#### `GetMemoryInfo() (MemoryInfo, error)`

Returns the machine's total physical RAM, in **bytes**.

#### `GetGPUInfo() ([]GPUInfo, error)`

Returns every GPU the machine reports, with its name, vendor and memory in MiB. Uses `system_profiler` on macOS, `nvidia-smi` merged with the DRM sysfs tree and `lspci` on Linux, and `nvidia-smi` merged with CIM on Windows.

Every external tool is run with a timeout, and the well-known system utilities are invoked by absolute path so that a writable `PATH` entry cannot inject code into the calling process.

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

## ⚠️ Breaking changes

The following releases change behaviour that existing code may rely on.

### Unreleased

- **`sysinfo.MemoryInfo.Total` is now genuinely in bytes.** It was documented as bytes but every backend divided by 1,000,000 and returned megabytes, so values read from it are now roughly a million times larger. Divide by `1_000_000` at the call site to restore the old figure.
- **`fs.Unzip` and `fs.Un7zip` no longer force every extracted file to `0o755`.** Files keep the permission bits recorded in the archive, matching what `fs.UntarXz` always did. An extracted key or config file is no longer left world-readable and world-executable. Pass `fs.WithFileMode(0o755)` to restore the old behaviour.
- **`os.ReExec` now returns an `error`.** It previously discarded every failure, which made it a silent no-op on Windows — where `syscall.Exec` cannot replace a process — while its documentation promised otherwise.
- **`o11y.NewTelemetry` now returns `(*Telemetry, error)`.** The exporter's initialisation error was previously dropped, and one of its failure paths left the `Telemetry` in a state where `Close` panicked. The returned value is never nil and is safe to use and close even when the error is non-nil.
- **`o11y` no longer contacts ipinfo.io when telemetry is disabled**, and identifies the machine with an application-scoped `machineid.ProtectedID` rather than the raw host id.
- **`memo.Memoizer.Sf` is now unexported.** `singleflight.Group` embeds a `sync.Mutex`, so exporting it made a `Memoizer` copied by value silently unsafe.
- **The three `fs` extractors take variadic `ExtractOption` arguments.** Existing two-argument calls are unaffected; only code assigning them to a `func(string, string) error` variable needs changing.

## 📝 License

**go-sak** is released under the MIT License. See [LICENSE](LICENSE) for details.

## 👨🏾‍💻 Author

Vinicius Egidio ([vinicius.io](http://vinicius.io))
