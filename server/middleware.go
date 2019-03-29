package server

import (
	"github.com/tozny/e3db-clients-go/authClient"
	"github.com/tozny/utils-go"
	"log"
	"net/http"
)

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
