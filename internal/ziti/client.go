package ziti

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/agynio/ziti-management/internal/id"
	"github.com/go-openapi/runtime"
	"github.com/google/uuid"
	"github.com/openziti/edge-api/rest_management_api_client/config"
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
)

type identityService interface {
	CreateIdentity(params *identity.CreateIdentityParams, authInfo runtime.ClientAuthInfoWriter, opts ...identity.ClientOption) (*identity.CreateIdentityCreated, error)
	DeleteIdentity(params *identity.DeleteIdentityParams, authInfo runtime.ClientAuthInfoWriter, opts ...identity.ClientOption) (*identity.DeleteIdentityOK, error)
	DetailIdentity(params *identity.DetailIdentityParams, authInfo runtime.ClientAuthInfoWriter, opts ...identity.ClientOption) (*identity.DetailIdentityOK, error)
	ListIdentities(params *identity.ListIdentitiesParams, authInfo runtime.ClientAuthInfoWriter, opts ...identity.ClientOption) (*identity.ListIdentitiesOK, error)
}

type serviceService interface {
	CreateService(params *service.CreateServiceParams, authInfo runtime.ClientAuthInfoWriter, opts ...service.ClientOption) (*service.CreateServiceCreated, error)
	DeleteService(params *service.DeleteServiceParams, authInfo runtime.ClientAuthInfoWriter, opts ...service.ClientOption) (*service.DeleteServiceOK, error)
}

type configService interface {
	CreateConfig(params *config.CreateConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...config.ClientOption) (*config.CreateConfigCreated, error)
	DeleteConfig(params *config.DeleteConfigParams, authInfo runtime.ClientAuthInfoWriter, opts ...config.ClientOption) (*config.DeleteConfigOK, error)
}

type servicePolicyService interface {
	CreateServicePolicy(params *service_policy.CreateServicePolicyParams, authInfo runtime.ClientAuthInfoWriter, opts ...service_policy.ClientOption) (*service_policy.CreateServicePolicyCreated, error)
	DeleteServicePolicy(params *service_policy.DeleteServicePolicyParams, authInfo runtime.ClientAuthInfoWriter, opts ...service_policy.ClientOption) (*service_policy.DeleteServicePolicyOK, error)
}

type Client struct {
	mu               sync.Mutex
	identity         identityService
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
	name := fmt.Sprintf("agent-%s-%s", agentID.String(), id.ShortUUID())
	identityType := rest_model.IdentityTypeDevice
	isAdmin := false
	roleAttrs := rest_model.Attributes{
		roleAttributeAgents,
		fmt.Sprintf("agent-%s", agentID.String()),
		fmt.Sprintf("workload-%s", workloadID.String()),
	}
	externalID := workloadID.String()
	identityCreate := &rest_model.IdentityCreate{
		Name:           &name,
		Type:           &identityType,
		IsAdmin:        &isAdmin,
		RoleAttributes: &roleAttrs,
		ExternalID:     &externalID,
		Enrollment:     &rest_model.IdentityCreateEnrollment{Ott: true},
	}

	zitiID, err := c.createIdentity(ctx, identityCreate)
	if err != nil {
		return "", "", err
	}

	jwt, err := c.fetchEnrollmentJWT(ctx, zitiID)
	if err != nil {
		return "", "", err
	}
	return zitiID, jwt, nil
}

func (c *Client) CreateDeviceIdentity(ctx context.Context, userIdentityID uuid.UUID, name string) (string, string, error) {
	identityType := rest_model.IdentityTypeDevice
	isAdmin := false
	roleAttrs := rest_model.Attributes{roleAttributeDevices}
	externalID := userIdentityID.String()
	identityCreate := &rest_model.IdentityCreate{
		Name:           &name,
		Type:           &identityType,
		IsAdmin:        &isAdmin,
		RoleAttributes: &roleAttrs,
		ExternalID:     &externalID,
		Enrollment:     &rest_model.IdentityCreateEnrollment{Ott: true},
	}

	zitiID, err := c.createIdentity(ctx, identityCreate)
	if err != nil {
		return "", "", err
	}

	jwt, err := c.fetchEnrollmentJWT(ctx, zitiID)
	if err != nil {
		return "", "", err
	}
	return zitiID, jwt, nil
}

func (c *Client) CreateService(ctx context.Context, name string, roleAttributes []string) (string, error) {
	return c.createService(ctx, name, roleAttributes, nil)
}

func (c *Client) CreateServiceWithConfigs(ctx context.Context, name string, roleAttributes []string, hostV1 *HostV1ConfigData, interceptV1 *InterceptV1ConfigData) (string, error) {
	if hostV1 == nil && interceptV1 == nil {
		return c.CreateService(ctx, name, roleAttributes)
	}

	configIDs := make([]string, 0, 2)
	if hostV1 != nil {
		data := map[string]any{
			"protocol": hostV1.Protocol,
			"address":  hostV1.Address,
			"port":     hostV1.Port,
		}
		configID, err := c.createConfig(ctx, "host.v1", fmt.Sprintf("%s-host-v1", name), data)
		if err != nil {
			return "", err
		}
		configIDs = append(configIDs, configID)
	}
	if interceptV1 != nil {
		portRanges := make([]map[string]any, len(interceptV1.PortRanges))
		for i, portRange := range interceptV1.PortRanges {
			portRanges[i] = map[string]any{
				"low":  portRange.Low,
				"high": portRange.High,
			}
		}
		data := map[string]any{
			"protocols":  interceptV1.Protocols,
			"addresses":  interceptV1.Addresses,
			"portRanges": portRanges,
		}
		configID, err := c.createConfig(ctx, "intercept.v1", fmt.Sprintf("%s-intercept-v1", name), data)
		if err != nil {
			return "", c.cleanupConfigs(ctx, configIDs, err)
		}
		configIDs = append(configIDs, configID)
	}

	serviceID, err := c.createService(ctx, name, roleAttributes, configIDs)
	if err != nil {
		return "", c.cleanupConfigs(ctx, configIDs, err)
	}
	return serviceID, nil
}

func (c *Client) createService(ctx context.Context, name string, roleAttributes []string, configIDs []string) (string, error) {
	encryptionRequired := true
	params := service.NewCreateServiceParamsWithContext(ctx)
	params.Service = &rest_model.ServiceCreate{
		Name:               &name,
		RoleAttributes:     roleAttributes,
		EncryptionRequired: &encryptionRequired,
		Configs:            configIDs,
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

func (c *Client) createConfig(ctx context.Context, configTypeID, name string, data map[string]any) (string, error) {
	params := config.NewCreateConfigParamsWithContext(ctx)
	params.Config = &rest_model.ConfigCreate{
		ConfigTypeID: &configTypeID,
		Name:         &name,
		Data:         data,
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
	policy := rest_model.DialBind(policyType)
	semantic := rest_model.SemanticAnyOf
	params := service_policy.NewCreateServicePolicyParamsWithContext(ctx)
	params.Policy = &rest_model.ServicePolicyCreate{
		Name:          &name,
		Type:          &policy,
		Semantic:      &semantic,
		IdentityRoles: rest_model.Roles(identityRoles),
		ServiceRoles:  rest_model.Roles(serviceRoles),
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
	name := fmt.Sprintf("app-%s-%s", slug, id.ShortUUID())
	identityType := rest_model.IdentityTypeDevice
	isAdmin := false
	roleAttrs := rest_model.Attributes{roleAttributeApps}
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
	}

	return c.createAndEnrollIdentity(ctx, identityCreate)
}

func (c *Client) CreateAndEnrollRunnerIdentity(ctx context.Context, runnerID uuid.UUID, roleAttributes []string) (string, []byte, error) {
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

	claims, _, err := parseToken(jwt)
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

func (c *Client) fetchEnrollmentJWT(ctx context.Context, zitiIdentityID string) (string, error) {
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
		return "", fmt.Errorf("detail ziti identity: %w", err)
	}
	if detail.Payload == nil || detail.Payload.Data == nil || detail.Payload.Data.Enrollment == nil || detail.Payload.Data.Enrollment.Ott == nil {
		return "", errors.New("detail ziti identity response missing enrollment")
	}
	jwt := detail.Payload.Data.Enrollment.Ott.JWT
	if jwt == "" {
		return "", errors.New("detail ziti identity response missing enrollment jwt")
	}
	return jwt, nil
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
