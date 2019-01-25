package configuration

import "net/http"

// HTTPClientOption options passed to the HTTP Client
type HTTPClientOption func(client *http.Client)
