package server

import (
	"fmt"
	"strings"

	identityv1 "github.com/agynio/ziti-management/.gen/go/agynio/api/identity/v1"
	zitimanagementv1 "github.com/agynio/ziti-management/.gen/go/agynio/api/ziti_management/v1"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/agynio/ziti-management/internal/store"
	"github.com/agynio/ziti-management/internal/ziti"
)

const maxPort int32 = 65535

func parseUUID(value string) (uuid.UUID, error) {
	if value == "" {
		return uuid.UUID{}, fmt.Errorf("value is empty")
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.UUID{}, err
	}
	return id, nil
}

func fromProtoIdentityType(value identityv1.IdentityType) (store.IdentityType, error) {
	switch value {
	case identityv1.IdentityType_IDENTITY_TYPE_AGENT:
		return store.IdentityTypeAgent, nil
	case identityv1.IdentityType_IDENTITY_TYPE_RUNNER:
		return store.IdentityTypeRunner, nil
	case identityv1.IdentityType_IDENTITY_TYPE_APP:
		return store.IdentityTypeApp, nil
	case identityv1.IdentityType_IDENTITY_TYPE_UNSPECIFIED:
		return store.IdentityTypeUnspecified, fmt.Errorf("identity type unspecified")
	default:
		return store.IdentityTypeUnspecified, fmt.Errorf("unknown identity type %v", value)
	}
}

func fromProtoServiceType(value zitimanagementv1.ServiceType) (store.ServiceType, error) {
	switch value {
	case zitimanagementv1.ServiceType_SERVICE_TYPE_GATEWAY:
		return store.ServiceTypeGateway, nil
	case zitimanagementv1.ServiceType_SERVICE_TYPE_ORCHESTRATOR:
		return store.ServiceTypeOrchestrator, nil
	case zitimanagementv1.ServiceType_SERVICE_TYPE_LLM_PROXY:
		return store.ServiceTypeLLMProxy, nil
	case zitimanagementv1.ServiceType_SERVICE_TYPE_TRACING:
		return store.ServiceTypeTracing, nil
	case zitimanagementv1.ServiceType_SERVICE_TYPE_RUNNERS:
		return store.ServiceTypeRunners, nil
	case zitimanagementv1.ServiceType_SERVICE_TYPE_EGRESS_GATEWAY:
		return store.ServiceTypeEgressGateway, nil
	case zitimanagementv1.ServiceType_SERVICE_TYPE_UNSPECIFIED:
		return store.ServiceTypeUnspecified, fmt.Errorf("service type unspecified")
	default:
		return store.ServiceTypeUnspecified, fmt.Errorf("unknown service type %v", value)
	}
}

func fromProtoServicePolicyType(value zitimanagementv1.ServicePolicyType) (string, error) {
	switch value {
	case zitimanagementv1.ServicePolicyType_SERVICE_POLICY_TYPE_BIND:
		return "Bind", nil
	case zitimanagementv1.ServicePolicyType_SERVICE_POLICY_TYPE_DIAL:
		return "Dial", nil
	case zitimanagementv1.ServicePolicyType_SERVICE_POLICY_TYPE_UNSPECIFIED:
		return "", fmt.Errorf("service policy type unspecified")
	default:
		return "", fmt.Errorf("unknown service policy type %v", value)
	}
}

func fromProtoHostV1Config(value *zitimanagementv1.HostV1Config) (*ziti.HostV1ConfigData, error) {
	if value == nil {
		return nil, nil
	}
	protocol := strings.TrimSpace(value.GetProtocol())
	if protocol == "" && !value.GetForwardProtocol() {
		return nil, fmt.Errorf("protocol is required")
	}
	address := strings.TrimSpace(value.GetAddress())
	if address == "" && !value.GetForwardAddress() {
		return nil, fmt.Errorf("address is required")
	}
	port := value.GetPort()
	if !value.GetForwardPort() && (port <= 0 || port > maxPort) {
		return nil, fmt.Errorf("port must be between 1 and %d", maxPort)
	}
	if value.GetForwardPort() && port > maxPort {
		return nil, fmt.Errorf("port must be between 1 and %d", maxPort)
	}
	if value.GetForwardProtocol() && len(value.GetAllowedProtocols()) == 0 {
		return nil, fmt.Errorf("allowed_protocols is required when forward_protocol is true")
	}
	if value.GetForwardAddress() && len(value.GetAllowedAddresses()) == 0 {
		return nil, fmt.Errorf("allowed_addresses is required when forward_address is true")
	}
	if value.GetForwardPort() && len(value.GetAllowedPortRanges()) == 0 {
		return nil, fmt.Errorf("allowed_port_ranges is required when forward_port is true")
	}
	allowedProtocols, err := cleanOptionalStrings(value.GetAllowedProtocols(), "allowed_protocols")
	if err != nil {
		return nil, err
	}
	allowedAddresses, err := cleanOptionalStrings(value.GetAllowedAddresses(), "allowed_addresses")
	if err != nil {
		return nil, err
	}
	allowedPortRanges, err := fromProtoOptionalPortRanges(value.GetAllowedPortRanges(), "allowed_port_ranges")
	if err != nil {
		return nil, err
	}

	return &ziti.HostV1ConfigData{
		Protocol:          protocol,
		Address:           address,
		Port:              port,
		ForwardProtocol:   value.GetForwardProtocol(),
		ForwardAddress:    value.GetForwardAddress(),
		ForwardPort:       value.GetForwardPort(),
		AllowedProtocols:  allowedProtocols,
		AllowedAddresses:  allowedAddresses,
		AllowedPortRanges: allowedPortRanges,
	}, nil
}

func fromProtoInterceptV1Config(value *zitimanagementv1.InterceptV1Config) (*ziti.InterceptV1ConfigData, error) {
	if value == nil {
		return nil, nil
	}
	protocols := value.GetProtocols()
	if len(protocols) == 0 {
		return nil, fmt.Errorf("protocols is required")
	}
	cleanProtocols := make([]string, len(protocols))
	for i, protocol := range protocols {
		cleaned := strings.TrimSpace(protocol)
		if cleaned == "" {
			return nil, fmt.Errorf("protocols[%d] is empty", i)
		}
		cleanProtocols[i] = cleaned
	}

	addresses := value.GetAddresses()
	if len(addresses) == 0 {
		return nil, fmt.Errorf("addresses is required")
	}
	cleanAddresses := make([]string, len(addresses))
	for i, address := range addresses {
		cleaned := strings.TrimSpace(address)
		if cleaned == "" {
			return nil, fmt.Errorf("addresses[%d] is empty", i)
		}
		cleanAddresses[i] = cleaned
	}

	convertedRanges, err := fromProtoPortRanges(value.GetPortRanges(), "port_ranges")
	if err != nil {
		return nil, err
	}

	return &ziti.InterceptV1ConfigData{
		Protocols:  cleanProtocols,
		Addresses:  cleanAddresses,
		PortRanges: convertedRanges,
	}, nil
}

func cleanOptionalStrings(values []string, fieldName string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}
	cleanValues := make([]string, len(values))
	for i, value := range values {
		cleaned := strings.TrimSpace(value)
		if cleaned == "" {
			return nil, fmt.Errorf("%s[%d] is empty", fieldName, i)
		}
		cleanValues[i] = cleaned
	}
	return cleanValues, nil
}

func fromProtoPortRanges(portRanges []*zitimanagementv1.PortRange, fieldName string) ([]ziti.PortRangeData, error) {
	if len(portRanges) == 0 {
		return nil, fmt.Errorf("%s is required", fieldName)
	}
	return fromProtoOptionalPortRanges(portRanges, fieldName)
}

func fromProtoOptionalPortRanges(portRanges []*zitimanagementv1.PortRange, fieldName string) ([]ziti.PortRangeData, error) {
	if len(portRanges) == 0 {
		return nil, nil
	}
	convertedRanges := make([]ziti.PortRangeData, len(portRanges))
	for i, portRange := range portRanges {
		low := portRange.GetLow()
		high := portRange.GetHigh()
		if low <= 0 || high <= 0 || low > maxPort || high > maxPort {
			return nil, fmt.Errorf("%s[%d] must be between 1 and %d", fieldName, i, maxPort)
		}
		if high < low {
			return nil, fmt.Errorf("%s[%d] high must be >= low", fieldName, i)
		}
		convertedRanges[i] = ziti.PortRangeData{Low: low, High: high}
	}
	return convertedRanges, nil
}

func toProtoIdentityType(value store.IdentityType) (identityv1.IdentityType, error) {
	switch value {
	case store.IdentityTypeAgent:
		return identityv1.IdentityType_IDENTITY_TYPE_AGENT, nil
	case store.IdentityTypeRunner:
		return identityv1.IdentityType_IDENTITY_TYPE_RUNNER, nil
	case store.IdentityTypeApp:
		return identityv1.IdentityType_IDENTITY_TYPE_APP, nil
	case store.IdentityTypeUnspecified:
		return identityv1.IdentityType_IDENTITY_TYPE_UNSPECIFIED, nil
	default:
		return identityv1.IdentityType_IDENTITY_TYPE_UNSPECIFIED, fmt.Errorf("unknown identity type %d", value)
	}
}

func toProtoManagedIdentity(identity store.ManagedIdentity) (*zitimanagementv1.ManagedIdentity, error) {
	identityType, err := toProtoIdentityType(identity.IdentityType)
	if err != nil {
		return nil, err
	}
	zitiServiceID := ""
	if identity.ZitiServiceID != nil {
		zitiServiceID = *identity.ZitiServiceID
	}
	protoIdentity := &zitimanagementv1.ManagedIdentity{
		ZitiIdentityId: identity.ZitiIdentityID,
		IdentityId:     identity.IdentityID.String(),
		IdentityType:   identityType,
		ZitiServiceId:  zitiServiceID,
		CreatedAt:      timestamppb.New(identity.CreatedAt),
	}
	return protoIdentity, nil
}
