// file: slice_append_benchmark_test.go
package samplecode_test

import "testing"

const n = 100_000

func BenchmarkAppendNoPrealloc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var s []int
		for j := 0; j < n; j++ {
			s = append(s, j)
		}
	}
}

func BenchmarkAppendPrealloc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		s := make([]int, 0, n)
		for j := 0; j < n; j++ {
			s = append(s, j)
		}
	}
}
