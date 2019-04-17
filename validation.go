package utils

import (
	"encoding/base64"
	"strings"
)

// IsValidKey ensures a base64URL encoded key of a specific type is base64URL
// encoded and has the right number of bytes for a key of its type.
func IsValidKey(key string, keyType string) bool {
	numBytes, ok := IsValidBase64URL(key)
	if !ok {
		return false
	}
	switch keyType {
	case "Curve25519", "Ed25519":
		return numBytes == 32
	case "P384":
		return numBytes == 97
	}
	// Invalid key type is never valid
	return false
}

// IsValidBase64URL checks to see if a string has a valid base64URL encoded
// value, ruturning the number of bytes
func IsValidBase64URL(subject string) (int, bool) {
	counter, err := base64.RawURLEncoding.DecodeString(subject)
	return len(counter), err == nil
}

// IsValidDotted is a basic check for base64URL dot serialized strings.It returns
// the number of dotted parts, and a boolean representing if all of the
// parts are valid Base64URL encoded.
func IsValidDotted(subject string) (int, bool) {
	parts := strings.Split(subject, ".")
	for _, part := range parts {
		if _, ok := IsValidBase64URL(part); !ok {
			return len(parts), false
		}
	}
	return len(parts), true
}
