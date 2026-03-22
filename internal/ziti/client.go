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
	"github.com/google/uuid"
	"github.com/openziti/edge-api/rest_management_api_client"
	"github.com/openziti/edge-api/rest_management_api_client/identity"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/ziti"
	"github.com/openziti/sdk-golang/ziti/enroll"
)

var ErrIdentityNotFound = errors.New("identity not found")

type Client struct {
	client *rest_management_api_client.ZitiEdgeManagement
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
	return &Client{client: client}, nil
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
	name := fmt.Sprintf("agent-%s-%s", agentID.String(), id.ShortUUID())
	identityType := rest_model.IdentityTypeDevice
	isAdmin := false
	roleAttrs := rest_model.Attributes{
		"agents",
		fmt.Sprintf("agent-%s", agentID.String()),
	}
	externalID := agentID.String()
	params := identity.NewCreateIdentityParamsWithContext(ctx)
	params.Identity = &rest_model.IdentityCreate{
		Name:           &name,
		Type:           &identityType,
		IsAdmin:        &isAdmin,
		RoleAttributes: &roleAttrs,
		ExternalID:     &externalID,
		Enrollment:     &rest_model.IdentityCreateEnrollment{Ott: true},
	}

	created, err := c.client.Identity.CreateIdentity(params, nil)
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

	created, err := c.client.Identity.CreateIdentity(params, nil)
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

	jwt, err := c.fetchEnrollmentJWT(ctx, zitiID)
	if err != nil {
		return "", nil, c.cleanupServiceIdentity(ctx, zitiID, err)
	}

	claims, _, err := enroll.ParseToken(jwt)
	if err != nil {
		return "", nil, c.cleanupServiceIdentity(ctx, zitiID, fmt.Errorf("parse enrollment token: %w", err))
	}

	var keyAlg ziti.KeyAlgVar
	if err := keyAlg.Set("EC"); err != nil {
		return "", nil, c.cleanupServiceIdentity(ctx, zitiID, fmt.Errorf("set key algorithm: %w", err))
	}
	cfg, err := enroll.Enroll(enroll.EnrollmentFlags{Token: claims, KeyAlg: keyAlg})
	if err != nil {
		return "", nil, c.cleanupServiceIdentity(ctx, zitiID, fmt.Errorf("enroll identity: %w", err))
	}
	identityJSON, err := json.Marshal(cfg)
	if err != nil {
		return "", nil, c.cleanupServiceIdentity(ctx, zitiID, fmt.Errorf("marshal identity json: %w", err))
	}
	return zitiID, identityJSON, nil
}

func (c *Client) fetchEnrollmentJWT(ctx context.Context, zitiIdentityID string) (string, error) {
	detailParams := identity.NewDetailIdentityParamsWithContext(ctx)
	detailParams.ID = zitiIdentityID
	detail, err := c.client.Identity.DetailIdentity(detailParams, nil)
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
	_, err := c.client.Identity.DeleteIdentity(params, nil)
	if err == nil {
		return nil
	}
	var notFound *identity.DeleteIdentityNotFound
	if errors.As(err, &notFound) {
		return ErrIdentityNotFound
	}
	return fmt.Errorf("delete ziti identity: %w", err)
}
