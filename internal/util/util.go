// Package util contains small shared helpers that do not belong to a domain package.
package util

// Clamp limits v to the inclusive [lo, hi] range.
func Clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Min returns the smaller integer.
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Max returns the larger integer.
func Max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
