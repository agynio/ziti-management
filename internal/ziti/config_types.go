package ziti

import (
	_ "embed"
	"encoding/json"
)

const (
	hostV1ConfigType      = "host.v1"
	interceptV1ConfigType = "intercept.v1"
)

//go:embed schemas/host.v1.json
var hostV1SchemaRaw []byte

//go:embed schemas/intercept.v1.json
var interceptV1SchemaRaw []byte

func configTypeSchema(name string) (json.RawMessage, bool) {
	switch name {
	case hostV1ConfigType:
		return json.RawMessage(hostV1SchemaRaw), true
	case interceptV1ConfigType:
		return json.RawMessage(interceptV1SchemaRaw), true
	default:
		return nil, false
	}
}
