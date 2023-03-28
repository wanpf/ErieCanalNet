// Package util defines utilities
package util

import (
	"hash/fnv"
	"time"
)

// Hash calculates an FNV-1 hash from a given byte array,
// returns it as an uint64 and error, if any
func Hash(bytes []byte) uint64 {
	if hashCode, err := HashFromString(string(bytes)); err == nil {
		return hashCode
	}
	return uint64(time.Now().Nanosecond())
}

// HashFromString calculates an FNV-1 hash from a given string,
// returns it as an uint64 and error, if any
func HashFromString(s string) (uint64, error) {
	h := fnv.New64()
	_, err := h.Write([]byte(s))
	if err != nil {
		return 0, err
	}

	return h.Sum64(), nil
}
