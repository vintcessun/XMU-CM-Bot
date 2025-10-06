package utils

import (
	"math/rand"
	"time"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func Uniform(min, max float64) float64 {
	if min > max {
		min, max = max, min
	}
	return min + r.Float64()*(max-min)
}
