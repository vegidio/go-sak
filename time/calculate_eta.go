package time

import gotime "time"

// CalculateEta estimates the time remaining to complete a task based on progress made so far.
//
// It calculates the estimated time of arrival (ETA) by analyzing the average time per completed unit and extrapolating
// for the remaining work.
//
// # Parameters:
//   - total: The total number of units to complete (must be > 0)
//   - completed: The number of units already completed (must be > 0)
//   - elapsed: The time duration spent completing the current units (must be > 0)
//
// # Returns:
//   - The estimated duration to complete the remaining work
//   - Returns 0 if the task is already complete (completed >= total)
//   - Returns 7 days (168 hours) as a fallback for invalid inputs
//
// # Example:
//
//	// If 3 out of 10 tasks completed in 30 minutes
//	eta := CalculateEta(10, 3, 30*time.Minute)
//	// Returns approximately 70 minutes (for the remaining 7 tasks)
func CalculateEta(total, completed int, elapsed gotime.Duration) gotime.Duration {
	// Validate inputs
	if total <= 0 || completed <= 0 || elapsed <= 0 {
		return gotime.Duration(7 * 24 * gotime.Hour)
	}

	// Nothing to do
	if completed >= total {
		return 0
	}

	remaining := total - completed
	avgPerTask := elapsed / gotime.Duration(completed)
	eta := avgPerTask * gotime.Duration(remaining)

	return eta
}
