package identity

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/tozny/utils-go/logging"
	"github.com/tozny/utils-go/server"
	"github.com/tozny/utils-go/test"
	"gopkg.in/square/go-jose.v2"
	"gopkg.in/square/go-jose.v2/jwt"
)

var sampleJWKS = `{
	"keys": [
		{
			"kid":"example1",
			"kty":"RSA",
			"alg":"RS256",
			"use":"sig",
			"n":"pVQtaCq_tYAoxMx3q2-yFwe7r4T8yxNnKC9C1bJpq79cjadM1n3ES0LODTv29UTM1clgOIC8bdbxlv_N6gZAuSfkn9Oczm8p_wDkCq7w688FUaOhOvoffVlpUFx9g4yKw3iI5sGClpnANy0ybyRU8gZRadBi-179iI_5S6XZgGNo8TRd3LQaD8prw1G3Zp5JvSrfcR4VdU_oQLEQV8ERASwB-3iAQx6FmOYw1iaF368up1VlL9-6nFYDyYRroifbbiTllAL-fPEguYdPZ7UiNaU0QGLkJbQYsCIzpzsEbioQbuSBgXcVcxjxqJGO79h2z3yea0sY54imp9X16gySMQ",
			"e":"AQAB"
		},
		{
			"kid":"example2",
			"kty":"RSA",
			"alg":"RS256",
			"use":"sig","n":"jrpSnnCXb3hllL-IBcQSu9kkZWTIFdzZkOAXBwXvGQqNTJywXD4ABAoSc2KO07AEtiX72tu1KipnIhEU8uF3aX0fPPyK-5Okp1GJ-oLHZUBobmML-DCav07oLoDJaLI4gzOUL8GoeHUaIEw8Otkst4fcZOZvWZP_TRkGC3GkDY4EiUOGnsORy6vIzOexpDe1bFqwj-cyMafsbumHHNc5neHrshrrD9ZQm7JszzHcnf7VLJuZo2XZrfs0BuGp20llIVC6-Mz2AZRtQKF2kJ1PwyXXAQJJDf1ADNjiDWDV-fNMpQZKxeq5BOeWIOxe_IjmS4uqXGxCUHBPKCWHLycvJQ",
			"e":"AQAB"
		},
		{
			"kid":"example3",
			"kty":"RSA",
			"alg":"RS256",
			"use":"enc",
			"n":"n9u2tWKM92zdm1UcMsCQwPthG5P9i_zVDUM48zWj5CBpPgwR3dWhbCvTz_sIrSQkbVVBegIB_bxENZmrLsGQ1xQtm5PDLfUoXCmNSTgnbBouOoA-wITaLXbY-cFRsI8V-4E6Fc5tFucf0wYgThQ_QjRC1dI8WvKuv4W7Dtn9Oe1RKO6HgX2h2_wjRPqmJnY6rsyQ-wDOlgWI9qiz0ra2zAUb5jYLZR-OhxLZEsm6QE4jjBhSoS08PQAPVvYUOCF2dVG_fp2epyZZOX_puSNHcUFqLMlMD22CccX4kPQTtwLrWTeE47LOFnicUVJg_MtnGP58LyiYIt0SRF8MbIR8mOzo6M7Eu4JLzRKnTRijqSGnDOdKawKmlZLezbgewpxs2Q4E5DgBYzk-xVC37lHuBugubHxCS6VSWdrcpp0XChspKTEpvhM3-8N80VJOJVDGNLHuUbWzxGTxKU_mhLDInExLap_H2ZVCiw7LQi2KlaUJ6YoNE_vtJpk7CeD8ax09",
			"e":"AQAB"
		}
	]
}`

func TestCanFetchJWKS(t *testing.T) {
	// Set up a test server to respond with the example JWKS
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, sampleJWKS)
	}))
	defer testServer.Close()
	logger := logging.NewServiceLogger(ioutil.Discard, "", "ERROR")
	jwks := NewJWKS(testServer.URL, 300, &logger)
	set, err := jwks.Set(context.Background())
	if err != nil {
		t.Fatalf("unable to load JWKS from endpoint %q: %+v", testServer.URL, err)
	}
	if len(set.Keys) != 3 {
		t.Errorf("expected to find 3 JWKs in the set, but found %d: %+v", len(set.Keys), jwks)
	}
}

func TestValidateRequests(t *testing.T) {
	testResponseBody := "You are authenticated"
	// Set up new key
	privateKey, err := newRSASigKey(2048, "RS256")
	if err != nil {
		t.Fatalf("unable to generate JWK: %+v", err)
	}
	// Set up signer
	signerKey := jose.SigningKey{Algorithm: jose.SignatureAlgorithm(privateKey.Algorithm), Key: &privateKey}
	var signerOpts = jose.SignerOptions{}
	signerOpts.WithType("JWT")
	signer, err := jose.NewSigner(signerKey, &signerOpts)
	if err != nil {
		t.Fatalf("unable to create signer: %+v", err)
	}
	// Create a test JWT
	claims, token, err := newTestToken(signer)
	if err != nil {
		t.Fatalf("error creating test token: %+v", err)
	}
	// Set up the public key in a JWKS
	var set jose.JSONWebKeySet
	set.Keys = append(set.Keys, privateKey.Public())
	publicJWKS, err := json.Marshal(&set)
	if err != nil {
		t.Fatalf("unable to marshal public JWKS: %+v", err)
	}
	// Set up a test server to respond with the example JWKS
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(publicJWKS)
	}))
	defer testServer.Close()
	// Set up a new JWKS struct
	logger := logging.NewServiceLogger(ioutil.Discard, "", "ERROR")
	jwks := NewJWKS(testServer.URL, 300, &logger)
	// See if the JWKS middleware validates the token
	expected := Expected{
		Issuer:   claims.Issuer,
		Subject:  claims.Subject,
		Audience: claims.Audience,
		ID:       claims.ID,
	}
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", token))
	handler := server.ApplyMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(testResponseBody))
	}), jwks.Middleware(expected))
	handler.ServeHTTP(recorder, req)
	// Validate the response
	resp := recorder.Result()
	test.AssertRespStatus(t, "jwks middleware success test", resp, http.StatusOK)
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("unable to read response body: %+v", err)
	}
	if string(body) != testResponseBody {
		t.Errorf("unexpected response body. Expected: %q Recieved: %q", testResponseBody, body)
	}
}

func newRSASigKey(size int, alg string) (jose.JSONWebKey, error) {
	// Set up return values
	var key jose.JSONWebKey
	// Set up new RSA Key
	reader := rand.Reader
	privateKey, err := rsa.GenerateKey(reader, size)
	if err != nil {
		return key, err
	}
	// Private JWK
	key.KeyID = uuid.New().String()
	key.Algorithm = alg
	key.Use = "sig"
	key.Key = privateKey
	//Send it back
	return key, nil
}

func newTestToken(signer jose.Signer) (jwt.Claims, string, error) {
	now := time.Now()
	claims := jwt.Claims{
		Issuer:    "test_issuer",
		Subject:   uuid.New().String(),
		Audience:  jwt.Audience{"test1", "test2"},
		NotBefore: jwt.NewNumericDate(time.Time{}),
		IssuedAt:  jwt.NewNumericDate(now),
		Expiry:    jwt.NewNumericDate(now.Add(1 * time.Hour)),
	}

	token, err := jwt.Signed(signer).Claims(claims).FullSerialize()
	// if there was an error, it is getting returned here
	return claims, token, err
}
