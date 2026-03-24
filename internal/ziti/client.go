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

	"github.com/agynio/ziti-management/internal/id"
	"github.com/go-openapi/runtime"
	"github.com/google/uuid"
	"github.com/openziti/edge-api/rest_management_api_client/identity"
	"github.com/openziti/edge-api/rest_management_api_client/service"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/ziti"
	"github.com/openziti/sdk-golang/ziti/enroll"
)

var ErrIdentityNotFound = errors.New("identity not found")
var ErrServiceNotFound = errors.New("service not found")

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

type Client struct {
	identity identityService
	service  serviceService
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
	return &Client{identity: client.Identity, service: client.Service}, nil
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

func (c *Client) CreateAgentIdentity(ctx context.Context, agentID uuid.UUID) (string, string, error) {
	externalID := agentID.String()
	existingIdentity, err := c.findIdentityByExternalID(ctx, externalID)
	if err != nil {
		return "", "", err
	}
	if existingIdentity != nil {
		if err := c.DeleteIdentity(ctx, *existingIdentity.ID); err != nil && !errors.Is(err, ErrIdentityNotFound) {
			return "", "", fmt.Errorf("delete existing ziti identity: %w", err)
		}
	}

	name := fmt.Sprintf("agent-%s-%s", agentID.String(), id.ShortUUID())
	identityType := rest_model.IdentityTypeDevice
	isAdmin := false
	roleAttrs := rest_model.Attributes{
		"agents",
		fmt.Sprintf("agent-%s", agentID.String()),
	}
	params := identity.NewCreateIdentityParamsWithContext(ctx)
	params.Identity = &rest_model.IdentityCreate{
		Name:           &name,
		Type:           &identityType,
		IsAdmin:        &isAdmin,
		RoleAttributes: &roleAttrs,
		ExternalID:     &externalID,
		Enrollment:     &rest_model.IdentityCreateEnrollment{Ott: true},
	}

	created, err := c.identity.CreateIdentity(params, nil)
	if err != nil {
		return "", "", fmt.Errorf("create ziti identity: %w", err)
	}
	if created.Payload == nil || created.Payload.Data == nil {
		return "", "", errors.New("create ziti identity response missing data")
	}
	zitiID := created.Payload.Data.ID
	if zitiID == "" {
		return "", "", errors.New("create ziti identity response missing id")
	}

	jwt, err := c.fetchEnrollmentJWT(ctx, zitiID)
	if err != nil {
		return "", "", err
	}
	return zitiID, jwt, nil
}

func (c *Client) findIdentityByExternalID(ctx context.Context, externalID string) (*rest_model.IdentityDetail, error) {
	filter := fmt.Sprintf("externalId = \"%s\"", externalID)
	params := identity.NewListIdentitiesParamsWithContext(ctx)
	params.Filter = &filter
	listed, err := c.identity.ListIdentities(params, nil)
	if err != nil {
		return nil, fmt.Errorf("list ziti identities: %w", err)
	}
	if listed.Payload == nil || listed.Payload.Data == nil {
		return nil, errors.New("list ziti identities response missing data")
	}
	if len(listed.Payload.Data) == 0 {
		return nil, nil
	}
	identityDetail := listed.Payload.Data[0]
	if identityDetail == nil || identityDetail.ID == nil || *identityDetail.ID == "" {
		return nil, errors.New("list ziti identities response missing id")
	}
	return identityDetail, nil
}

func (c *Client) CreateService(ctx context.Context, name string, roleAttributes []string) (string, error) {
	encryptionRequired := true
	params := service.NewCreateServiceParamsWithContext(ctx)
	params.Service = &rest_model.ServiceCreate{
		Name:               &name,
		RoleAttributes:     roleAttributes,
		EncryptionRequired: &encryptionRequired,
	}

	created, err := c.service.CreateService(params, nil)
	if err != nil {
		return "", fmt.Errorf("create ziti service: %w", err)
	}
	if created.Payload == nil || created.Payload.Data == nil {
		return "", errors.New("create ziti service response missing data")
	}
	serviceID := created.Payload.Data.ID
	if serviceID == "" {
		return "", errors.New("create ziti service response missing id")
	}
	return serviceID, nil
}

func (c *Client) DeleteService(ctx context.Context, serviceID string) error {
	params := service.NewDeleteServiceParamsWithContext(ctx)
	params.ID = serviceID
	_, err := c.service.DeleteService(params, nil)
	if err == nil {
		return nil
	}
	var notFound *service.DeleteServiceNotFound
	if errors.As(err, &notFound) {
		return ErrServiceNotFound
	}
	return fmt.Errorf("delete ziti service: %w", err)
}

func (c *Client) CreateAndEnrollServiceIdentity(ctx context.Context, name string, roleAttributes []string) (string, []byte, error) {
	identityType := rest_model.IdentityTypeDevice
	isAdmin := false
	attrs := rest_model.Attributes(roleAttributes)
	params := identity.NewCreateIdentityParamsWithContext(ctx)
	params.Identity = &rest_model.IdentityCreate{
		Name:           &name,
		Type:           &identityType,
		IsAdmin:        &isAdmin,
		RoleAttributes: &attrs,
		Enrollment:     &rest_model.IdentityCreateEnrollment{Ott: true},
	}

	created, err := c.identity.CreateIdentity(params, nil)
	if err != nil {
		return "", nil, fmt.Errorf("create ziti identity: %w", err)
	}
	if created.Payload == nil || created.Payload.Data == nil {
		return "", nil, errors.New("create ziti identity response missing data")
	}
	zitiID := created.Payload.Data.ID
	if zitiID == "" {
		return "", nil, errors.New("create ziti identity response missing id")
	}

	identityJSON, err := c.enrollIdentity(ctx, zitiID)
	if err != nil {
		return "", nil, c.cleanupServiceIdentity(ctx, zitiID, err)
	}
	return zitiID, identityJSON, nil
}

func (c *Client) CreateAndEnrollAppIdentity(ctx context.Context, appID uuid.UUID, slug string) (string, []byte, string, error) {
	name := fmt.Sprintf("app-%s-%s", slug, id.ShortUUID())
	identityType := rest_model.IdentityTypeDevice
	isAdmin := false
	roleAttrs := rest_model.Attributes{"apps"}
	externalID := appID.String()
	params := identity.NewCreateIdentityParamsWithContext(ctx)
	params.Identity = &rest_model.IdentityCreate{
		Name:           &name,
		Type:           &identityType,
		IsAdmin:        &isAdmin,
		RoleAttributes: &roleAttrs,
		ExternalID:     &externalID,
		Enrollment:     &rest_model.IdentityCreateEnrollment{Ott: true},
	}

	created, err := c.identity.CreateIdentity(params, nil)
	if err != nil {
		return "", nil, "", fmt.Errorf("create ziti identity: %w", err)
	}
	if created.Payload == nil || created.Payload.Data == nil {
		return "", nil, "", errors.New("create ziti identity response missing data")
	}
	zitiID := created.Payload.Data.ID
	if zitiID == "" {
		return "", nil, "", errors.New("create ziti identity response missing id")
	}

	serviceID, err := c.CreateService(ctx, name, []string{"app-services"})
	if err != nil {
		return "", nil, "", c.CleanupAppResources(ctx, zitiID, "", err)
	}

	identityJSON, err := c.enrollIdentity(ctx, zitiID)
	if err != nil {
		return "", nil, "", c.CleanupAppResources(ctx, zitiID, serviceID, err)
	}
	return zitiID, identityJSON, serviceID, nil
}

func (c *Client) enrollIdentity(ctx context.Context, zitiIdentityID string) ([]byte, error) {
	jwt, err := c.fetchEnrollmentJWT(ctx, zitiIdentityID)
	if err != nil {
		return nil, err
	}

	claims, _, err := enroll.ParseToken(jwt)
	if err != nil {
		return nil, fmt.Errorf("parse enrollment token: %w", err)
	}

	var keyAlg ziti.KeyAlgVar
	if err := keyAlg.Set("EC"); err != nil {
		return nil, fmt.Errorf("set key algorithm: %w", err)
	}
	config, err := enroll.Enroll(enroll.EnrollmentFlags{Token: claims, KeyAlg: keyAlg})
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
	detail, err := c.identity.DetailIdentity(detailParams, nil)
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

func (c *Client) CleanupAppResources(ctx context.Context, zitiIdentityID, zitiServiceID string, err error) error {
	identityErr := c.DeleteIdentity(ctx, zitiIdentityID)
	if identityErr != nil && !errors.Is(identityErr, ErrIdentityNotFound) {
		err = fmt.Errorf("%w; cleanup identity failed: %w", err, identityErr)
	}
	if zitiServiceID == "" {
		return err
	}
	serviceErr := c.DeleteService(ctx, zitiServiceID)
	if serviceErr != nil && !errors.Is(serviceErr, ErrServiceNotFound) {
		err = fmt.Errorf("%w; cleanup service failed: %w", err, serviceErr)
	}
	return err
}

func (c *Client) DeleteIdentity(ctx context.Context, zitiIdentityID string) error {
	params := identity.NewDeleteIdentityParamsWithContext(ctx)
	params.ID = zitiIdentityID
	_, err := c.identity.DeleteIdentity(params, nil)
	if err == nil {
		return nil
	}
	var notFound *identity.DeleteIdentityNotFound
	if errors.As(err, &notFound) {
		return ErrIdentityNotFound
	}
	return fmt.Errorf("delete ziti identity: %w", err)
}
