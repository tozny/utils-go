package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// ReqToObject un-marshals a request object body JSON into an object
func ReqToObject(r *http.Request, obj interface{}) error {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(body, obj); err != nil {
		return err
	}
	return nil
}

// WriteObjectResponse marshals an object into the response body and sets
// JSON content type headers
func WriteObjectResponse(obj interface{}, w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(obj); err != nil {
		return err
	}
	return nil
}

// HandleError is a generic error handler for responding with the given status and error
// using the provided ResponseWriter.
func HandleError(w http.ResponseWriter, statusCode int, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}
