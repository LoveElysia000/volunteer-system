package util

import (
	"crypto/rand"
	"errors"
)

// RandomIndex returns a random index in [0, n).
// Note: byte % n has modulo bias when n is not a factor of 256.
func RandomIndex(n int) (int, error) {
	if n <= 0 {
		return 0, errors.New("随机范围无效")
	}

	randomByte := make([]byte, 1)
	if _, err := rand.Read(randomByte); err != nil {
		return 0, err
	}
	return int(randomByte[0]) % n, nil
}
