package cryptic

import (
	"crypto/rand"
	"encoding/hex"
)

const sidSizeBytes = 32

func MustSID() string {
	b := make([]byte, sidSizeBytes)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}

	return hex.EncodeToString(b)
}
