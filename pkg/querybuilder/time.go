package querybuilder

import "math"

// ToNanoSecs takes an epoch value and returns it in nanoseconds.
func ToNanoSecs(epoch uint64) uint64 {
	temp := epoch
	count := 0
	if epoch == 0 {
		count = 1
	} else {
		for epoch != 0 {
			epoch /= 10
			count++
		}
	}
	return temp * uint64(math.Pow(10, float64(19-count)))
}
