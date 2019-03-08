// Package test provides helper functions and common structs for use in tests across tozny golang repositories.
package test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/tozny/e3db-clients-go"
	"github.com/tozny/e3db-clients-go/accountClient"
	"github.com/tozny/e3db-go/v2"
	"net/http"
	"testing"
)

// MakeHttpRequest attempts to make the provided http request and JSON deserialize the response using the provided result interface , returning the raw http response and error (if any).
func MakeHttpRequest(t *testing.T, method string, url string, body interface{}, result interface{}, headers map[string]string) (*http.Response, error) {
	encodedBody, err := json.Marshal(body)
	if err != nil {
		t.Errorf("error %s encoding body %+v for request %s %s %s\n", err, body, method, url, headers)
	}
	request, err := http.NewRequest(method, url, bytes.NewBuffer(encodedBody))
	if err != nil {
		t.Errorf("error %s constructing http request %s %s %s %s\n", err, encodedBody, method, url, headers)
	}
	client := &http.Client{}
	for key, value := range headers {
		request.Header.Add(key, value)
	}
	response, err := client.Do(request)
	if err != nil {
		t.Errorf("error %s making http request %+v\n", err, request)
		return response, err
	}
	// If no result is expected, don't attempt to decode a potentially
	// empty response stream and avoid incurring EOF errors
	if result == nil {
		return response, err
	}
	err = json.NewDecoder(response.Body).Decode(&result)
	return response, err
}

// MakeE3DBAccount attempts to create a valid e3db account returning the root client config for the created account and error (if any).
func MakeE3DBAccount(t *testing.T, accounter *accountClient.E3dbAccountClient, accountTag string) (e3dbClients.ClientConfig, *accountClient.CreateAccountResponse, error) {
	var accountClientConfig = e3dbClients.ClientConfig{
		Host: accounter.Host,
	}
	var accountResponse *accountClient.CreateAccountResponse
	// Generate info for creating a new account
	const saltSize = 16
	saltSeed := [saltSize]byte{}
	_, err := rand.Read(saltSeed[:])
	if err != nil {
		t.Errorf("Failed creating salt: %s", err)
		return accountClientConfig, accountResponse, err
	}
	salt := base64.RawURLEncoding.EncodeToString(saltSeed[:])
	publicKey, _, err := e3db.GenerateKeyPair()
	if err != nil {
		t.Errorf("Failed generating key pair %s", err)
		return accountClientConfig, accountResponse, err
	}
	backupPublicKey, _, err := e3db.GenerateKeyPair()
	if err != nil {
		t.Errorf("Failed generating key pair %s", err)
		return accountClientConfig, accountResponse, err
	}
	createAccountParams := accountClient.CreateAccountRequest{
		Profile: accountClient.Profile{
			Name:               accountTag,
			Email:              fmt.Sprintf("test+%s@test.com", accountTag),
			AuthenticationSalt: salt,
			EncodingSalt:       salt,
			SigningKey: accountClient.EncryptionKey{
				Ed25519: publicKey,
			},
			PaperAuthenticationSalt: salt,
			PaperEncodingSalt:       salt,
			PaperSigningKey: accountClient.EncryptionKey{
				Ed25519: publicKey,
			},
		},
		Account: accountClient.Account{
			Company: "ACME Testing",
			Plan:    "free0",
			PublicKey: accountClient.ClientKey{
				Curve25519: backupPublicKey,
			},
		},
	}
	// Create an account and client for that account using the specified params
	ctx := context.TODO()
	accountResponse, err = accounter.CreateAccount(ctx, createAccountParams)
	if err != nil {
		t.Errorf("Error %s creating account with params %+v\n", err, createAccountParams)
		return accountClientConfig, accountResponse, err
	}
	accountClientConfig.APIKey = accountResponse.Account.Client.APIKeyID
	accountClientConfig.APISecret = accountResponse.Account.Client.APISecretKey
	return accountClientConfig, accountResponse, err
}
