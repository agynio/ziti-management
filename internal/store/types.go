package store

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var ErrManagedIdentityNotFound = errors.New("managed identity not found")

type IdentityType int16

const (
	IdentityTypeUnspecified IdentityType = 0
	IdentityTypeAgent       IdentityType = 1
	IdentityTypeRunner      IdentityType = 2
	IdentityTypeChannel     IdentityType = 3
)

type ManagedIdentity struct {
	ZitiIdentityID string
	IdentityID     uuid.UUID
	IdentityType   IdentityType
	TenantID       uuid.UUID
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
