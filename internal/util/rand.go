package util

import (
	"math/rand"
	"time"
)

var r = rand.New(rand.NewSource(time.Now().UnixNano()))

func RandInt(min, max int) int {
	if min >= max || min == 0 || max == 0 {
		return max
	}
	return r.Intn(max-min) + min
}
