package common

import (
	"net/http"
	"net/http/httptest"
)

func MockServer(b []byte) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer req.Body.Close()
		w.Write(b)
		w.Header().Set("Content-Type", "application/json")
	}))
}
