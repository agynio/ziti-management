package ziti

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/agynio/ziti-management/internal/id"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/google/uuid"
	"github.com/openziti/edge-api/rest_management_api_client/config"
	"github.com/openziti/edge-api/rest_management_api_client/enrollment"
	"github.com/openziti/edge-api/rest_management_api_client/identity"
	"github.com/openziti/edge-api/rest_management_api_client/service"
	"github.com/openziti/edge-api/rest_management_api_client/service_policy"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/ziti"
	"github.com/openziti/sdk-golang/ziti/enroll"
)

var ErrIdentityNotFound = errors.New("identity not found")
var ErrServiceNotFound = errors.New("service not found")
var ErrServicePolicyNotFound = errors.New("service policy not found")

const (
	roleAttributeAgents  = "agents"
	roleAttributeApps    = "apps"
	roleAttributeDevices = "devices"
	roleAttributeTunnels = "tunnels"
)

type identityService interface {
	CreateIdentity(params *identity.CreateIdentityParams, authInfo runtime.ClientAuthInfoWriter, opts ...identity.ClientOption) (*identity.CreateIdentityCreated, error)
	DeleteIdentity(params *identity.DeleteIdentityParams, authInfo runtime.ClientAuthInfoWriter, opts ...identity.ClientOption) (*identity.DeleteIdentityOK, error)
	DetailIdentity(params *identity.DetailIdentityParams, authInfo runtime.ClientAuthInfoWriter, opts ...identity.ClientOption) (*identity.DetailIdentityOK, error)
	ListIdentities(params *identity.ListIdentitiesParams, authInfo runtime.ClientAuthInfoWriter, opts ...identity.ClientOption) (*identity.ListIdentitiesOK, error)
	PatchIdentity(params *identity.PatchIdentityParams, authInfo runtime.ClientAuthInfoWriter, opts ...identity.ClientOption) (*identity.PatchIdentityOK, error)
}

type enrollmentService interface {
	CreateEnrollment(params *enrollment.CreateEnrollmentParams, authInfo runtime.ClientAuthInfoWriter, opts ...enrollment.ClientOption) (*enrollment.CreateEnrollmentCreated, error)
	DetailEnrollment(params *enrollment.DetailEnrollmentParams, authInfo runtime.ClientAuthInfoWriter, opts ...enrollment.ClientOption) (*enrollment.DetailEnrollmentOK, error)
}

type serviceService interface {
	CreateService(params *service.CreateServiceParams, authInfo runtime.ClientAuthInfoWriter, opts ...service.ClientOption) (*service.CreateServiceCreated, error)
	UpdateService(params *service.UpdateServiceParams, authInfo runtime.ClientAuthInfoWriter, opts ...service.ClientOption) (*service.UpdateServiceOK, error)
	DeleteService(params *service.DeleteServiceParams, authInfo runtime.ClientAuthInfoWriter, opts ...service.ClientOption) (*service.DeleteServiceOK, error)
	DetailService(params *service.DetailServiceParams, authInfo runtime.ClientAuthInfoWriter, opts ...service.ClientOption) (*service.DetailServiceOK, error)
	ListServiceConfig(params *service.ListServiceConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...service.ClientOption) (*service.ListServiceConfigOK, error)
	ListServices(params *service.ListServicesParams, authInfo runtime.ClientAuthInfoWriter, opts ...service.ClientOption) (*service.ListServicesOK, error)
	PatchService(params *service.PatchServiceParams, authInfo runtime.ClientAuthInfoWriter, opts ...service.ClientOption) (*service.PatchServiceOK, error)
}

type configService interface {
	CreateConfig(params *config.CreateConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...config.ClientOption) (*config.CreateConfigCreated, error)
	ListConfigs(params *config.ListConfigsParams, authInfo runtime.ClientAuthInfoWriter, opts ...config.ClientOption) (*config.ListConfigsOK, error)
	DeleteConfig(params *config.DeleteConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...config.ClientOption) (*config.DeleteConfigOK, error)
	PatchConfig(params *config.PatchConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...config.ClientOption) (*config.PatchConfigOK, error)
}

type servicePolicyService interface {
	CreateServicePolicy(params *service_policy.CreateServicePolicyParams, authInfo runtime.ClientAuthInfoWriter, opts ...service_policy.ClientOption) (*service_policy.CreateServicePolicyCreated, error)
	DetailServicePolicy(params *service_policy.DetailServicePolicyParams, authInfo runtime.ClientAuthInfoWriter, opts ...service_policy.ClientOption) (*service_policy.DetailServicePolicyOK, error)
	DeleteServicePolicy(params *service_policy.DeleteServicePolicyParams, authInfo runtime.ClientAuthInfoWriter, opts ...service_policy.ClientOption) (*service_policy.DeleteServicePolicyOK, error)
	ListServicePolicies(params *service_policy.ListServicePoliciesParams, authInfo runtime.ClientAuthInfoWriter, opts ...service_policy.ClientOption) (*service_policy.ListServicePoliciesOK, error)
}

type Client struct {
	mu               sync.Mutex
	identity         identityService
	enrollment       enrollmentService
	service          serviceService
	config           configService
	servicePolicy    servicePolicyService
	controllerURL    string
	certFile         string
	keyFile          string
	caFile           string
	reauthenticateFn func() error
}

func NewClient(controllerURL, certFile, keyFile, caFile string) (*Client, error) {
	clientCert, privateKey, err := loadClientCredentials(certFile, keyFile)
	if err != nil {
		return nil, err
	}
	caPool, err := loadCAPool(caFile)
	if err != nil {
		return nil, err
	}
	client, err := rest_util.NewEdgeManagementClientWithCert(clientCert, privateKey, controllerURL, caPool)
	if err != nil {
		return nil, fmt.Errorf("create edge management client: %w", err)
	}
	return &Client{
		identity:      client.Identity,
		enrollment:    client.Enrollment,
		service:       client.Service,
		config:        client.Config,
		servicePolicy: client.ServicePolicy,
		controllerURL: controllerURL,
		certFile:      certFile,
		keyFile:       keyFile,
		caFile:        caFile,
	}, nil
}

func loadClientCredentials(certFile, keyFile string) (*x509.Certificate, crypto.PrivateKey, error) {
	certPEM, err := os.ReadFile(certFile)
	if err != nil {
		return nil, nil, fmt.Errorf("read ziti cert: %w", err)
	}
	keyPEM, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("read ziti key: %w", err)
	}

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, nil, fmt.Errorf("parse ziti cert/key: %w", err)
	}
	if len(tlsCert.Certificate) == 0 {
		return nil, nil, errors.New("ziti cert missing certificate data")
	}
	clientCert, err := x509.ParseCertificate(tlsCert.Certificate[0])
	if err != nil {
		return nil, nil, fmt.Errorf("parse ziti cert: %w", err)
	}
	if tlsCert.PrivateKey == nil {
		return nil, nil, errors.New("ziti key missing private key data")
	}
	return clientCert, tlsCert.PrivateKey, nil
}

func loadCAPool(caFile string) (*x509.CertPool, error) {
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return nil, fmt.Errorf("read ziti ca: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caPEM) {
		return nil, errors.New("parse ziti ca bundle")
	}
	return pool, nil
}

type statusCodeChecker interface {
	IsCode(code int) bool
}

func (c *Client) identityClient() identityService {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.identity
}

func (c *Client) enrollmentClient() enrollmentService {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.enrollment
}

func (c *Client) serviceClient() serviceService {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.service
}

func (c *Client) configClient() configService {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.config
}

func (c *Client) servicePolicyClient() servicePolicyService {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.servicePolicy
}

func (c *Client) reauthenticate() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	clientCert, privateKey, err := loadClientCredentials(c.certFile, c.keyFile)
	if err != nil {
		return err
	}
	caPool, err := loadCAPool(c.caFile)
	if err != nil {
		return err
	}
	client, err := rest_util.NewEdgeManagementClientWithCert(clientCert, privateKey, c.controllerURL, caPool)
	if err != nil {
		return fmt.Errorf("create edge management client: %w", err)
	}
	c.identity = client.Identity
	c.enrollment = client.Enrollment
	c.service = client.Service
	c.config = client.Config
	c.servicePolicy = client.ServicePolicy
	return nil
}

func (c *Client) withReauth(operation func() error) error {
	err := operation()
	if err == nil || !isUnauthorized(err) {
		return err
	}
	reauthFn := c.reauthenticate
	if c.reauthenticateFn != nil {
		reauthFn = func() error {
			c.mu.Lock()
			defer c.mu.Unlock()
			return c.reauthenticateFn()
		}
	}
	if reauthErr := reauthFn(); reauthErr != nil {
		return fmt.Errorf("reauthenticate ziti client: %w", reauthErr)
	}
	return operation()
}

func isUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	var checker statusCodeChecker
	if errors.As(err, &checker) && checker.IsCode(401) {
		return true
	}
	return false
}

func extractCreateID(resource string, payload *rest_model.CreateEnvelope) (string, error) {
	if payload == nil || payload.Data == nil {
		return "", fmt.Errorf("create ziti %s response missing data", resource)
	}
	resourceID := payload.Data.ID
	if resourceID == "" {
		return "", fmt.Errorf("create ziti %s response missing id", resource)
	}
	return resourceID, nil
}

func (c *Client) createWithReauth(resource string, operation func() (*rest_model.CreateEnvelope, error)) (string, error) {
	var payload *rest_model.CreateEnvelope
	err := c.withReauth(func() error {
		var callErr error
		payload, callErr = operation()
		return callErr
	})
	if err != nil {
		return "", fmt.Errorf("create ziti %s: %w", resource, err)
	}
	return extractCreateID(resource, payload)
}

func (c *Client) createIdentity(ctx context.Context, identityCreate *rest_model.IdentityCreate) (string, error) {
	params := identity.NewCreateIdentityParamsWithContext(ctx)
	params.Identity = identityCreate
	return c.createWithReauth("identity", func() (*rest_model.CreateEnvelope, error) {
		identityClient := c.identityClient()
		created, err := identityClient.CreateIdentity(params, nil)
		if err != nil {
			return nil, err
		}
		if created == nil {
			return nil, nil
		}
		return created.Payload, nil
	})
}

func mergeRoleAttributes(base []string, additional []string) rest_model.Attributes {
	seen := make(map[string]bool, len(base)+len(additional))
	merged := make(rest_model.Attributes, 0, len(base)+len(additional))
	for _, attr := range append(base, additional...) {
		if attr == "" || seen[attr] {
			continue
		}
		seen[attr] = true
		merged = append(merged, attr)
	}
	return merged
}

func tagsFromMap(values map[string]string) *rest_model.Tags {
	if len(values) == 0 {
		return nil
	}
	tags := rest_model.Tags{}
	for key, value := range values {
		if tags.SubTags == nil {
			tags.SubTags = rest_model.SubTags{}
		}
		tags.SubTags[key] = value
	}
	return &tags
}

func mapFromTags(tags *rest_model.Tags) map[string]string {
	if tags == nil || len(tags.SubTags) == 0 {
		return nil
	}
	values := make(map[string]string, len(tags.SubTags))
	for key, value := range tags.SubTags {
		if stringValue, ok := value.(string); ok {
			values[key] = stringValue
		}
	}
	return values
}

func tagFilter(tags map[string]string) string {
	if len(tags) == 0 {
		return ""
	}
	filters := make([]string, 0, len(tags))
	for key, value := range tags {
		filters = append(filters, fmt.Sprintf("tags.%s=%s", key, strconv.Quote(value)))
	}
	return strings.Join(filters, " and ")
}

func listPagination(pageSize int32, pageToken string) (int64, int64, error) {
	limit := int64(pageSize)
	if limit <= 0 {
		limit = 100
	}
	offset := int64(0)
	if pageToken != "" {
		parsed, err := strconv.ParseInt(pageToken, 10, 64)
		if err != nil || parsed < 0 {
			return 0, 0, fmt.Errorf("invalid page token")
		}
		offset = parsed
	}
	return limit, offset, nil
}

func tagNextPageToken(meta *rest_model.Meta) string {
	if meta == nil || meta.Pagination == nil || meta.Pagination.Offset == nil || meta.Pagination.Limit == nil || meta.Pagination.TotalCount == nil {
		return ""
	}
	nextOffset := *meta.Pagination.Offset + *meta.Pagination.Limit
	if nextOffset >= *meta.Pagination.TotalCount {
		return ""
	}
	return strconv.FormatInt(nextOffset, 10)
}

func (c *Client) deleteIdentityByExternalID(ctx context.Context, externalID string) error {
	identityIDs, err := c.listIdentityIDsByExternalID(ctx, externalID)
	if err != nil {
		return err
	}
	for _, identityID := range identityIDs {
		if err := c.DeleteIdentity(ctx, identityID); err != nil && !errors.Is(err, ErrIdentityNotFound) {
			return err
		}
	}
	return nil
}

func (c *Client) listIdentityIDsByExternalID(ctx context.Context, externalID string) ([]string, error) {
	if externalID == "" {
		return nil, errors.New("external id is empty")
	}
	filter := fmt.Sprintf("externalId=%s", strconv.Quote(externalID))
	limit := int64(100)
	offset := int64(0)
	identityIDs := make([]string, 0)

	for {
		params := identity.NewListIdentitiesParamsWithContext(ctx)
		params.Filter = &filter
		params.Limit = &limit
		params.Offset = &offset

		var listed *identity.ListIdentitiesOK
		err := c.withReauth(func() error {
			var callErr error
			identityClient := c.identityClient()
			listed, callErr = identityClient.ListIdentities(params, nil)
			return callErr
		})
		if err != nil {
			return nil, fmt.Errorf("list ziti identities: %w", err)
		}
		if listed.Payload == nil {
			return nil, errors.New("list ziti identities response missing payload")
		}
		if listed.Payload.Meta == nil || listed.Payload.Meta.Pagination == nil {
			return nil, errors.New("list ziti identities response missing pagination")
		}
		pagination := listed.Payload.Meta.Pagination
		if pagination.TotalCount == nil || pagination.Limit == nil || pagination.Offset == nil {
			return nil, errors.New("list ziti identities response missing pagination details")
		}
		totalCount := *pagination.TotalCount
		if totalCount == 0 {
			if len(listed.Payload.Data) == 0 {
				return nil, nil
			}
			return nil, errors.New("list ziti identities response returned data with zero total count")
		}
		for _, identity := range listed.Payload.Data {
			if identity == nil || identity.ID == nil {
				return nil, errors.New("list ziti identities response missing identity id")
			}
			identityIDs = append(identityIDs, *identity.ID)
		}
		pageCount := int64(len(listed.Payload.Data))
		if pageCount == 0 {
			return nil, errors.New("list ziti identities response returned empty page")
		}
		if offset+pageCount >= totalCount {
			return identityIDs, nil
		}
		offset += pageCount
	}
}

func (c *Client) CreateAgentIdentity(ctx context.Context, agentID, workloadID uuid.UUID) (string, string, error) {
	return c.CreateAgentIdentityWithOptions(ctx, agentID, workloadID, nil, nil)
}

func (c *Client) CreateAgentIdentityWithOptions(ctx context.Context, agentID, workloadID uuid.UUID, additionalRoleAttributes []string, tags map[string]string) (string, string, error) {
	name := fmt.Sprintf("agent-%s-%s", agentID.String(), id.ShortUUID())
	identityType := rest_model.IdentityTypeDevice
	isAdmin := false
	roleAttrs := mergeRoleAttributes([]string{
		roleAttributeAgents,
		fmt.Sprintf("agent-%s", agentID.String()),
		fmt.Sprintf("workload-%s", workloadID.String()),
	}, additionalRoleAttributes)
	externalID := workloadID.String()
	if err := c.deleteIdentityByExternalID(ctx, externalID); err != nil {
		return "", "", fmt.Errorf("delete existing ziti identity: %w", err)
	}
	identityCreate := &rest_model.IdentityCreate{
		Name:           &name,
		Type:           &identityType,
		IsAdmin:        &isAdmin,
		RoleAttributes: &roleAttrs,
		ExternalID:     &externalID,
		Tags:           tagsFromMap(tags),
	}

	zitiID, err := c.createIdentity(ctx, identityCreate)
	if err != nil {
		return "", "", err
	}

	jwt, err := c.createEnrollmentJWT(ctx, zitiID)
	if err != nil {
		return "", "", err
	}
	return zitiID, jwt.Token, nil
}

func (c *Client) createEnrollmentJWT(ctx context.Context, zitiIdentityID string) (EnrollmentJWT, error) {
	method := rest_model.EnrollmentCreateMethodOtt
	expiresAt := time.Now().Add(time.Hour).UTC()
	expires := strfmt.DateTime(expiresAt)
	params := enrollment.NewCreateEnrollmentParamsWithContext(ctx)
	params.Enrollment = &rest_model.EnrollmentCreate{IdentityID: &zitiIdentityID, Method: &method, ExpiresAt: &expires}
	var created *enrollment.CreateEnrollmentCreated
	err := c.withReauth(func() error {
		var callErr error
		enrollmentClient := c.enrollmentClient()
		created, callErr = enrollmentClient.CreateEnrollment(params, nil)
		return callErr
	})
	if err != nil {
		return EnrollmentJWT{}, fmt.Errorf("create ziti enrollment: %w", err)
	}
	enrollmentID, err := extractCreateID("enrollment", created.Payload)
	if err != nil {
		return EnrollmentJWT{}, err
	}

	detailParams := enrollment.NewDetailEnrollmentParamsWithContext(ctx)
	detailParams.ID = enrollmentID
	var detail *enrollment.DetailEnrollmentOK
	err = c.withReauth(func() error {
		var callErr error
		enrollmentClient := c.enrollmentClient()
		detail, callErr = enrollmentClient.DetailEnrollment(detailParams, nil)
		return callErr
	})
	if err != nil {
		return EnrollmentJWT{}, fmt.Errorf("detail ziti enrollment: %w", err)
	}
	if detail.Payload == nil || detail.Payload.Data == nil {
		return EnrollmentJWT{}, errors.New("detail ziti enrollment response missing data")
	}
	data := detail.Payload.Data
	if data.JWT == "" {
		return EnrollmentJWT{}, errors.New("detail ziti enrollment response missing enrollment jwt")
	}
	if data.ExpiresAt != nil {
		expiresAt = time.Time(*data.ExpiresAt)
	}
	return EnrollmentJWT{Token: data.JWT, ExpiresAt: expiresAt}, nil
}

func (c *Client) CreateDeviceIdentity(ctx context.Context, userIdentityID uuid.UUID, name string) (string, string, error) {
	return c.CreateDeviceIdentityWithOptions(ctx, userIdentityID, name, nil, nil)
}

func (c *Client) CreateDeviceIdentityWithOptions(ctx context.Context, userIdentityID uuid.UUID, name string, additionalRoleAttributes []string, tags map[string]string) (string, string, error) {
	identityType := rest_model.IdentityTypeDevice
	isAdmin := false
	roleAttrs := mergeRoleAttributes([]string{roleAttributeDevices}, additionalRoleAttributes)
	externalID := userIdentityID.String()
	identityCreate := &rest_model.IdentityCreate{
		Name:           &name,
		Type:           &identityType,
		IsAdmin:        &isAdmin,
		RoleAttributes: &roleAttrs,
		ExternalID:     &externalID,
		Enrollment:     &rest_model.IdentityCreateEnrollment{Ott: true},
		Tags:           tagsFromMap(tags),
	}

	zitiID, err := c.createIdentity(ctx, identityCreate)
	if err != nil {
		return "", "", err
	}

	jwt, err := c.fetchEnrollmentJWT(ctx, zitiID)
	if err != nil {
		return "", "", err
	}
	return zitiID, jwt.Token, nil
}

func (c *Client) CreateService(ctx context.Context, name string, roleAttributes []string) (string, error) {
	return c.createService(ctx, name, roleAttributes, nil, nil)
}

func (c *Client) CreateServiceWithConfigs(ctx context.Context, name string, roleAttributes []string, hostV1 *HostV1ConfigData, interceptV1 *InterceptV1ConfigData) (string, error) {
	return c.CreateServiceWithConfigsAndTags(ctx, name, roleAttributes, hostV1, interceptV1, nil)
}

func (c *Client) CreateServiceWithConfigsAndTags(ctx context.Context, name string, roleAttributes []string, hostV1 *HostV1ConfigData, interceptV1 *InterceptV1ConfigData, tags map[string]string) (string, error) {
	if hostV1 == nil && interceptV1 == nil {
		return c.createService(ctx, name, roleAttributes, nil, tags)
	}

	configIDs := make([]string, 0, 2)
	if hostV1 != nil {
		data := map[string]any{
			"protocol":        hostV1.Protocol,
			"address":         hostV1.Address,
			"port":            hostV1.Port,
			"forwardProtocol": hostV1.ForwardProtocol,
			"forwardAddress":  hostV1.ForwardAddress,
			"forwardPort":     hostV1.ForwardPort,
		}
		if len(hostV1.AllowedProtocols) > 0 {
			data["allowedProtocols"] = hostV1.AllowedProtocols
		}
		if len(hostV1.AllowedAddresses) > 0 {
			data["allowedAddresses"] = hostV1.AllowedAddresses
		}
		if len(hostV1.AllowedPortRanges) > 0 {
			data["allowedPortRanges"] = portRangeConfigData(hostV1.AllowedPortRanges)
		}
		configID, err := c.createConfig(ctx, hostV1ConfigTypeID, fmt.Sprintf("%s-host-v1", name), data, tags)
		if err != nil {
			return "", err
		}
		configIDs = append(configIDs, configID)
	}
	if interceptV1 != nil {
		data := map[string]any{
			"protocols":  interceptV1.Protocols,
			"addresses":  interceptV1.Addresses,
			"portRanges": portRangeConfigData(interceptV1.PortRanges),
		}
		configID, err := c.createConfig(ctx, interceptV1ConfigTypeID, fmt.Sprintf("%s-intercept-v1", name), data, tags)
		if err != nil {
			return "", c.cleanupConfigs(ctx, configIDs, err)
		}
		configIDs = append(configIDs, configID)
	}

	serviceID, err := c.createService(ctx, name, roleAttributes, configIDs, tags)
	if err != nil {
		return "", c.cleanupConfigs(ctx, configIDs, err)
	}
	return serviceID, nil
}

func portRangeConfigData(portRanges []PortRangeData) []map[string]any {
	data := make([]map[string]any, len(portRanges))
	for i, portRange := range portRanges {
		data[i] = map[string]any{
			"low":  portRange.Low,
			"high": portRange.High,
		}
	}
	return data
}

func (c *Client) createService(ctx context.Context, name string, roleAttributes []string, configIDs []string, tags map[string]string) (string, error) {
	encryptionRequired := true
	params := service.NewCreateServiceParamsWithContext(ctx)
	params.Service = &rest_model.ServiceCreate{
		Name:               &name,
		RoleAttributes:     roleAttributes,
		EncryptionRequired: &encryptionRequired,
		Configs:            configIDs,
		Tags:               tagsFromMap(tags),
	}

	return c.createWithReauth("service", func() (*rest_model.CreateEnvelope, error) {
		serviceClient := c.serviceClient()
		created, err := serviceClient.CreateService(params, nil)
		if err != nil {
			return nil, err
		}
		if created == nil {
			return nil, nil
		}
		return created.Payload, nil
	})
}

func (c *Client) createConfig(ctx context.Context, configTypeID, name string, data map[string]any, tags map[string]string) (string, error) {
	params := config.NewCreateConfigParamsWithContext(ctx)
	params.Config = &rest_model.ConfigCreate{
		ConfigTypeID: &configTypeID,
		Name:         &name,
		Data:         data,
		Tags:         tagsFromMap(tags),
	}

	return c.createWithReauth("config", func() (*rest_model.CreateEnvelope, error) {
		configClient := c.configClient()
		created, err := configClient.CreateConfig(params, nil)
		if err != nil {
			return nil, err
		}
		if created == nil {
			return nil, nil
		}
		return created.Payload, nil
	})
}

func (c *Client) cleanupConfigs(ctx context.Context, configIDs []string, err error) error {
	if len(configIDs) == 0 {
		return err
	}
	var cleanupErr error
	for _, configID := range configIDs {
		if deleteErr := c.deleteConfig(ctx, configID); deleteErr != nil && cleanupErr == nil {
			cleanupErr = deleteErr
		}
	}
	if cleanupErr != nil {
		return fmt.Errorf("%w; cleanup config failed: %w", err, cleanupErr)
	}
	return err
}

func (c *Client) deleteConfig(ctx context.Context, configID string) error {
	params := config.NewDeleteConfigParamsWithContext(ctx)
	params.ID = configID
	err := c.withReauth(func() error {
		configClient := c.configClient()
		_, callErr := configClient.DeleteConfig(params, nil)
		return callErr
	})
	if err == nil {
		return nil
	}
	var notFound *config.DeleteConfigNotFound
	if errors.As(err, &notFound) {
		return nil
	}
	return fmt.Errorf("delete ziti config: %w", err)
}

func (c *Client) DeleteService(ctx context.Context, serviceID string) error {
	params := service.NewDeleteServiceParamsWithContext(ctx)
	params.ID = serviceID
	err := c.withReauth(func() error {
		serviceClient := c.serviceClient()
		_, callErr := serviceClient.DeleteService(params, nil)
		return callErr
	})
	if err == nil {
		return nil
	}
	var notFound *service.DeleteServiceNotFound
	if errors.As(err, &notFound) {
		return ErrServiceNotFound
	}
	return fmt.Errorf("delete ziti service: %w", err)
}

func (c *Client) CreateServicePolicy(ctx context.Context, name, policyType string, identityRoles, serviceRoles []string) (string, error) {
	return c.CreateServicePolicyWithTags(ctx, name, policyType, identityRoles, serviceRoles, nil)
}

func (c *Client) CreateServicePolicyWithTags(ctx context.Context, name, policyType string, identityRoles, serviceRoles []string, tags map[string]string) (string, error) {
	policy := rest_model.DialBind(policyType)
	semantic := rest_model.SemanticAnyOf
	params := service_policy.NewCreateServicePolicyParamsWithContext(ctx)
	params.Policy = &rest_model.ServicePolicyCreate{
		Name:          &name,
		Type:          &policy,
		Semantic:      &semantic,
		IdentityRoles: rest_model.Roles(identityRoles),
		ServiceRoles:  rest_model.Roles(serviceRoles),
		Tags:          tagsFromMap(tags),
	}

	return c.createWithReauth("service policy", func() (*rest_model.CreateEnvelope, error) {
		servicePolicyClient := c.servicePolicyClient()
		created, err := servicePolicyClient.CreateServicePolicy(params, nil)
		if err != nil {
			return nil, err
		}
		if created == nil {
			return nil, nil
		}
		return created.Payload, nil
	})
}

func (c *Client) DeleteServicePolicy(ctx context.Context, policyID string) error {
	params := service_policy.NewDeleteServicePolicyParamsWithContext(ctx)
	params.ID = policyID
	err := c.withReauth(func() error {
		servicePolicyClient := c.servicePolicyClient()
		_, callErr := servicePolicyClient.DeleteServicePolicy(params, nil)
		return callErr
	})
	if err == nil {
		return nil
	}
	var notFound *service_policy.DeleteServicePolicyNotFound
	if errors.As(err, &notFound) {
		return ErrServicePolicyNotFound
	}
	return fmt.Errorf("delete ziti service policy: %w", err)
}

func (c *Client) CreateTunnelIdentity(ctx context.Context, networkID, tunnelCredentialID string, tags map[string]string) (string, EnrollmentJWT, error) {
	name := fmt.Sprintf("tunnel-%s-%s", tunnelCredentialID, id.ShortUUID())
	identityType := rest_model.IdentityTypeDevice
	isAdmin := false
	roleAttrs := rest_model.Attributes{roleAttributeTunnels, fmt.Sprintf("network-%s", networkID)}
	externalID := tunnelCredentialID
	identityCreate := &rest_model.IdentityCreate{
		Name:           &name,
		Type:           &identityType,
		IsAdmin:        &isAdmin,
		RoleAttributes: &roleAttrs,
		ExternalID:     &externalID,
		Enrollment:     &rest_model.IdentityCreateEnrollment{Ott: true},
		Tags:           tagsFromMap(tags),
	}

	zitiID, err := c.createIdentity(ctx, identityCreate)
	if err != nil {
		return "", EnrollmentJWT{}, err
	}

	jwt, err := c.fetchEnrollmentJWT(ctx, zitiID)
	if err != nil {
		return "", EnrollmentJWT{}, err
	}
	return zitiID, jwt, nil
}

func (c *Client) PatchIdentityRoleAttributes(ctx context.Context, zitiIdentityID string, add, remove []string) error {
	detail, err := c.detailIdentity(ctx, zitiIdentityID)
	if err != nil {
		return err
	}

	roleAttributes := patchRoleAttributes(detail.RoleAttributes, add, remove)
	params := identity.NewPatchIdentityParamsWithContext(ctx)
	params.ID = zitiIdentityID
	params.Identity = &rest_model.IdentityPatch{RoleAttributes: &roleAttributes}
	err = c.withReauth(func() error {
		identityClient := c.identityClient()
		_, callErr := identityClient.PatchIdentity(params, nil)
		return callErr
	})
	if err != nil {
		var notFound *identity.PatchIdentityNotFound
		if errors.As(err, &notFound) {
			return ErrIdentityNotFound
		}
		return fmt.Errorf("patch ziti identity role attributes: %w", err)
	}
	return nil
}

func patchRoleAttributes(current *rest_model.Attributes, add, remove []string) rest_model.Attributes {
	roleAttributeSet := make(map[string]struct{}, len(add))
	if current != nil {
		for _, roleAttribute := range *current {
			roleAttributeSet[roleAttribute] = struct{}{}
		}
	}
	for _, roleAttribute := range add {
		roleAttributeSet[roleAttribute] = struct{}{}
	}
	for _, roleAttribute := range remove {
		delete(roleAttributeSet, roleAttribute)
	}

	roleAttributes := make(rest_model.Attributes, 0, len(roleAttributeSet))
	if current != nil {
		for _, roleAttribute := range *current {
			if _, ok := roleAttributeSet[roleAttribute]; ok {
				roleAttributes = append(roleAttributes, roleAttribute)
				delete(roleAttributeSet, roleAttribute)
			}
		}
	}
	for _, roleAttribute := range add {
		if _, ok := roleAttributeSet[roleAttribute]; ok {
			roleAttributes = append(roleAttributes, roleAttribute)
			delete(roleAttributeSet, roleAttribute)
		}
	}
	return roleAttributes
}

func (c *Client) GetIdentityLiveness(ctx context.Context, zitiIdentityID string) (IdentityLiveness, error) {
	detail, err := c.detailIdentity(ctx, zitiIdentityID)
	if err != nil {
		return IdentityLiveness{}, err
	}
	return IdentityLiveness{
		EnrollmentPending:       detail.Enrollment != nil && detail.Enrollment.Ott != nil,
		HasEdgeRouterConnection: detail.HasEdgeRouterConnection != nil && *detail.HasEdgeRouterConnection,
	}, nil
}

func (c *Client) detailIdentity(ctx context.Context, zitiIdentityID string) (*rest_model.IdentityDetail, error) {
	params := identity.NewDetailIdentityParamsWithContext(ctx)
	params.ID = zitiIdentityID
	var detail *identity.DetailIdentityOK
	err := c.withReauth(func() error {
		var callErr error
		identityClient := c.identityClient()
		detail, callErr = identityClient.DetailIdentity(params, nil)
		return callErr
	})
	if err != nil {
		var notFound *identity.DetailIdentityNotFound
		if errors.As(err, &notFound) {
			return nil, ErrIdentityNotFound
		}
		return nil, fmt.Errorf("detail ziti identity: %w", err)
	}
	if detail.Payload == nil || detail.Payload.Data == nil {
		return nil, errors.New("detail ziti identity response missing data")
	}
	return detail.Payload.Data, nil
}

func (c *Client) ListIdentitiesByTag(ctx context.Context, tags map[string]string, pageSize int32, pageToken string) (ListResult[OpenZitiIdentity], error) {
	limit, offset, err := listPagination(pageSize, pageToken)
	if err != nil {
		return ListResult[OpenZitiIdentity]{}, err
	}
	filter := tagFilter(tags)
	params := identity.NewListIdentitiesParamsWithContext(ctx)
	params.Limit = &limit
	params.Offset = &offset
	if filter != "" {
		params.Filter = &filter
	}
	var listed *identity.ListIdentitiesOK
	err = c.withReauth(func() error {
		var callErr error
		identityClient := c.identityClient()
		listed, callErr = identityClient.ListIdentities(params, nil)
		return callErr
	})
	if err != nil {
		return ListResult[OpenZitiIdentity]{}, fmt.Errorf("list ziti identities: %w", err)
	}
	if listed.Payload == nil {
		return ListResult[OpenZitiIdentity]{}, errors.New("list ziti identities response missing payload")
	}
	items := make([]OpenZitiIdentity, 0, len(listed.Payload.Data))
	for _, identityDetail := range listed.Payload.Data {
		items = append(items, toOpenZitiIdentity(identityDetail))
	}
	return ListResult[OpenZitiIdentity]{Items: items, NextPageToken: tagNextPageToken(listed.Payload.Meta)}, nil
}

func (c *Client) ListServicesByTag(ctx context.Context, tags map[string]string, pageSize int32, pageToken string) (ListResult[OpenZitiService], error) {
	limit, offset, err := listPagination(pageSize, pageToken)
	if err != nil {
		return ListResult[OpenZitiService]{}, err
	}
	filter := tagFilter(tags)
	params := service.NewListServicesParamsWithContext(ctx)
	params.Limit = &limit
	params.Offset = &offset
	if filter != "" {
		params.Filter = &filter
	}
	var listed *service.ListServicesOK
	err = c.withReauth(func() error {
		var callErr error
		serviceClient := c.serviceClient()
		listed, callErr = serviceClient.ListServices(params, nil)
		return callErr
	})
	if err != nil {
		return ListResult[OpenZitiService]{}, fmt.Errorf("list ziti services: %w", err)
	}
	if listed.Payload == nil {
		return ListResult[OpenZitiService]{}, errors.New("list ziti services response missing payload")
	}
	items := make([]OpenZitiService, 0, len(listed.Payload.Data))
	for _, serviceDetail := range listed.Payload.Data {
		items = append(items, toOpenZitiService(serviceDetail))
	}
	return ListResult[OpenZitiService]{Items: items, NextPageToken: tagNextPageToken(listed.Payload.Meta)}, nil
}

func (c *Client) ListServicePoliciesByTag(ctx context.Context, tags map[string]string, pageSize int32, pageToken string) (ListResult[OpenZitiServicePolicy], error) {
	limit, offset, err := listPagination(pageSize, pageToken)
	if err != nil {
		return ListResult[OpenZitiServicePolicy]{}, err
	}
	filter := tagFilter(tags)
	params := service_policy.NewListServicePoliciesParamsWithContext(ctx)
	params.Limit = &limit
	params.Offset = &offset
	if filter != "" {
		params.Filter = &filter
	}
	var listed *service_policy.ListServicePoliciesOK
	err = c.withReauth(func() error {
		var callErr error
		servicePolicyClient := c.servicePolicyClient()
		listed, callErr = servicePolicyClient.ListServicePolicies(params, nil)
		return callErr
	})
	if err != nil {
		return ListResult[OpenZitiServicePolicy]{}, fmt.Errorf("list ziti service policies: %w", err)
	}
	if listed.Payload == nil {
		return ListResult[OpenZitiServicePolicy]{}, errors.New("list ziti service policies response missing payload")
	}
	items := make([]OpenZitiServicePolicy, 0, len(listed.Payload.Data))
	for _, policyDetail := range listed.Payload.Data {
		items = append(items, toOpenZitiServicePolicy(policyDetail))
	}
	return ListResult[OpenZitiServicePolicy]{Items: items, NextPageToken: tagNextPageToken(listed.Payload.Meta)}, nil
}

func (c *Client) updateServiceConfigsAndTags(ctx context.Context, serviceID string, hostV1 *HostV1ConfigData, interceptV1 *InterceptV1ConfigData, tags map[string]string, updateTags bool) (OpenZitiService, error) {
	detail, err := c.detailService(ctx, serviceID)
	if err != nil {
		return OpenZitiService{}, err
	}
	configIDs := append([]string(nil), detail.Configs...)
	if hostV1 != nil {
		data := map[string]any{
			"protocol":        hostV1.Protocol,
			"address":         hostV1.Address,
			"port":            hostV1.Port,
			"forwardProtocol": hostV1.ForwardProtocol,
			"forwardAddress":  hostV1.ForwardAddress,
			"forwardPort":     hostV1.ForwardPort,
		}
		if len(hostV1.AllowedProtocols) > 0 {
			data["allowedProtocols"] = hostV1.AllowedProtocols
		}
		if len(hostV1.AllowedAddresses) > 0 {
			data["allowedAddresses"] = hostV1.AllowedAddresses
		}
		if len(hostV1.AllowedPortRanges) > 0 {
			data["allowedPortRanges"] = portRangeConfigData(hostV1.AllowedPortRanges)
		}
		configID, err := c.upsertServiceConfig(ctx, serviceID, hostV1ConfigTypeID, fmt.Sprintf("%s-host-v1", *detail.Name), data, tags, updateTags)
		if err != nil {
			return OpenZitiService{}, err
		}
		configIDs = appendMissing(configIDs, configID)
	}
	if interceptV1 != nil {
		data := map[string]any{
			"protocols":  interceptV1.Protocols,
			"addresses":  interceptV1.Addresses,
			"portRanges": portRangeConfigData(interceptV1.PortRanges),
		}
		configID, err := c.upsertServiceConfig(ctx, serviceID, interceptV1ConfigTypeID, fmt.Sprintf("%s-intercept-v1", *detail.Name), data, tags, updateTags)
		if err != nil {
			return OpenZitiService{}, err
		}
		configIDs = appendMissing(configIDs, configID)
	}
	patch := &rest_model.ServicePatch{}
	if hostV1 != nil || interceptV1 != nil {
		patch.Configs = configIDs
	}
	if updateTags {
		patch.Tags = tagsFromMap(tags)
	}
	params := service.NewPatchServiceParamsWithContext(ctx)
	params.ID = serviceID
	params.Service = patch
	err = c.withReauth(func() error {
		serviceClient := c.serviceClient()
		_, callErr := serviceClient.PatchService(params, nil)
		return callErr
	})
	if err != nil {
		return OpenZitiService{}, fmt.Errorf("patch ziti service: %w", err)
	}
	updated, err := c.detailService(ctx, serviceID)
	if err != nil {
		return OpenZitiService{}, err
	}
	return toOpenZitiService(updated), nil
}

func (c *Client) detailService(ctx context.Context, serviceID string) (*rest_model.ServiceDetail, error) {
	params := service.NewDetailServiceParamsWithContext(ctx)
	params.ID = serviceID
	var detail *service.DetailServiceOK
	err := c.withReauth(func() error {
		var callErr error
		serviceClient := c.serviceClient()
		detail, callErr = serviceClient.DetailService(params, nil)
		return callErr
	})
	if err != nil {
		return nil, fmt.Errorf("detail ziti service: %w", err)
	}
	if detail.Payload == nil || detail.Payload.Data == nil {
		return nil, errors.New("detail ziti service response missing data")
	}
	return detail.Payload.Data, nil
}

func (c *Client) upsertServiceConfig(ctx context.Context, serviceID, configTypeID, name string, data map[string]any, tags map[string]string, updateTags bool) (string, error) {
	configDetail, err := c.findServiceConfigByType(ctx, serviceID, configTypeID)
	if err != nil {
		return "", err
	}
	if configDetail == nil {
		configID, _, err := c.upsertConfigByName(ctx, configTypeID, name, data, tags, updateTags)
		return configID, err
	}
	return c.patchConfig(ctx, configDetail, name, data, tags, updateTags)
}

func (c *Client) findServiceConfigByType(ctx context.Context, serviceID, configTypeID string) (*rest_model.ConfigDetail, error) {
	limit := int64(100)
	offset := int64(0)
	for {
		params := service.NewListServiceConfigParamsWithContext(ctx)
		params.ID = serviceID
		params.Limit = &limit
		params.Offset = &offset
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
		if listed.Payload == nil {
			return nil, errors.New("list ziti service configs response missing payload")
		}
		for _, configDetail := range listed.Payload.Data {
			if configDetail != nil && configDetail.ConfigTypeID != nil && *configDetail.ConfigTypeID == configTypeID {
				return configDetail, nil
			}
		}
		if listed.Payload.Meta == nil || listed.Payload.Meta.Pagination == nil || listed.Payload.Meta.Pagination.TotalCount == nil {
			return nil, nil
		}
		pageCount := int64(len(listed.Payload.Data))
		if pageCount == 0 || offset+pageCount >= *listed.Payload.Meta.Pagination.TotalCount {
			return nil, nil
		}
		offset += pageCount
	}
}

func appendMissing(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

func toOpenZitiIdentity(detail *rest_model.IdentityDetail) OpenZitiIdentity {
	return OpenZitiIdentity{ID: *detail.ID, Name: *detail.Name, RoleAttributes: []string(*detail.RoleAttributes), Tags: mapFromTags(detail.Tags)}
}

func toOpenZitiService(detail *rest_model.ServiceDetail) OpenZitiService {
	return OpenZitiService{ID: *detail.ID, Name: *detail.Name, RoleAttributes: []string(*detail.RoleAttributes), Tags: mapFromTags(detail.Tags)}
}

func toOpenZitiServicePolicy(detail *rest_model.ServicePolicyDetail) OpenZitiServicePolicy {
	return OpenZitiServicePolicy{ID: *detail.ID, Name: *detail.Name, Type: string(*detail.Type), IdentityRoles: []string(detail.IdentityRoles), ServiceRoles: []string(detail.ServiceRoles), Tags: mapFromTags(detail.Tags)}
}

func (c *Client) CreateAndEnrollServiceIdentity(ctx context.Context, name string, roleAttributes []string) (string, []byte, error) {
	identityType := rest_model.IdentityTypeDevice
	isAdmin := false
	attrs := rest_model.Attributes(roleAttributes)
	identityCreate := &rest_model.IdentityCreate{
		Name:           &name,
		Type:           &identityType,
		IsAdmin:        &isAdmin,
		RoleAttributes: &attrs,
		Enrollment:     &rest_model.IdentityCreateEnrollment{Ott: true},
	}

	return c.createAndEnrollIdentity(ctx, identityCreate)
}

func (c *Client) createAndEnrollIdentity(ctx context.Context, identityCreate *rest_model.IdentityCreate) (string, []byte, error) {
	zitiID, err := c.createIdentity(ctx, identityCreate)
	if err != nil {
		return "", nil, err
	}

	identityJSON, err := c.enrollIdentity(ctx, zitiID)
	if err != nil {
		return "", nil, c.cleanupServiceIdentity(ctx, zitiID, err)
	}
	return zitiID, identityJSON, nil
}

func (c *Client) CreateAndEnrollAppIdentity(ctx context.Context, appID uuid.UUID, slug string) (string, []byte, error) {
	return c.CreateAndEnrollAppIdentityWithOptions(ctx, appID, slug, nil, nil)
}

func (c *Client) CreateAndEnrollAppIdentityWithOptions(ctx context.Context, appID uuid.UUID, slug string, additionalRoleAttributes []string, tags map[string]string) (string, []byte, error) {
	name := fmt.Sprintf("app-%s-%s", slug, id.ShortUUID())
	identityType := rest_model.IdentityTypeDevice
	isAdmin := false
	roleAttrs := mergeRoleAttributes([]string{roleAttributeApps}, additionalRoleAttributes)
	externalID := appID.String()
	if err := c.deleteIdentityByExternalID(ctx, externalID); err != nil {
		return "", nil, fmt.Errorf("delete existing ziti identity: %w", err)
	}
	identityCreate := &rest_model.IdentityCreate{
		Name:           &name,
		Type:           &identityType,
		IsAdmin:        &isAdmin,
		RoleAttributes: &roleAttrs,
		ExternalID:     &externalID,
		Enrollment:     &rest_model.IdentityCreateEnrollment{Ott: true},
		Tags:           tagsFromMap(tags),
	}

	return c.createAndEnrollIdentity(ctx, identityCreate)
}

func (c *Client) CreateAndEnrollRunnerIdentity(ctx context.Context, runnerID uuid.UUID, roleAttributes []string) (string, []byte, error) {
	return c.CreateAndEnrollRunnerIdentityWithTags(ctx, runnerID, roleAttributes, nil)
}

func (c *Client) CreateAndEnrollRunnerIdentityWithTags(ctx context.Context, runnerID uuid.UUID, roleAttributes []string, tags map[string]string) (string, []byte, error) {
	name := fmt.Sprintf("runner-%s-%s", runnerID.String(), id.ShortUUID())
	identityType := rest_model.IdentityTypeDevice
	isAdmin := false
	roleAttrs := rest_model.Attributes(roleAttributes)
	externalID := runnerID.String()
	if err := c.deleteIdentityByExternalID(ctx, externalID); err != nil {
		return "", nil, fmt.Errorf("delete existing ziti identity: %w", err)
	}
	identityCreate := &rest_model.IdentityCreate{
		Name:           &name,
		Type:           &identityType,
		IsAdmin:        &isAdmin,
		RoleAttributes: &roleAttrs,
		ExternalID:     &externalID,
		Enrollment:     &rest_model.IdentityCreateEnrollment{Ott: true},
		Tags:           tagsFromMap(tags),
	}

	return c.createAndEnrollIdentity(ctx, identityCreate)
}

func (c *Client) enrollIdentity(ctx context.Context, zitiIdentityID string) ([]byte, error) {
	jwt, err := c.fetchEnrollmentJWT(ctx, zitiIdentityID)
	if err != nil {
		return nil, err
	}

	parseToken := parseEnrollmentToken
	enrollFn := enrollIdentity

	claims, _, err := parseToken(jwt.Token)
	if err != nil {
		return nil, fmt.Errorf("parse enrollment token: %w", err)
	}

	var keyAlg ziti.KeyAlgVar
	if err := keyAlg.Set("EC"); err != nil {
		return nil, fmt.Errorf("set key algorithm: %w", err)
	}
	config, err := enrollFn(enroll.EnrollmentFlags{Token: claims, KeyAlg: keyAlg})
	if err != nil {
		return nil, fmt.Errorf("enroll identity: %w", err)
	}
	identityJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshal identity json: %w", err)
	}
	return identityJSON, nil
}

func (c *Client) fetchEnrollmentJWT(ctx context.Context, zitiIdentityID string) (EnrollmentJWT, error) {
	detailParams := identity.NewDetailIdentityParamsWithContext(ctx)
	detailParams.ID = zitiIdentityID
	var detail *identity.DetailIdentityOK
	err := c.withReauth(func() error {
		var callErr error
		identityClient := c.identityClient()
		detail, callErr = identityClient.DetailIdentity(detailParams, nil)
		return callErr
	})
	if err != nil {
		return EnrollmentJWT{}, fmt.Errorf("detail ziti identity: %w", err)
	}
	if detail.Payload == nil || detail.Payload.Data == nil || detail.Payload.Data.Enrollment == nil || detail.Payload.Data.Enrollment.Ott == nil {
		return EnrollmentJWT{}, errors.New("detail ziti identity response missing enrollment")
	}
	ott := detail.Payload.Data.Enrollment.Ott
	if ott.JWT == "" {
		return EnrollmentJWT{}, errors.New("detail ziti identity response missing enrollment jwt")
	}
	expiresAt := time.Time(ott.ExpiresAt)
	if expiresAt.IsZero() {
		claims, _, err := parseEnrollmentToken(ott.JWT)
		if err != nil {
			return EnrollmentJWT{}, fmt.Errorf("parse enrollment token: %w", err)
		}
		if claims.ExpiresAt == nil {
			return EnrollmentJWT{}, errors.New("enrollment jwt missing expiration")
		}
		expiresAt = claims.ExpiresAt.Time
	}
	return EnrollmentJWT{Token: ott.JWT, ExpiresAt: expiresAt}, nil
}

func (c *Client) cleanupServiceIdentity(ctx context.Context, zitiIdentityID string, err error) error {
	cleanupErr := c.DeleteIdentity(ctx, zitiIdentityID)
	if cleanupErr == nil || errors.Is(cleanupErr, ErrIdentityNotFound) {
		return err
	}
	return fmt.Errorf("%w; cleanup failed: %w", err, cleanupErr)
}

func (c *Client) DeleteIdentity(ctx context.Context, zitiIdentityID string) error {
	params := identity.NewDeleteIdentityParamsWithContext(ctx)
	params.ID = zitiIdentityID
	err := c.withReauth(func() error {
		identityClient := c.identityClient()
		_, callErr := identityClient.DeleteIdentity(params, nil)
		return callErr
	})
	if err == nil {
		return nil
	}
	var notFound *identity.DeleteIdentityNotFound
	if errors.As(err, &notFound) {
		return ErrIdentityNotFound
	}
	return fmt.Errorf("delete ziti identity: %w", err)
}

const defaultListLimit int64 = 100

func (c *Client) GetService(ctx context.Context, id string) (Service, error) {
	params := service.NewDetailServiceParamsWithContext(ctx)
	params.ID = id
	var detail *rest_model.ServiceDetail
	err := c.withReauth(func() error {
		serviceClient := c.serviceClient()
		resp, callErr := serviceClient.DetailService(params, nil)
		if callErr != nil {
			return callErr
		}
		if resp != nil && resp.Payload != nil {
			detail = resp.Payload.Data
		}
		return nil
	})
	if err != nil {
		var notFound *service.DetailServiceNotFound
		if errors.As(err, &notFound) {
			return Service{}, ErrServiceNotFound
		}
		return Service{}, fmt.Errorf("get ziti service: %w", err)
	}
	if detail == nil {
		return Service{}, fmt.Errorf("get ziti service: missing response data")
	}
	return serviceFromDetail(detail), nil
}

func (c *Client) GetServiceByName(ctx context.Context, name string) (Service, error) {
	result, err := c.ListServices(ctx, ServiceListFilter{Name: name, PageSize: 2})
	if err != nil {
		return Service{}, err
	}
	if len(result.Services) == 0 {
		return Service{}, ErrServiceNotFound
	}
	return result.Services[0], nil
}

func (c *Client) ListServices(ctx context.Context, filter ServiceListFilter) (ServiceListResult, error) {
	limit := listLimit(filter.PageSize)
	offset, err := decodePageToken(filter.PageToken)
	if err != nil {
		return ServiceListResult{}, err
	}
	queryFilter := serviceFilter(filter)
	params := service.NewListServicesParamsWithContext(ctx)
	params.Limit = &limit
	params.Offset = &offset
	if queryFilter != "" {
		params.Filter = &queryFilter
	}
	if len(filter.RoleAttributes) > 0 {
		params.RoleFilter = filter.RoleAttributes
	}
	var envelope *rest_model.ListServicesEnvelope
	err = c.withReauth(func() error {
		serviceClient := c.serviceClient()
		resp, callErr := serviceClient.ListServices(params, nil)
		if callErr != nil {
			return callErr
		}
		if resp != nil {
			envelope = resp.Payload
		}
		return nil
	})
	if err != nil {
		return ServiceListResult{}, fmt.Errorf("list ziti services: %w", err)
	}
	if envelope == nil {
		return ServiceListResult{}, fmt.Errorf("list ziti services: missing response data")
	}
	items := make([]Service, 0, len(envelope.Data))
	for _, detail := range envelope.Data {
		items = append(items, serviceFromDetail(detail))
	}
	return ServiceListResult{Services: items, NextPageToken: nextPageToken(envelope.Meta)}, nil
}

func (c *Client) CreateServiceReturning(ctx context.Context, name string, roleAttributes []string, hostV1 *HostV1ConfigData, interceptV1 *InterceptV1ConfigData, returnExisting bool) (Service, error) {
	serviceID, err := c.CreateServiceWithConfigs(ctx, name, roleAttributes, hostV1, interceptV1)
	if err == nil {
		return c.GetService(ctx, serviceID)
	}
	if !returnExisting || !isConflictError(err) {
		return Service{}, err
	}
	return c.GetServiceByName(ctx, name)
}

func (c *Client) UpdateService(ctx context.Context, id string, name string, roleAttributes []string, hostV1 *HostV1ConfigData, interceptV1 *InterceptV1ConfigData) (Service, error) {
	configIDs := make([]string, 0, 2)
	createdConfigIDs := make([]string, 0, 2)
	if hostV1 != nil {
		configID, created, err := c.upsertConfigByName(ctx, hostV1ConfigTypeID, fmt.Sprintf("%s-host-v1", name), hostV1ConfigData(hostV1), nil, false)
		if err != nil {
			return Service{}, err
		}
		configIDs = append(configIDs, configID)
		if created {
			createdConfigIDs = append(createdConfigIDs, configID)
		}
	}
	if interceptV1 != nil {
		configID, created, err := c.upsertConfigByName(ctx, interceptV1ConfigTypeID, fmt.Sprintf("%s-intercept-v1", name), interceptV1ConfigData(interceptV1), nil, false)
		if err != nil {
			return Service{}, c.cleanupConfigs(ctx, createdConfigIDs, err)
		}
		configIDs = append(configIDs, configID)
		if created {
			createdConfigIDs = append(createdConfigIDs, configID)
		}
	}
	params := service.NewUpdateServiceParamsWithContext(ctx)
	params.ID = id
	params.Service = &rest_model.ServiceUpdate{Name: &name, RoleAttributes: roleAttributes, Configs: configIDs}
	err := c.withReauth(func() error {
		serviceClient := c.serviceClient()
		_, callErr := serviceClient.UpdateService(params, nil)
		return callErr
	})
	if err != nil {
		var notFound *service.UpdateServiceNotFound
		if errors.As(err, &notFound) {
			return Service{}, c.cleanupConfigs(ctx, createdConfigIDs, ErrServiceNotFound)
		}
		return Service{}, c.cleanupConfigs(ctx, createdConfigIDs, fmt.Errorf("update ziti service: %w", err))
	}
	return c.GetService(ctx, id)
}

func (c *Client) GetServicePolicy(ctx context.Context, id string) (ServicePolicy, error) {
	params := service_policy.NewDetailServicePolicyParamsWithContext(ctx)
	params.ID = id
	var detail *rest_model.ServicePolicyDetail
	err := c.withReauth(func() error {
		servicePolicyClient := c.servicePolicyClient()
		resp, callErr := servicePolicyClient.DetailServicePolicy(params, nil)
		if callErr != nil {
			return callErr
		}
		if resp != nil && resp.Payload != nil {
			detail = resp.Payload.Data
		}
		return nil
	})
	if err != nil {
		var notFound *service_policy.DetailServicePolicyNotFound
		if errors.As(err, &notFound) {
			return ServicePolicy{}, ErrServicePolicyNotFound
		}
		return ServicePolicy{}, fmt.Errorf("get ziti service policy: %w", err)
	}
	if detail == nil {
		return ServicePolicy{}, fmt.Errorf("get ziti service policy: missing response data")
	}
	return servicePolicyFromDetail(detail), nil
}

func (c *Client) GetServicePolicyByName(ctx context.Context, name string) (ServicePolicy, error) {
	result, err := c.ListServicePolicies(ctx, ServicePolicyListFilter{Name: name, PageSize: 2})
	if err != nil {
		return ServicePolicy{}, err
	}
	if len(result.ServicePolicies) == 0 {
		return ServicePolicy{}, ErrServicePolicyNotFound
	}
	return result.ServicePolicies[0], nil
}

func (c *Client) ListServicePolicies(ctx context.Context, filter ServicePolicyListFilter) (ServicePolicyListResult, error) {
	limit := listLimit(filter.PageSize)
	offset, err := decodePageToken(filter.PageToken)
	if err != nil {
		return ServicePolicyListResult{}, err
	}
	queryFilter := servicePolicyFilter(filter)
	params := service_policy.NewListServicePoliciesParamsWithContext(ctx)
	params.Limit = &limit
	params.Offset = &offset
	if queryFilter != "" {
		params.Filter = &queryFilter
	}
	var envelope *rest_model.ListServicePoliciesEnvelope
	err = c.withReauth(func() error {
		servicePolicyClient := c.servicePolicyClient()
		resp, callErr := servicePolicyClient.ListServicePolicies(params, nil)
		if callErr != nil {
			return callErr
		}
		if resp != nil {
			envelope = resp.Payload
		}
		return nil
	})
	if err != nil {
		return ServicePolicyListResult{}, fmt.Errorf("list ziti service policies: %w", err)
	}
	if envelope == nil {
		return ServicePolicyListResult{}, fmt.Errorf("list ziti service policies: missing response data")
	}
	items := make([]ServicePolicy, 0, len(envelope.Data))
	for _, detail := range envelope.Data {
		items = append(items, servicePolicyFromDetail(detail))
	}
	return ServicePolicyListResult{ServicePolicies: items, NextPageToken: nextPageToken(envelope.Meta)}, nil
}

func (c *Client) CreateServicePolicyReturning(ctx context.Context, name, policyType string, identityRoles, serviceRoles []string, returnExisting bool) (ServicePolicy, error) {
	policyID, err := c.CreateServicePolicy(ctx, name, policyType, identityRoles, serviceRoles)
	if err == nil {
		return c.GetServicePolicy(ctx, policyID)
	}
	if !returnExisting || !isConflictError(err) {
		return ServicePolicy{}, err
	}
	return c.GetServicePolicyByName(ctx, name)
}

func hostV1ConfigData(hostV1 *HostV1ConfigData) map[string]any {
	data := map[string]any{
		"protocol":        hostV1.Protocol,
		"address":         hostV1.Address,
		"port":            hostV1.Port,
		"forwardProtocol": hostV1.ForwardProtocol,
		"forwardAddress":  hostV1.ForwardAddress,
		"forwardPort":     hostV1.ForwardPort,
	}
	if len(hostV1.AllowedProtocols) > 0 {
		data["allowedProtocols"] = hostV1.AllowedProtocols
	}
	if len(hostV1.AllowedAddresses) > 0 {
		data["allowedAddresses"] = hostV1.AllowedAddresses
	}
	if len(hostV1.AllowedPortRanges) > 0 {
		data["allowedPortRanges"] = portRangeConfigData(hostV1.AllowedPortRanges)
	}
	return data
}

func interceptV1ConfigData(interceptV1 *InterceptV1ConfigData) map[string]any {
	return map[string]any{
		"protocols":  interceptV1.Protocols,
		"addresses":  interceptV1.Addresses,
		"portRanges": portRangeConfigData(interceptV1.PortRanges),
	}
}

func serviceFromDetail(detail *rest_model.ServiceDetail) Service {
	return Service{
		ID:                stringValue(detail.ID),
		Name:              stringValue(detail.Name),
		RoleAttributes:    attributesToStrings(detail.RoleAttributes),
		HostV1Config:      hostV1FromConfig(detail.Config["host.v1"]),
		InterceptV1Config: interceptV1FromConfig(detail.Config["intercept.v1"]),
	}
}

func servicePolicyFromDetail(detail *rest_model.ServicePolicyDetail) ServicePolicy {
	return ServicePolicy{
		ID:            stringValue(detail.ID),
		Name:          stringValue(detail.Name),
		Type:          stringValue((*string)(detail.Type)),
		IdentityRoles: rolesToStrings(detail.IdentityRoles),
		ServiceRoles:  rolesToStrings(detail.ServiceRoles),
	}
}

func serviceFilter(filter ServiceListFilter) string {
	parts := make([]string, 0)
	if filter.Name != "" {
		parts = append(parts, fmt.Sprintf(`name = "%s"`, escapeFilterValue(filter.Name)))
	}
	if filter.NamePrefix != "" {
		parts = append(parts, fmt.Sprintf(`name contains "%s"`, escapeFilterValue(filter.NamePrefix)))
	}
	return strings.Join(parts, " and ")
}

func servicePolicyFilter(filter ServicePolicyListFilter) string {
	parts := make([]string, 0)
	if filter.Name != "" {
		parts = append(parts, fmt.Sprintf(`name = "%s"`, escapeFilterValue(filter.Name)))
	}
	if filter.NamePrefix != "" {
		parts = append(parts, fmt.Sprintf(`name contains "%s"`, escapeFilterValue(filter.NamePrefix)))
	}
	if filter.Type != "" {
		parts = append(parts, fmt.Sprintf(`type = "%s"`, escapeFilterValue(filter.Type)))
	}
	for _, role := range filter.IdentityRoles {
		parts = append(parts, fmt.Sprintf(`identityRoles contains "%s"`, escapeFilterValue(role)))
	}
	for _, role := range filter.ServiceRoles {
		parts = append(parts, fmt.Sprintf(`serviceRoles contains "%s"`, escapeFilterValue(role)))
	}
	return strings.Join(parts, " and ")
}

func escapeFilterValue(value string) string {
	return strings.ReplaceAll(value, `"`, `\"`)
}

func listLimit(pageSize int32) int64 {
	if pageSize <= 0 {
		return defaultListLimit
	}
	return int64(pageSize)
}

func decodePageToken(token string) (int64, error) {
	if token == "" {
		return 0, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return 0, fmt.Errorf("invalid page token")
	}
	offset, err := strconv.ParseInt(string(raw), 10, 64)
	if err != nil || offset < 0 {
		return 0, fmt.Errorf("invalid page token")
	}
	return offset, nil
}

func nextPageToken(meta *rest_model.Meta) string {
	if meta == nil || meta.Pagination == nil || meta.Pagination.Offset == nil || meta.Pagination.Limit == nil || meta.Pagination.TotalCount == nil {
		return ""
	}
	nextOffset := *meta.Pagination.Offset + *meta.Pagination.Limit
	if nextOffset >= *meta.Pagination.TotalCount {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString([]byte(strconv.FormatInt(nextOffset, 10)))
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func attributesToStrings(values *rest_model.Attributes) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), ([]string)(*values)...)
}

func rolesToStrings(values rest_model.Roles) []string {
	return append([]string(nil), ([]string)(values)...)
}

func hostV1FromConfig(config map[string]any) *HostV1ConfigData {
	if config == nil {
		return nil
	}
	return &HostV1ConfigData{
		Protocol:          stringFromMap(config, "protocol"),
		Address:           stringFromMap(config, "address"),
		Port:              int32FromMap(config, "port"),
		ForwardProtocol:   boolFromMap(config, "forwardProtocol"),
		ForwardAddress:    boolFromMap(config, "forwardAddress"),
		ForwardPort:       boolFromMap(config, "forwardPort"),
		AllowedProtocols:  stringsFromMap(config, "allowedProtocols"),
		AllowedAddresses:  stringsFromMap(config, "allowedAddresses"),
		AllowedPortRanges: portRangesFromMap(config, "allowedPortRanges"),
	}
}

func interceptV1FromConfig(config map[string]any) *InterceptV1ConfigData {
	if config == nil {
		return nil
	}
	return &InterceptV1ConfigData{
		Protocols:  stringsFromMap(config, "protocols"),
		Addresses:  stringsFromMap(config, "addresses"),
		PortRanges: portRangesFromMap(config, "portRanges"),
	}
}

func stringFromMap(values map[string]any, key string) string {
	value, ok := values[key].(string)
	if !ok {
		return ""
	}
	return value
}

func boolFromMap(values map[string]any, key string) bool {
	value, ok := values[key].(bool)
	return ok && value
}

func int32FromMap(values map[string]any, key string) int32 {
	switch value := values[key].(type) {
	case int32:
		return value
	case int:
		return int32(value)
	case int64:
		return int32(value)
	case float64:
		return int32(value)
	case json.Number:
		parsed, err := strconv.ParseInt(value.String(), 10, 32)
		if err != nil {
			return 0
		}
		return int32(parsed)
	default:
		return 0
	}
}

func stringsFromMap(values map[string]any, key string) []string {
	raw, ok := values[key].([]any)
	if !ok {
		if strings, ok := values[key].([]string); ok {
			return append([]string(nil), strings...)
		}
		return nil
	}
	strings := make([]string, 0, len(raw))
	for _, item := range raw {
		value, ok := item.(string)
		if ok {
			strings = append(strings, value)
		}
	}
	return strings
}

func portRangesFromMap(values map[string]any, key string) []PortRangeData {
	raw, ok := values[key].([]any)
	if !ok {
		return nil
	}
	portRanges := make([]PortRangeData, 0, len(raw))
	for _, item := range raw {
		value, ok := item.(map[string]any)
		if !ok {
			continue
		}
		portRanges = append(portRanges, PortRangeData{Low: int32FromMap(value, "low"), High: int32FromMap(value, "high")})
	}
	return portRanges
}

func isConflictError(err error) bool {
	var checker statusCodeChecker
	return errors.As(err, &checker) && checker.IsCode(409)
}

func (c *Client) upsertConfigByName(ctx context.Context, configTypeID, name string, data map[string]any, tags map[string]string, updateTags bool) (string, bool, error) {
	configDetail, err := c.findConfigByName(ctx, name)
	if err != nil {
		return "", false, err
	}
	if configDetail == nil {
		configID, err := c.createConfig(ctx, configTypeID, name, data, tags)
		return configID, true, err
	}
	if configDetail.ConfigTypeID == nil || *configDetail.ConfigTypeID != configTypeID {
		return "", false, fmt.Errorf("config %s has unexpected config type", name)
	}
	configID, err := c.patchConfig(ctx, configDetail, name, data, tags, updateTags)
	return configID, false, err
}

func (c *Client) patchConfig(ctx context.Context, configDetail *rest_model.ConfigDetail, name string, data map[string]any, tags map[string]string, updateTags bool) (string, error) {
	patch := &rest_model.ConfigPatch{Data: data, Name: name}
	if updateTags {
		patch.Tags = tagsFromMap(tags)
	}
	params := config.NewPatchConfigParamsWithContext(ctx)
	params.ID = *configDetail.ID
	params.Config = patch
	err := c.withReauth(func() error {
		configClient := c.configClient()
		_, callErr := configClient.PatchConfig(params, nil)
		return callErr
	})
	if err != nil {
		return "", fmt.Errorf("patch ziti config: %w", err)
	}
	return *configDetail.ID, nil
}

func (c *Client) findConfigByName(ctx context.Context, name string) (*rest_model.ConfigDetail, error) {
	limit := int64(2)
	offset := int64(0)
	filter := fmt.Sprintf(`name = "%s"`, escapeFilterValue(name))
	params := config.NewListConfigsParamsWithContext(ctx)
	params.Limit = &limit
	params.Offset = &offset
	params.Filter = &filter
	var listed *config.ListConfigsOK
	err := c.withReauth(func() error {
		configClient := c.configClient()
		var callErr error
		listed, callErr = configClient.ListConfigs(params, nil)
		return callErr
	})
	if err != nil {
		return nil, fmt.Errorf("list ziti configs: %w", err)
	}
	if listed.Payload == nil {
		return nil, errors.New("list ziti configs response missing payload")
	}
	if len(listed.Payload.Data) == 0 {
		return nil, nil
	}
	return listed.Payload.Data[0], nil
}
