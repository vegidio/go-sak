package string

import "strings"

// RightOf returns everything to the right of the first or last occurrence of a substring.
//
// Returns an empty string if the substring is not found.
//
// Example:
//
//	RightOf("hello-world-test", "-", false)  // returns "world-test"
//	RightOf("hello-world-test", "-", true)   // returns "test"
//	RightOf("hello", "x", false)             // returns ""
func RightOf(s, sub string, useLast bool) string {
	var i int
	if useLast {
		i = strings.LastIndex(s, sub)
	} else {
		i = strings.Index(s, sub)
	}

	if i == -1 {
		return ""
	}

	return s[i+len(sub):]
}
