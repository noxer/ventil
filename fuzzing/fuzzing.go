package fuzzing

import (
	"bytes"

	"github.com/noxer/ventil"
)

// Fuzz allows us to fuzz this library
func Fuzz(data []byte) int {
	kv, err := ventil.Parse(bytes.NewReader(data), nil)
	if err != nil {
		return 0
	}

	_ = kv.String()
	return 1
}
