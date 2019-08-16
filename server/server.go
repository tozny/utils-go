package server

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

var (
	// SupportedAuthTypes is a whitelist of supported authentication types. Deafult: Bearer
	SupportedAuthTypes = [...]string{"Bearer"}
	// ErrorInvalidAuthorizationHeader is a static error for invalid authorization
	ErrorInvalidAuthorizationHeader = errors.New("InvalidAuthorizationHeader")
	// ErrorUnsupportedAuthorizationType is a static error returned if the auth type is not in the whitelist
	ErrorUnsupportedAuthorizationType = fmt.Errorf("UnsupportedAuthorizationType, supported types are %v", SupportedAuthTypes)
	// ErrorInvalidAuthToken is a static error returned when authentication fails
	ErrorInvalidAuthToken = errors.New("InvalidAuthToken")
	// ErrorInvalidAuthentication is a static error returned when request authentication fails
	ErrorInvalidAuthentication = errors.New("Invalid authentication attempt")
)

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
