# memo

Memoization with pluggable storage: keep the result of an expensive call around for a while, in memory, on disk, or
both, and don't compute it twice.

```bash
go get github.com/vegidio/go-sak
```

## What it does

Wrap any function that's expensive to call — an HTTP request, a database query, an image resize — and `memo` will:

- **Cache the result** under a key you choose, for as long as the TTL you give it.
- **Deduplicate concurrent calls.** If ten goroutines ask for the same missing key at once, the function runs once and
  all ten get that result.
- **Persist between runs**, if you pick a disk-backed store.

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/vegidio/go-sak/memo"
)

func main() {
    m, err := memo.NewMemoryOnly(memo.CacheOpts{})
    if err != nil {
        panic(err)
    }
    defer m.Close()

    ctx := context.Background()

    for range 2 {
        // "fetching..." prints once: the second call is served from cache, until the TTL runs out.
        user, err := memo.Do(m, ctx, "user:42", 5*time.Minute, func(ctx context.Context) (string, error) {
            fmt.Println("fetching...")
            return fetchUser(ctx, 42)
        })
        fmt.Println(user, err)
    }
}
```

## Choosing a store

| | Survives restart | Speed | Use it when |
|---|---|---|---|
| `NewMemoryOnly` | No | Fastest | Data is hot, cheap to recompute, and one process owns it |
| `NewDiskOnly` | Yes | Disk-bound | Results are expensive and must outlive the process |
| `NewMemoryDisk` | Yes | Fast on repeat | The usual pick: memory in front, disk behind it |
| `NewDiskShared` | Yes | Disk-bound | Same as `NewDiskOnly`, but the setup that opens it can run more than once |

### Memory only

Backed by [Ristretto](https://github.com/dgraph-io/ristretto). Everything is lost when the process exits.

```go
m, err := memo.NewMemoryOnly(memo.CacheOpts{
    MaxEntries:  100_000,
    MaxCapacity: 256 << 20, // 256 MiB
})
if err != nil {
    return err
}
defer m.Close()
```

### Disk only

Backed by [Badger](https://github.com/dgraph-io/badger). The directory is created if it doesn't exist, and the cache is
still there next time you start.

```go
m, err := memo.NewDiskOnly("/var/cache/myapp", memo.CacheOpts{})
if err != nil {
    return err
}
defer m.Close()
```

### Disk, opened more than once

Badger takes one lock per directory. It isn't reentrant and can't be waited on, so a process that opens a path it
already holds doesn't block — it fails, with an error that reads exactly like a *different* process holding the lock:

```
Cannot acquire directory lock on "/var/cache/myapp".  Another process is using this Badger database
```

That bites anything whose setup can run twice: a library re-initialised without being torn down first, a desktop app
whose UI reloads while the backend keeps running, a test that opens a fixture per case. `NewDiskShared` returns the
store this process already has open instead of trying to open it again.

```go
m, err := memo.NewDiskShared("/var/cache/myapp", memo.CacheOpts{})
if err != nil {
    return err
}
defer m.Close()
```

Each call returns its own handle, so each needs its own `Close`; the store closes with the last one. Handles share the
store but not the memoization — `Do` deduplicates per memoizer, so two handles computing the same key at once may both
compute it. Directories are matched by absolute path, not by inode, so two symlinks to the same directory still
collide on the lock. Sizing comes from whichever call opens the store.

### Memory + disk

Two tiers: reads try memory first, fall back to disk, and a disk hit is promoted back into memory for `promoteTTL`.
Writes go to both. Note the different signature — this one hands you a cleanup function rather than relying on `Close`:

```go
m, cleanup, err := memo.NewMemoryDisk("/var/cache/myapp", memo.CacheOpts{}, time.Hour)
if err != nil {
    return err
}
defer cleanup()
```

`promoteTTL` is how long a promoted entry stays in memory, and is independent of the TTL you pass to `Do`. Pass `0` to
disable promotion, in which case disk hits are served without being cached in memory.

## `Do` is a function, not a method

Go doesn't allow type parameters on methods, so the memoizer is the first argument rather than the receiver:

```go
result, err := memo.Do(m, ctx, key, ttl, compute)
```

The type is inferred from `compute`, so `Do` returns a real `[]Product`, not an `any` you have to assert:

```go
products, err := memo.Do(m, ctx, "products:featured", time.Hour,
    func(ctx context.Context) ([]Product, error) {
        return db.FeaturedProducts(ctx)
    })
```

`ctx` is passed straight through to `compute`, so cancelling it aborts a slow computation if that function respects it.
The cache lookup itself is fast and doesn't observe cancellation.

## Building keys

`KeyFrom` hashes any set of JSON-encodable values into one key, so you don't have to hand-format strings:

```go
key := memo.KeyFrom("search", query, page, filters)

results, err := memo.Do(m, ctx, key, 10*time.Minute, func(ctx context.Context) ([]Result, error) {
    return search(ctx, query, page, filters)
})
```

Argument order matters — `KeyFrom("a", "b")` and `KeyFrom("b", "a")` are different keys. Maps with string keys encode in
sorted order, so they hash the same every time.

## TTL, expiry, and disk space

Expired entries are never served: once the TTL passes, the entry reads as a miss and `compute` runs again.

Reclaiming the *space* on disk is a separate matter. Badger only discards expired entries while compacting, and it
compacts based on how much has been written, not on the clock — so a cache that goes quiet would otherwise hold onto
expired data indefinitely. Disk-backed memoizers handle this for you: opening one kicks off a background sweep that
deletes expired entries, compacts, and reclaims the freed space. It doesn't slow down the constructor.

To run that sweep yourself — before shutting down, or after a batch of entries expires — call `Cleanup`:

```go
if err := m.Cleanup(ctx); err != nil {
    log.Printf("cache cleanup failed: %v", err)
}
```

It's safe to call at any time, is a no-op on memory-only memoizers, and only one sweep runs at a time. Prefer a quiet
moment: a sweep pauses the compaction that normally keeps up with incoming writes, so writes issued while it runs queue
up and get slower — they don't fail — and disk usage settles over a couple of sweeps rather than dropping to the floor
immediately.

## `CacheOpts`

```go
type CacheOpts struct {
    MaxEntries  int64 // default 1,000,000
    MaxCapacity int64 // default 1 GiB
}
```

For the **memory** store these are real limits: `MaxEntries` sizes Ristretto's admission counters and `MaxCapacity` is
the eviction budget in bytes, counting each entry as its encoded length.

For the **disk** store they are not a size cap. They map to Badger's value-log rollover thresholds, which control how
large a single value-log file gets before a new one is started — not how much the directory may total. **The disk cache
is not size-bounded**; use TTLs to keep it in check, and don't read `MaxCapacity: 512 << 20` as "this directory stays
under 512 MiB".

## Things worth knowing

- **Values are gob-encoded.** They round-trip through `encoding/gob`, so exported fields survive and unexported ones
  don't. Interface-typed values need `gob.Register` before they can be cached.
- **Errors aren't cached.** If `compute` fails, the error goes to every caller waiting on that key and nothing is
  written. The next call retries.
- **Cache writes are best-effort.** If the store fails to persist a result, `Do` still returns it — a broken cache
  degrades to no cache, it doesn't break your call path.
- **Corrupt entries are ignored.** If a cached value can't be decoded into `T` — say the struct changed shape since it
  was written — it's treated as a miss and recomputed. Changing a cached type is safe, but consider versioning the key
  (`memo.KeyFrom("user:v2", id)`) so old entries expire rather than being re-decoded on every read.
- **Close what you open.** `NewMemoryOnly`, `NewDiskOnly` and `NewDiskShared` return a memoizer you `Close`;
  `NewMemoryDisk` returns a cleanup function. Both are safe to call more than once, and both return the same result
  each time. Closing a disk store interrupts any sweep in flight rather than waiting for it to finish.
- **A declined write isn't a failed one.** The memory store returns `memo.ErrNotAdmitted` when it drops a write — its
  set buffer was full, or the store is closing. Nothing is lost but a future hit, so it's worth telling apart from a
  store that actually broke: `errors.Is(err, memo.ErrNotAdmitted)`.
- **`Path()` reports the directory.** Empty for a memory store, the disk tier's directory for a composite one. It
  answers "am I already open on this path?" for code that can't use `NewDiskShared`.

## Custom stores

`NewMemoizer` accepts any store implementation, which is what the three constructors above use internally:

```go
m := memo.NewMemoizer(store)
```

The `Store` interface (`Get`, `Set`, `Cleanup`, `Close`) currently lives in `memo/internal`, so this is only usable from
inside the module — it isn't an extension point for third-party stores yet.
