package ziti

import (
	_ "embed"
	"encoding/json"
)

const (
	hostV1ConfigType        = "host.v1"
	interceptV1ConfigType   = "intercept.v1"
	hostV1ConfigTypeID      = "NH5p4FpGR"
	interceptV1ConfigTypeID = "g7cIWbcGg"
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

func knownConfigTypeID(name string) (string, bool) {
	switch name {
	case hostV1ConfigType:
		return hostV1ConfigTypeID, true
	case interceptV1ConfigType:
		return interceptV1ConfigTypeID, true
	default:
		return "", false
	}
}

func normalizeConfigTypeID(name, id string) string {
	if knownID, ok := knownConfigTypeID(name); ok {
		if id == "" || id == name {
			return knownID
		}
	}
	return id
}
