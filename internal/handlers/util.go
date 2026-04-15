package handlers

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"strconv"
)

func randomHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func randomDecimalChallenge() string {
	b := make([]byte, 8)
	rand.Read(b)
	n := binary.BigEndian.Uint64(b)
	return strconv.FormatUint(n, 10)
}
