// Package utils provides common utilities used by search service components.
package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tozny/e3db-clients-go/authClient"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

const (
	ToznyClientIDHeader                = "X-TOZNY-CLIENT-ID"
	ToznyOpenAuthenticationTokenHeader = "X-TOZNY-TOT"
	HealthCheckPathSuffix              = "/healthcheck"
	ServiceCheckPathSuffix             = "/servicecheck"
)

var (
	SupportedAuthTypes                = [...]string{"Bearer"}
	ErrorInvalidAuthorizationHeader   = errors.New("InvalidAuthorizationHeader")
	ErrorUnsupportedAuthorizationType = errors.New(fmt.Sprintf("UnsupportedAuthorizationType, supported types are %v", SupportedAuthTypes))
	ErrorInvalidAuthToken             = errors.New("InvalidAuthToken")
	DefaultCORSHeaders                = []http.Header{
		// https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS#The_HTTP_response_headers
		map[string][]string{
			"Access-Control-Allow-Origin":      []string{"*"},
			"Access-Control-Allow-Methods":     []string{"*, GET, POST, DELETE, PUT, OPTIONS, HEAD"}, // Because to Firefox * does not mean all.
			"Access-Control-Allow-Headers":     []string{"Authorization, Content-Type, *"},           // Because to Firefox * does not mean all.
			"Access-Control-Allow-Credentials": []string{"true"},
			"Access-Control-Max-Age":           []string{"86400"},
		},
	}
)

// JsonLoggingHandler wraps an HTTP handler and logs
// the request and de-serialized JSON body.
func JsonLoggingHandler(h http.Handler, logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var requestBody interface{}
		json.NewDecoder(r.Body).Decode(&requestBody)
		logger.Print(map[string]interface{}{
			"request_method":    r.Method,
			"request_uri":       r.RequestURI,
			"requester_address": r.RemoteAddr,
			"requester_host":    r.Host,
			"request_body":      requestBody,
		})
		// Repopulate body with the data read
		jsonBytes := new(bytes.Buffer)
		json.NewEncoder(jsonBytes).Encode(requestBody)
		r.Body = ioutil.NopCloser(jsonBytes)
		h.ServeHTTP(w, r)
	})
}

// ExtractBearerToken attempts to extract an Oauth bearer token
// from the provided request, returning extracted token and error (if any)
func ExtractBearerToken(r *http.Request) (string, error) {
	var authToken string
	authHeader := r.Header.Get("Authorization")
	authParts := strings.Split(authHeader, " ")
	if len(authParts) != 2 {
		return authToken, ErrorInvalidAuthorizationHeader
	}
	authType := authParts[0]
	var invalidAuthType = true
	for _, supportedType := range SupportedAuthTypes {
		if authType == supportedType {
			invalidAuthType = false
			break
		}
	}
	if invalidAuthType {
		return authToken, ErrorUnsupportedAuthorizationType
	}
	authToken = authParts[1]
	return authToken, nil
}

// HandleError is a generic error handler for responding with the given status and error
// using the provided ResponseWriter.
func HandleError(w http.ResponseWriter, statusCode int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// HandleOptionsRequest is a generic handler for responding 200 OK for an HTTP Options request.
func HandleOptionsRequest(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
}

// E3dbAuthHandler provides http middleware for enforcing requests as coming from e3db
// authenticated entities (either external or internal clients) for any request with a path
// not ending in `HealthCheckPathSuffix` or `ServiceCheckPathSuffix`
func E3dbAuthHandler(h http.Handler, e3dbAuth authClient.E3dbAuthClient, privateService bool, logger *log.Logger) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check to see if this request is a health or service check requests
		requestPath := r.URL.Path
		isMonitoringRequest := strings.HasSuffix(requestPath, HealthCheckPathSuffix) || strings.HasSuffix(requestPath, ServiceCheckPathSuffix)
		if isMonitoringRequest {
			// NoOp authentication, continue processing request
			h.ServeHTTP(w, r)
			return
		}
		token, err := ExtractBearerToken(r)
		if err != nil {
			logger.Printf("E3dbAuthHandler: error extracting bearer token %s\n", err)
			HandleError(w, http.StatusUnauthorized, err)
			return
		}
		ctx := context.Background()
		validateParams := authClient.ValidateTokenRequest{
			Token:    token,
			Internal: privateService,
		}
		validateTokenResponse, err := e3dbAuth.ValidateToken(ctx, validateParams)
		if err != nil || !validateTokenResponse.Valid {
			logger.Printf("E3dbAuthHandler: error validating token %s %+v\n", err, validateTokenResponse)
			HandleError(w, http.StatusUnauthorized, ErrorInvalidAuthToken)
			return
		}
		// Add the clients id and token to the request headers
		r.Header.Set(ToznyClientIDHeader, validateTokenResponse.ClientId)
		r.Header.Set(ToznyOpenAuthenticationTokenHeader, token)
		// Authenticated, continue processing request
		h.ServeHTTP(w, r)
	})
}

// CORSHandler provides http middleware for allowing cross origin requests by
// decorating the request with the provided CORS headers and returning default 200 OK for any options requests
func CORSHandler(h http.Handler, corsHeaders []http.Header) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, corsHeader := range corsHeaders {
			for key, values := range corsHeader {
				for _, value := range values {
					w.Header().Set(key, value)
				}
			}
		}
		switch r.Method {
		case http.MethodOptions:
			HandleOptionsRequest(w)
			return
		}
		h.ServeHTTP(w, r)
	})
}
