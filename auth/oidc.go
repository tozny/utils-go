package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"time"

	"github.com/pascaldekloe/jwt"
)

// Republish supported algorithm constants
const (
	RS256 = jwt.RS256
)

// A set of claims decoded from or for a JWT
type Claims = jwt.Claims

// TokenFactory wraps the tools needed to generate JWTs from a set of Claims
type TokenFactory struct {
	SigningKey *rsa.PrivateKey
	Algorithm  string
}

// NewTokenFactory sets up a new TokenFactory parsing the singing key and algorithm.
func NewTokenFactory(signingKey string, algorithm string) (*TokenFactory, error) {
	tokenFactory := TokenFactory{
		Algorithm: algorithm,
	}
	privateKey, err := parseRSAKey(signingKey)
	if err != nil {
		return &tokenFactory, fmt.Errorf("could not create token factory: %+v", err)
	}
	tokenFactory.SigningKey = privateKey
	return &tokenFactory, nil
}

// Sign creates a fully signed and encoded JWT from a set of token claims
func (tf *TokenFactory) Sign(claims Claims, validTime time.Duration) ([]byte, error) {
	now := time.Now()
	claims.Issued = jwt.NewNumericTime(now.Round(time.Second))
	if validTime > 0 {
		claims.Expires = jwt.NewNumericTime(now.Add(validTime).Round(time.Second))
	}
	return claims.RSASign(tf.Algorithm, tf.SigningKey)
}

// Parse converts a signed token to its Claims & verifies its signature
// will return a non-nil error if the token was not signed by this TokenFactory's SigningKey
func (tf *TokenFactory) Parse(token []byte) (*Claims, error) {
	return jwt.RSACheck(token, &tf.SigningKey.PublicKey)
}

// ParseAndVerifyToken converts a signed token to its Claims and verifies it signature
// with the passed in public signing key, returning a non-nil error if the token as not signed
// by the signing key.
func ParseAndVerifyToken(token []byte, publicKey string) (*Claims, error) {
	rsaPubicKey, err := x509.ParsePKCS1PublicKey([]byte(publicKey))
	if err != nil {
		return nil, fmt.Errorf("error %s. could not parse public key", err)
	}
	return jwt.RSACheck(token, rsaPubicKey)
}

// parseRSA key takes a base64url RSA private key in PEM format and decodes a useable RSA private key
func parseRSAKey(key string) (*rsa.PrivateKey, error) {
	rsaPrivateKey := &rsa.PrivateKey{}
	privateSigningKeyPEMBytes, err := base64.RawURLEncoding.DecodeString(key)
	if err != nil {
		return rsaPrivateKey, fmt.Errorf("invalid private, %s not valid base64", err)
	}
	block, _ := pem.Decode(privateSigningKeyPEMBytes)
	if block == nil {
		return rsaPrivateKey, errors.New("no PEM block")
	}

	rsaPrivateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return rsaPrivateKey, err
	}

	return rsaPrivateKey, nil
}
