package mock

import (
	"bytes"
	"net/http"
)

// ResponseWriter used by an HTTP handler to construct an HTTP response.
// This is the mock implementation
type ResponseWriter struct {
	Buffer       bytes.Buffer
	WriterStatus int
}

// Header returns the header map that will be sent by
func (m *ResponseWriter) Header() (h http.Header) {
	return http.Header{}
}

// Write writes the data to the connection as part of an HTTP reply.
func (m *ResponseWriter) Write(p []byte) (n int, err error) {
	m.Buffer.Write(p)
	return len(p), nil
}

// WriteString write string to the request
func (m *ResponseWriter) WriteString(s string) (n int, err error) {
	m.Buffer.WriteString(s)
	return len(s), nil
}

// WriteHeader sends an HTTP response header with the provided
// status code.
func (m *ResponseWriter) WriteHeader(i int) {
	m.WriterStatus = i
}

// GetBody Get the body of the request as string
func (m *ResponseWriter) GetBody() string {
	return m.Buffer.String()
}
