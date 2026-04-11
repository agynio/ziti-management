package ziti

type HostV1ConfigData struct {
	Protocol string
	Address  string
	Port     int32
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
