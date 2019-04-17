package server

import (
	"github.com/tozny/e3db-clients-go/authClient"
	"github.com/tozny/utils-go"
	"log"
	"net/http"
	"strings"
)

// Middleware is a function that decorates an http.Handler
type Middleware func(handler http.Handler) http.Handler

// ApplyMiddleware creates a final http Handler with a slice of middleware functions applied
func ApplyMiddleware(handlers []Middleware, final http.Handler) http.Handler {
	for _, handler := range handlers {
		final = handler(final)
	}
	return final
}

// JSONLoggingMiddleware converts the utils JsonLogger into Middleware
func JSONLoggingMiddleware(logger *log.Logger) Middleware {
	return func(h http.Handler) http.Handler {
		return utils.JsonLoggingHandler(h, logger)
	}
}

// CORSMiddleware converts the utils CORSHandler into Middleware
func CORSMiddleware(corsHeaders []http.Header) Middleware {
	return func(h http.Handler) http.Handler {
		return utils.CORSHandler(h, corsHeaders)
	}
}

// AuthMiddleware converts utils E3dbAuthHandler into Middleware
func AuthMiddleware(auth authClient.E3dbAuthClient, privateService bool, logger *log.Logger) Middleware {
	return func(h http.Handler) http.Handler {
		return utils.E3dbAuthHandler(h, auth, privateService, logger)
	}
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
