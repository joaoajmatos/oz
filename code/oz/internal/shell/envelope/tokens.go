package envelope

import "math"

// EstimateTokens uses ceil(chars/4.0) per spec.
func EstimateTokens(s string) int {
	return int(math.Ceil(float64(len(s)) / 4.0))
}
