package samplecode_test

import (
	"fmt"
	"testing"
)

const setSize = 1000_000

func makeKeys() []string {
	keys := make([]string, 0, setSize)
	for i := 0; i < setSize; i++ {
		keys = append(keys, fmt.Sprintf("BUS-%06d", i))
	}
	return keys
}

func BenchmarkSetBool(b *testing.B) {
	keys := makeKeys()

	for i := 0; i < b.N; i++ {
		m := make(map[string]bool, len(keys))
		for _, k := range keys {
			m[k] = true
		}
	}
}

func BenchmarkSetStruct(b *testing.B) {
	keys := makeKeys()

	for i := 0; i < b.N; i++ {
		m := make(map[string]struct{}, len(keys))
		for _, k := range keys {
			m[k] = struct{}{}
		}
	}
}
