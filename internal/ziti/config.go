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

type OpenZitiServicePolicy struct {
	ID            string
	Name          string
	Type          string
	IdentityRoles []string
	ServiceRoles  []string
	Tags          map[string]string
}

type ListResult[T any] struct {
	Items         []T
	NextPageToken string
}
