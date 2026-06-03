package ziti

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
