package types

// Result is a generic struct that represents the result of an operation.
//
// Parameters:
//   - Data is data of type T.
//   - Err is an error that indicates if the operation failed.
type Result[T any] struct {
	Data T
	Err  error
}

// IsSuccess returns true if the operation was successful (no error occurred), false otherwise.
func (r *Result[T]) IsSuccess() bool {
	return r.Err == nil
}
