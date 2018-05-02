package tenant

import "time"

// InfoList defines list of informations about a tenant.
type InfoList struct {
	Data []InfoData
	Meta struct {
		TotalCount int
	}
	Errors []Error `json:"errors"`
}

// Info defines a single tenant information.
type Info struct {
	Data   InfoData
	Errors []Error `json:"errors"`
}

// Error defines an HTTP error.
type Error struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// InfoData is Data used in Info and InfoList.
type InfoData struct {
	Attributes Attributes
	ID         string
	Type       string
}

// Attributes provides information such as when the build was created, namespace and email.
type Attributes struct {
	CreatedAt  time.Time `json:"created-at"`
	Email      string
	Namespaces []Namespace
}

// Namespace of the build.
// It defines the space within each name must be unique.
type Namespace struct {
	ClusterURL               string `json:"cluster-url"`
	Name                     string
	State                    string
	Type                     string
	ClusterCapacityExhausted bool `json:"cluster-capacity-exhausted"`
}
