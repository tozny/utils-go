package logging

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

// ErrorLog is a generic error log containing a descriptive message and the error
type ErrorLog struct {
	Message string `json:"msg"`
	Error   string `json:"err"`
}

// RequestErrorLog is an error log from an http request.
type RequestErrorLog struct {
	ErrorLog
	Method string      `json:"req_method"`
	URL    string      `json:"req_url"`
	Body   interface{} `json:"req_body"` // Convenience so a decoded object can be passed in, or req.Body can be decoded
	Header string      `json:"req_header"`
}

// NewRequestErrorLog constructs a RequestErrorLog doing the work of breaking up
// an http request into its logical parts and converting them into marshal-able types.
// if the request body has not been read, this will attempt to read and re-populate the body,
// else provide the decodedBody so it can be logged
func NewRequestErrorLog(msg string, err error, req *http.Request, decodedBody interface{}) *RequestErrorLog {
	var bodyLog interface{}
	if decodedBody != nil {
		bodyLog = decodedBody
	} else {
		var bodyBytes []byte
		if req.Body != nil {
			bodyBytes, _ = ioutil.ReadAll(req.Body)
		}
		req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyBytes))
		bodyLog = string(bodyBytes)
	}
	return &RequestErrorLog{
		ErrorLog: ErrorLog{
			Message: msg,
			Error:   err.Error(),
		},
		Method: req.Method,
		URL:    req.URL.String(),
		Body:   bodyLog,
		Header: fmt.Sprintf("%+v", req.Header),
	}
}

// NewErrorLog constructs an Error log, turning error into a string.
func NewErrorLog(msg string, err error) *ErrorLog {
	return &ErrorLog{
		Message: msg,
		Error:   err.Error(),
	}
}
