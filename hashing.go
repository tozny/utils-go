package utils

import (
	"encoding/base64"

	"golang.org/x/crypto/blake2b"
)

// HashAndEncodeString uses blake2b to hash the provided string and
// returns the result in a base64 url encoded format. Returning an error if any.
func HashAndEncodeString(toHash string) (string, error) {
	h, err := blake2b.New(32, nil)
	if err != nil {
		return "", err
	}
	_, err = h.Write([]byte(toHash))
	noteName := base64.RawURLEncoding.EncodeToString(h.Sum(nil))
	return noteName, err
}
