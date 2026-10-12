package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
)

type HMACPANHasher struct {
	salt []byte
}

func NewHMACPANHasher(salt string) HMACPANHasher {
	return HMACPANHasher{salt: []byte(salt)}
}

func (h HMACPANHasher) Hash(pan string) (string, error) {
	mac := hmac.New(sha256.New, h.salt)
	mac.Write([]byte(pan))
	return hex.EncodeToString(mac.Sum(nil)), nil
}
