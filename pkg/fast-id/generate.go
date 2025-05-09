package fast_id

import (
	"encoding/base64"
	"sync/atomic"
	"time"
)

var count atomic.Uint64

func New() string {
	timestamp := time.Now().UnixNano()

	currentCounter := count.Add(1)

	id := base64.URLEncoding.EncodeToString(
		[]byte(
			string(int64ToBytes(timestamp)) +
				string(uint64ToBytes(currentCounter))))

	return id
}

func int64ToBytes(i int64) []byte {
	b := make([]byte, 8)
	for n := uint(0); n < 8; n++ {
		b[n] = byte(i >> (n * 8))
	}
	return b
}

func uint64ToBytes(i uint64) []byte {
	b := make([]byte, 8)
	for n := uint(0); n < 8; n++ {
		b[n] = byte(i >> (n * 8))
	}
	return b
}
