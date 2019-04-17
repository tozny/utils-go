package server

import (
	"net/http"
)

// NoMethodHandler handles HTTP requests if no other method is matched.
type NoMethodHandler interface {
	NoMethod(w http.ResponseWriter, r *http.Request)
}

// GetHandler is an HTTP handler function capable of handling GET requests.
type GetHandler interface {
	Get(w http.ResponseWriter, r *http.Request)
}

// PostHandler is an HTTP handler function capable of handling POST requests.
type PostHandler interface {
	Post(w http.ResponseWriter, r *http.Request)
}

// PutHandler is an HTTP handler function capable of handling PUT requests.
type PutHandler interface {
	Put(w http.ResponseWriter, r *http.Request)
}

// PatchHandler is an HTTP handler function capable of handling PATCH requests.
type PatchHandler interface {
	Patch(w http.ResponseWriter, r *http.Request)
}

// DeleteHandler is an HTTP handler function capable of handling DELETE requests.
type DeleteHandler interface {
	Delete(w http.ResponseWriter, r *http.Request)
}

// RouteMethods routes HTTP requests to corresponding handling functions based on request method
// This allows defining a struct that takes dependencies for a route, and define methods for
// corresponding HTTP methods. The dependencies from the struct are injected into the handling
// function automatically. This method converts a struct with any of Get, Post, Put, Patch, and
// Delete methods to a go http.Handler. If a method is called that is not defined, it will
// return HTTP 405 Method Not Supported, ro the struct can declare a NoHandler method to
// Custom handle the catch-all route.
func RouteMethods(mh interface{}) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if h, ok := mh.(GetHandler); ok {
				h.Get(w, r)
				return
			}
		case http.MethodPost:
			if h, ok := mh.(PostHandler); ok {
				h.Post(w, r)
				return
			}
		case http.MethodPut:
			if h, ok := mh.(PutHandler); ok {
				h.Put(w, r)
				return
			}
		case http.MethodPatch:
			if h, ok := mh.(PatchHandler); ok {
				h.Patch(w, r)
				return
			}
		case http.MethodDelete:
			if h, ok := mh.(DeleteHandler); ok {
				h.Delete(w, r)
				return
			}
		}
		if h, ok := mh.(NoMethodHandler); ok {
			h.NoMethod(w, r)
		} else {
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})
}
