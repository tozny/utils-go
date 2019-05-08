// Package test provides helper functions and common structs for use in tests across tozny golang repositories.
package test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
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

// AssertRespStatus asserts that the response status of r is a specific value.
func AssertRespStatus(t *testing.T, context string, r *http.Response, code int) {
	if r.StatusCode != code {
		t.Fatalf("%s: Unexpected response status: %d. Expected %d", context, r.StatusCode, code)
	}
}

// UnmarshalJSONRequest decodes the body from a response to the provided object.
func UnmarshalJSONRequest(t *testing.T, r *http.Response, obj interface{}) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatal("Error reading body")
	}
	err = json.Unmarshal(body, obj)
	if err != nil {
		t.Fatalf("Unable to decode response %s to object %T", string(body), obj)
	}
}

// DecodeResponseString decodes the body from a response to a string and returns it.
func DecodeResponseString(t *testing.T, r *http.Response) string {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		t.Fatal("Error reading body")
	}
	return string(body)
}
