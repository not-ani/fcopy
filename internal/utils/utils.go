package utils

// Min returns the minimum of three integers
func Min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

// Abs returns the absolute value of an integer
func Abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// CalculateSimilarity returns the Levenshtein distance between two strings
// Lower values mean strings are more similar
func CalculateSimilarity(s1, s2 string) int {
	// Levenshtein distance implementation
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	// Create two work vectors of integer distances
	v0 := make([]int, len(s2)+1)
	v1 := make([]int, len(s2)+1)

	// Initialize v0 (the previous row of distances)
	for i := 0; i <= len(s2); i++ {
		v0[i] = i
	}

	// Calculate v1 (current row distances) from the previous row v0
	for i := 0; i < len(s1); i++ {
		// First element of v1 is A[i+1][0]
		v1[0] = i + 1

		// Use formula to fill in the rest of the row
		for j := 0; j < len(s2); j++ {
			cost := 1
			if s1[i] == s2[j] {
				cost = 0
			}
			v1[j+1] = Min(v1[j]+1, v0[j+1]+1, v0[j]+cost)
		}

		// Copy v1 to v0 for next iteration
		for j := 0; j <= len(s2); j++ {
			v0[j] = v1[j]
		}
	}

	return v1[len(s2)]
}
