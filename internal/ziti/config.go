package ziti

import "time"

const (
	hostV1ConfigTypeID      = "NH5p4FpGR"
	interceptV1ConfigTypeID = "g7cIWbcGg"
)

type HostV1ConfigData struct {
	Protocol          string
	Address           string
	Port              int32
	ForwardProtocol   bool
	ForwardAddress    bool
	ForwardPort       bool
	AllowedProtocols  []string
	AllowedAddresses  []string
	AllowedPortRanges []PortRangeData
}

type InterceptV1ConfigData struct {
	Protocols  []string
	Addresses  []string
	PortRanges []PortRangeData
}

type PortRangeData struct {
	Low  int32
	High int32
}

type IdentityLiveness struct {
	EnrollmentPending       bool
	HasEdgeRouterConnection bool
}

type EnrollmentJWT struct {
	Token     string
	TokenID   string
	ExpiresAt time.Time
}

type OpenZitiIdentity struct {
	ID             string
	Name           string
	RoleAttributes []string
	Tags           map[string]string
}

type OpenZitiService struct {
	ID             string
	Name           string
	RoleAttributes []string
	Tags           map[string]string
}

type Service struct {
	ID                string
	Name              string
	RoleAttributes    []string
	HostV1Config      *HostV1ConfigData
	InterceptV1Config *InterceptV1ConfigData
}

type ServiceListFilter struct {
	Name           string
	NamePrefix     string
	RoleAttributes []string
	PageSize       int32
	PageToken      string
}

type ServiceListResult struct {
	Services      []Service
	NextPageToken string
}

type OpenZitiServicePolicy struct {
	ID            string
	Name          string
	Type          string
	IdentityRoles []string
	ServiceRoles  []string
	Tags          map[string]string
}

type ServicePolicy struct {
	ID            string
	Name          string
	Type          string
	IdentityRoles []string
	ServiceRoles  []string
}

type ServicePolicyListFilter struct {
	Name          string
	NamePrefix    string
	Type          string
	IdentityRoles []string
	ServiceRoles  []string
	PageSize      int32
	PageToken     string
}

type ServicePolicyListResult struct {
	ServicePolicies []ServicePolicy
	NextPageToken   string
}

type ListResult[T any] struct {
	Items         []T
	NextPageToken string
}
