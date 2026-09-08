package memo

import "github.com/vegidio/go-sak/memo/internal"

type CacheOpts = internal.CacheOpts

// ErrNotAdmitted is returned by a Set the store did not accept. Nothing is lost but a future hit - the caller's own
// computation is unaffected - so it is worth distinguishing from a store that actually broke, which is why it is
// re-exported here rather than left inside the internal package where no caller could errors.Is against it.
//
// Only the memory store produces it, and it means the write was dropped rather than that it failed: Ristretto reports
// it when its set buffer is full under concurrent writes, or when the store is closing. A value too large for the
// budget is deliberately not this case - the admission policy discards those asynchronously and the write still
// reports success, so an oversized value is simply a miss the next time it is asked for.
var ErrNotAdmitted = internal.ErrNotAdmitted
