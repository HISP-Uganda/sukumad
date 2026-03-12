package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

func NewResetToken() (plain string, hash string, err error) {
	var b [32]byte
	_, err = rand.Read(b[:])
	if err != nil {
		return "", "", err
	}
	plain = hex.EncodeToString(b[:])
	h := sha256.Sum256([]byte(plain))
	hash = hex.EncodeToString(h[:])
	return
}

func HashResetToken(plain string) string {
	h := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(h[:])
}
