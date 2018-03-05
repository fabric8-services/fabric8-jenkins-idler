package tenant

import "time"

// InfoList defines list of informations about a tenant
type InfoList struct {
	Data []InfoData
	Meta struct {
		TotalCount int
	}
	Errors []Error `json:"errors"`
}

// Info defines a single tanent information
type Info struct {
	Data   InfoData
	Errors []Error `json:"errors"`
}

// Error defines an http error
type Error struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// InfoData is Data used in Info and InfoList
type InfoData struct {
	Attributes Attributes
	ID         string
	Type       string
}

// Attributes is Attributes used in InfoData
type Attributes struct {
	CreatedAt  time.Time `json:"created-at"`
	Email      string
	Namespaces []Namespace
}

// Namespace provides an additional qualification to a resource name.
// This is helpful when multiple teams are using the same cluster and there is a potential of name collision.
// It can be as a virtual wall between multiple clusters.
type Namespace struct {
	ClusterURL string `json:"cluster-url"`
	Name       string
	State      string
	Type       string
}
