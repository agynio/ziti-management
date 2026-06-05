package ziti

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/openziti/edge-api/rest_management_api_client/config"
	"github.com/openziti/edge-api/rest_management_api_client/service"
	"github.com/openziti/edge-api/rest_model"
)

const debugListLimit int64 = 100

type DebugServiceState struct {
	ServiceID       string
	ServiceName     string
	RoleAttributes  []string
	Configs         []DebugConfig
	ServicePolicies []DebugServicePolicy
	Terminators     []DebugTerminator
}

type DebugConfig struct {
	ID             string
	Name           string
	ConfigTypeID   string
	ConfigTypeName string
	JSON           string
}

type DebugServicePolicy struct {
	ID            string
	Name          string
	Type          string
	IdentityRoles []string
	ServiceRoles  []string
}

type DebugTerminator struct {
	ID          string
	Identity    string
	RouterID    string
	RouterName  string
	Precedence  string
	Cost        int32
	DynamicCost int32
	Binding     string
	Address     string
}

func (c *Client) DebugServiceState(ctx context.Context, serviceID, serviceName string) (*DebugServiceState, error) {
	resolvedServiceID := serviceID
	if resolvedServiceID == "" {
		id, err := c.lookupServiceID(ctx, serviceName)
		if err != nil {
			return nil, err
		}
		resolvedServiceID = id
	}

	serviceDetail, err := c.detailService(ctx, resolvedServiceID)
	if err != nil {
		return nil, err
	}
	configs, err := c.debugServiceConfigs(ctx, resolvedServiceID)
	if err != nil {
		return nil, err
	}
	policies, err := c.debugServicePolicies(ctx, resolvedServiceID)
	if err != nil {
		return nil, err
	}
	terminators, err := c.debugServiceTerminators(ctx, resolvedServiceID)
	if err != nil {
		return nil, err
	}

	return &DebugServiceState{
		ServiceID:       stringValue(serviceDetail.ID),
		ServiceName:     stringValue(serviceDetail.Name),
		RoleAttributes:  attributesToStrings(serviceDetail.RoleAttributes),
		Configs:         configs,
		ServicePolicies: policies,
		Terminators:     terminators,
	}, nil
}

func (c *Client) lookupServiceID(ctx context.Context, serviceName string) (string, error) {
	if serviceName == "" {
		return "", errors.New("service name is empty")
	}
	filter := fmt.Sprintf("name=%s", strconv.Quote(serviceName))
	params := service.NewListServicesParamsWithContext(ctx)
	params.Filter = &filter
	params.Limit = int64Ptr(debugListLimit)

	var listed *service.ListServicesOK
	err := c.withReauth(func() error {
		var callErr error
		serviceClient := c.serviceClient()
		listed, callErr = serviceClient.ListServices(params, nil)
		return callErr
	})
	if err != nil {
		return "", fmt.Errorf("list ziti services: %w", err)
	}
	if listed == nil || listed.Payload == nil {
		return "", errors.New("list ziti services response missing payload")
	}
	if len(listed.Payload.Data) == 0 {
		return "", ErrServiceNotFound
	}
	if len(listed.Payload.Data) > 1 {
		return "", fmt.Errorf("multiple ziti services matched name %q", serviceName)
	}
	return requiredBaseID(listed.Payload.Data[0], "service")
}

func (c *Client) detailService(ctx context.Context, serviceID string) (*rest_model.ServiceDetail, error) {
	params := service.NewDetailServiceParamsWithContext(ctx)
	params.ID = serviceID
	var detailed *service.DetailServiceOK
	err := c.withReauth(func() error {
		var callErr error
		serviceClient := c.serviceClient()
		detailed, callErr = serviceClient.DetailService(params, nil)
		return callErr
	})
	if err != nil {
		var notFound *service.DetailServiceNotFound
		if errors.As(err, &notFound) {
			return nil, ErrServiceNotFound
		}
		return nil, fmt.Errorf("detail ziti service: %w", err)
	}
	if detailed == nil || detailed.Payload == nil || detailed.Payload.Data == nil {
		return nil, errors.New("detail ziti service response missing data")
	}
	return detailed.Payload.Data, nil
}

func (c *Client) debugServiceConfigs(ctx context.Context, serviceID string) ([]DebugConfig, error) {
	params := service.NewListServiceConfigParamsWithContext(ctx)
	params.ID = serviceID
	params.Limit = int64Ptr(debugListLimit)
	var listed *service.ListServiceConfigOK
	err := c.withReauth(func() error {
		var callErr error
		serviceClient := c.serviceClient()
		listed, callErr = serviceClient.ListServiceConfig(params, nil)
		return callErr
	})
	if err != nil {
		return nil, fmt.Errorf("list ziti service configs: %w", err)
	}
	if listed == nil || listed.Payload == nil {
		return nil, errors.New("list ziti service configs response missing payload")
	}
	configs := make([]DebugConfig, 0, len(listed.Payload.Data))
	for _, configDetail := range listed.Payload.Data {
		configID, err := requiredConfigID(configDetail)
		if err != nil {
			return nil, err
		}
		configDetail, err := c.detailConfig(ctx, configID)
		if err != nil {
			return nil, err
		}
		configs = append(configs, configDetail)
	}
	return configs, nil
}

func (c *Client) detailConfig(ctx context.Context, configID string) (DebugConfig, error) {
	params := config.NewDetailConfigParamsWithContext(ctx)
	params.ID = configID
	var detailed *config.DetailConfigOK
	err := c.withReauth(func() error {
		var callErr error
		configClient := c.configClient()
		detailed, callErr = configClient.DetailConfig(params, nil)
		return callErr
	})
	if err != nil {
		return DebugConfig{}, fmt.Errorf("detail ziti config: %w", err)
	}
	if detailed == nil || detailed.Payload == nil || detailed.Payload.Data == nil {
		return DebugConfig{}, errors.New("detail ziti config response missing data")
	}
	configDetail := detailed.Payload.Data
	jsonData, err := json.Marshal(configDetail.Data)
	if err != nil {
		return DebugConfig{}, fmt.Errorf("marshal ziti config data: %w", err)
	}
	configTypeName := ""
	if configDetail.ConfigType != nil {
		configTypeName = configDetail.ConfigType.Name
	}
	return DebugConfig{
		ID:             stringValue(configDetail.ID),
		Name:           stringValue(configDetail.Name),
		ConfigTypeID:   stringValue(configDetail.ConfigTypeID),
		ConfigTypeName: configTypeName,
		JSON:           string(jsonData),
	}, nil
}

func (c *Client) debugServicePolicies(ctx context.Context, serviceID string) ([]DebugServicePolicy, error) {
	params := service.NewListServiceServicePoliciesParamsWithContext(ctx)
	params.ID = serviceID
	params.Limit = int64Ptr(debugListLimit)
	var listed *service.ListServiceServicePoliciesOK
	err := c.withReauth(func() error {
		var callErr error
		serviceClient := c.serviceClient()
		listed, callErr = serviceClient.ListServiceServicePolicies(params, nil)
		return callErr
	})
	if err != nil {
		return nil, fmt.Errorf("list ziti service policies: %w", err)
	}
	if listed == nil || listed.Payload == nil {
		return nil, errors.New("list ziti service policies response missing payload")
	}
	policies := make([]DebugServicePolicy, 0, len(listed.Payload.Data))
	for _, policy := range listed.Payload.Data {
		if policy == nil {
			return nil, errors.New("list ziti service policies response missing policy")
		}
		policies = append(policies, DebugServicePolicy{
			ID:            stringValue(policy.ID),
			Name:          stringValue(policy.Name),
			Type:          dialBindToString(policy.Type),
			IdentityRoles: rolesToStrings(policy.IdentityRoles),
			ServiceRoles:  rolesToStrings(policy.ServiceRoles),
		})
	}
	return policies, nil
}

func (c *Client) debugServiceTerminators(ctx context.Context, serviceID string) ([]DebugTerminator, error) {
	params := service.NewListServiceTerminatorsParamsWithContext(ctx)
	params.ID = serviceID
	params.Limit = int64Ptr(debugListLimit)
	var listed *service.ListServiceTerminatorsOK
	err := c.withReauth(func() error {
		var callErr error
		serviceClient := c.serviceClient()
		listed, callErr = serviceClient.ListServiceTerminators(params, nil)
		return callErr
	})
	if err != nil {
		return nil, fmt.Errorf("list ziti service terminators: %w", err)
	}
	if listed == nil || listed.Payload == nil {
		return nil, errors.New("list ziti service terminators response missing payload")
	}
	terminators := make([]DebugTerminator, 0, len(listed.Payload.Data))
	for _, terminatorDetail := range listed.Payload.Data {
		if terminatorDetail == nil {
			return nil, errors.New("list ziti service terminators response missing terminator")
		}
		routerName := ""
		if terminatorDetail.Router != nil {
			routerName = terminatorDetail.Router.Name
		}
		terminators = append(terminators, DebugTerminator{
			ID:          stringValue(terminatorDetail.ID),
			Identity:    stringValue(terminatorDetail.Identity),
			RouterID:    stringValue(terminatorDetail.RouterID),
			RouterName:  routerName,
			Precedence:  terminatorPrecedenceToString(terminatorDetail.Precedence),
			Cost:        terminatorCostToInt32(terminatorDetail.Cost),
			DynamicCost: terminatorCostToInt32(terminatorDetail.DynamicCost),
			Binding:     stringValue(terminatorDetail.Binding),
			Address:     stringValue(terminatorDetail.Address),
		})
	}
	return terminators, nil
}

func requiredBaseID(entity *rest_model.ServiceDetail, resource string) (string, error) {
	if entity == nil || entity.ID == nil || *entity.ID == "" {
		return "", fmt.Errorf("%s missing id", resource)
	}
	return *entity.ID, nil
}

func requiredConfigID(config *rest_model.ConfigDetail) (string, error) {
	if config == nil || config.ID == nil || *config.ID == "" {
		return "", errors.New("service config missing config id")
	}
	return *config.ID, nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func int64Ptr(value int64) *int64 {
	return &value
}

func attributesToStrings(attributes *rest_model.Attributes) []string {
	if attributes == nil {
		return nil
	}
	values := make([]string, len(*attributes))
	copy(values, *attributes)
	return values
}

func rolesToStrings(roles rest_model.Roles) []string {
	values := make([]string, len(roles))
	copy(values, roles)
	return values
}

func dialBindToString(value *rest_model.DialBind) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func terminatorPrecedenceToString(value *rest_model.TerminatorPrecedence) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func terminatorCostToInt32(value *rest_model.TerminatorCost) int32 {
	if value == nil {
		return 0
	}
	return int32(*value)
}
