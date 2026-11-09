package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
)

// newClientID returns an unguessable, collision-resistant client ID.
func newClientID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand unavailable: " + err.Error())
	}
	return "c-" + hex.EncodeToString(b)
}

// sessionSigner issues and verifies resume tokens. A client may only reclaim
// an existing ID by presenting the token the server issued for that ID, so IDs
// cannot be spoofed.
type sessionSigner struct {
	secret []byte
}

func (s sessionSigner) sign(clientID string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(clientID))
	return hex.EncodeToString(mac.Sum(nil))
}

func (s sessionSigner) verify(clientID, token string) bool {
	return hmac.Equal([]byte(s.sign(clientID)), []byte(token))
}
