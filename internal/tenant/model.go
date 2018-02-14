package tenant

import "time"

type TenantInfoList struct {
	Data []TenantInfoData
	Meta struct {
		TotalCount int
	}
	Errors []Error `json:"errors"`
}
type TenantInfo struct {
	Data   TenantInfoData
	Errors []Error `json:"errors"`
}

type Error struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

type TenantInfoData struct {
	Attributes Attributes
	Id         string
	Type       string
}

type Attributes struct {
	CreatedAt  time.Time `json:"created-at"`
	Email      string
	Namespaces []Namespace
}

type Namespace struct {
	ClusterURL string `json:"cluster-url"`
	Name       string
	State      string
	Type       string
}
