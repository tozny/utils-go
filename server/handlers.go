package server

import (
	"fmt"
	"net/http"
)

// HandleOptionsRequest is a generic handler for responding 200 OK for an HTTP Options request.
func HandleOptionsRequest(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
}

// HealthCheckHandler handles health check requests to the specified service
// returning 200 if the service is up, otherwise nothing
func HealthCheckHandler(serviceName string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("%s service is up.\n", serviceName)))
	})
}
