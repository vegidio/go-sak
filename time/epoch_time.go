package time

import (
	"encoding/json"
	"errors"
	"math"
	"time"
)

// EpochTime is a wrapper around time.Time to handle JSON unmarshalling of epoch time.
type EpochTime struct {
	time.Time
}

func (t *EpochTime) UnmarshalJSON(b []byte) error {
	var epoch float64
	if err := json.Unmarshal(b, &epoch); err != nil {
		return err
	}

	if epoch < 0 {
		return errors.New("invalid epoch time")
	}

	// Converting a float64 larger than the int64 range is implementation-defined, so reject it rather than
	// producing an arbitrary time.
	if epoch > math.MaxInt64 {
		return errors.New("invalid epoch time")
	}

	// Split instead of truncating: the JSON number may carry sub-second precision, which time.Unix(sec, 0) would
	// silently discard.
	sec, frac := math.Modf(epoch)
	t.Time = time.Unix(int64(sec), int64(frac*float64(time.Second)))

	return nil
}
