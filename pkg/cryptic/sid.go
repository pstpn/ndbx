package cryptic

import "crypto/rand"

func SID() string { return rand.Text() + rand.Text() }
