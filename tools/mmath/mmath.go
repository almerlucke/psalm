package mmath

import "math"

func Quantize(v float64, q float64) float64 {
	return math.Round(v/q) * q
}

func Limit(x float64, minVal float64, maxVal float64) float64 {
	return math.Max(math.Min(x, maxVal), minVal)
}
