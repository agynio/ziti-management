package store

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrManagedIdentityNotFound = errors.New("managed identity not found")
var ErrServiceIdentityNotFound = errors.New("service identity not found")

type IdentityType int16

const (
	IdentityTypeUnspecified IdentityType = 0
	IdentityTypeAgent       IdentityType = 1
	IdentityTypeRunner      IdentityType = 2
	IdentityTypeChannel     IdentityType = 3
)

type ServiceType int16

const (
	ServiceTypeUnspecified  ServiceType = 0
	ServiceTypeGateway      ServiceType = 1
	ServiceTypeOrchestrator ServiceType = 2
	ServiceTypeRunner       ServiceType = 3
)

type ManagedIdentity struct {
	ZitiIdentityID string
	IdentityID     uuid.UUID
	IdentityType   IdentityType
	TenantID       uuid.UUID
	CreatedAt      time.Time
}

type ServiceIdentity struct {
	ZitiIdentityID string
	ServiceType    ServiceType
	LeaseExpiresAt time.Time
	CreatedAt      time.Time
}

type ListFilter struct {
	IdentityType *IdentityType
	TenantID     *uuid.UUID
}

type PageCursor struct {
	AfterID string
}

type ListResult struct {
	Identities []ManagedIdentity
	NextCursor *PageCursor
}
