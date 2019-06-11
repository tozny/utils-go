package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/tozny/utils-go/logging"
)

const (
	// ToznyClientIDHeader is the headers key containing a client ID
	ToznyClientIDHeader = "X-TOZNY-CLIENT-ID"
	// ToznyOpenAuthenticationTokenHeader is the header key contaning a Tot
	ToznyOpenAuthenticationTokenHeader = "X-TOZNY-TOT"
	// HealthCheckPathSuffix is a centrally defined health check path.
	HealthCheckPathSuffix = "/healthcheck"
	// ServiceCheckPathSuffix is a centrally defined service check path.
	ServiceCheckPathSuffix = "/servicecheck"
)

var (
	// DefaultCORSHeaders is a full set of CORS headers for use in the CORS middleware
	DefaultCORSHeaders = []http.Header{
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

// Middleware is a function that decorates an http.Handler
//
// The decorator function can determine whether to pass the request on to the
// next handler in the chain by calling the ServeHTTP method on the handler. If
// the middleware should pass additional information along with the request,
// context is available on the request object. Add a value to the context.
type Middleware func(http.Handler) http.Handler

// MiddlewareFunc is an adapter to allow the use of ordinary functions as
// Middleware. If f is a function with the appropriate signature MiddlewareFunc(f)
// is a Middleware that calls f.
func MiddlewareFunc(f func(http.Handler, http.ResponseWriter, *http.Request)) Middleware {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			f(h, w, r)
		})
	}
}

// ApplyMiddleware decorates an http.Handler with all passed middleware.
func ApplyMiddleware(handler http.Handler, middleware ...Middleware) http.Handler {
	for _, decorator := range middleware {
		handler = decorator(handler)
	}
	return handler
}

// DecorateHandlerFunc adapts ordinary http handler functions to http handlers
// decorated with middleware.
func DecorateHandlerFunc(f func(http.ResponseWriter, *http.Request), middleware ...Middleware) http.Handler {
	return ApplyMiddleware(http.HandlerFunc(f), middleware...)
}

// JSONLoggingMiddleware wraps an HTTP handler and logs
// the request and de-serialized JSON body.
func JSONLoggingMiddleware(logger logging.Logger) Middleware {
	return MiddlewareFunc(func(h http.Handler, w http.ResponseWriter, r *http.Request) {
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

// CORSMiddleware provides http middleware for allowing cross origin requests by
// decorating the request with the provided CORS headers and returning default 200 OK for any options requests
func CORSMiddleware(corsHeaders []http.Header) Middleware {
	return MiddlewareFunc(func(h http.Handler, w http.ResponseWriter, r *http.Request) {
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

// A E3DBTokenAuthenticator provides the ability to authenticate
// an E3DB entity using an Oauth2 bearer token.
type E3DBTokenAuthenticator interface {
	// AuthenticateE3DBClient validates the provided token belongs to
	// an internal OR external e3db client,
	// returning the clientID and validity of the provided token, and error (if any).
	AuthenticateE3DBClient(ctx context.Context, token string, internal bool) (clientID string, valid bool, err error)
}

// AuthMiddleware provides http middleware for enforcing requests as coming from e3db
// authenticated entities (either external or internal clients) for any request with a path
// not ending in `HealthCheckPathSuffix` or `ServiceCheckPathSuffix`
func AuthMiddleware(auth E3DBTokenAuthenticator, privateService bool, logger *log.Logger) Middleware {
	return MiddlewareFunc(func(h http.Handler, w http.ResponseWriter, r *http.Request) {
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
		clientID, valid, err := auth.AuthenticateE3DBClient(ctx, token, privateService)
		if err != nil || !valid {
			logger.Printf("E3dbAuthHandler: error validating token %s\n", err)
			HandleError(w, http.StatusUnauthorized, ErrorInvalidAuthToken)
			return
		}
		// Add the clients id and token to the request headers
		r.Header.Set(ToznyClientIDHeader, clientID)
		r.Header.Set(ToznyOpenAuthenticationTokenHeader, token)
		// Authenticated, continue processing request
		h.ServeHTTP(w, r)
	})
}

// TrimSlash is middleware to trim trailing slashes from request paths for usability. Without this
// requests to example.com/path works and example.com/path/ fails miserably. This makes them work the same.
func TrimSlash(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.URL.Path = strings.TrimSuffix(r.URL.Path, "/")
		// Call the next handler, which can be another middleware in the chain, or the final handler.
		h.ServeHTTP(w, r)
	})
}
