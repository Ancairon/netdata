package snmp_dev

import (
	"fmt"
	"strconv"
)

// TotalTimeToTemporalPercent converts a total time value to a temporal percentage.
// For a given totalTime and scale, it returns (totalTime / scale) * 100.
func TotalTimeToTemporalPercent(totalTime, scale float64) float64 {
	if scale == 0 {
		scale = 1000
	}
	return (totalTime / scale) * 100
}
func ToInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case string:
		return strconv.Atoi(v)
	default:
		return 0, fmt.Errorf("cannot convert %v to int", value)
	}
}

// TransformIndex applies slice rules to a source index.
func TransformIndex(src []string, slices []IndexSlice) ([]string, bool) {
	var dst []string
	for _, s := range slices {
		if s.Stop > len(src) {
			return nil, false
		}
		dst = append(dst, src[s.Start:s.Stop]...)
	}
	return dst, true
}

// Batches splits a slice into batches of at most 'size' items.
func Batches[T any](items []T, size int) [][]T {
	if size <= 0 {
		panic("batch size must be > 0")
	}
	var result [][]T
	for i := 0; i < len(items); i += size {
		end := i + size
		if end > len(items) {
			end = len(items)
		}
		result = append(result, items[i:end])
	}
	return result
}
