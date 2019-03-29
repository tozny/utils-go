package server

import (
	"fmt"
	"net/http"
	"time"
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

// ServeMux serves a set of endpoints defined in a ServerMux with the defined middleware applied
func ServeMux(httpd *http.ServeMux, mw []Middleware, servicePort string) *http.Server {
	// Create an http server with the provided config
	server := &http.Server{
		Addr: fmt.Sprintf(":%s", servicePort), //host:port for the server to listen on, defaults to localhost
		// Log all requests made to this server
		// Enable CORS support
		// enforce valid external e3db client auth
		Handler:      ApplyMiddleware(mw, httpd),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	// Run http server until program is aborted or service errors.
	panic(fmt.Errorf("Server error: %v", server.ListenAndServe()))
}
