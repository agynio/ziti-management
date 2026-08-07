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
	IdentityTypeApp         IdentityType = 5 // Matches agynio.api.identity.v1.IdentityType enum values.
	// An agent workload authenticates as the instance it runs, not as the class:
	// the instance owns the inbox, the volumes and the runner pinning.
	IdentityTypeAgentInstance IdentityType = 6
	IdentityTypeSandbox       IdentityType = 7
)

type ServiceType int16

const (
	ServiceTypeUnspecified   ServiceType = 0
	ServiceTypeGateway       ServiceType = 1
	ServiceTypeOrchestrator  ServiceType = 2
	ServiceTypeLLMProxy      ServiceType = 4
	ServiceTypeTracing       ServiceType = 5
	ServiceTypeRunners       ServiceType = 6
	ServiceTypeEgressGateway ServiceType = 7
	ServiceTypeTerminalProxy ServiceType = 8
)

type ManagedIdentity struct {
	ZitiIdentityID string
	IdentityID     uuid.UUID
	WorkloadID     *uuid.UUID
	IdentityType   IdentityType
	ZitiServiceID  *string
	CreatedAt      time.Time
	// Workload identities only: the agent class this instance runs, and the
	// environment it runs. AgentID is nil for a sandbox, which has no agent.
	AgentID       *uuid.UUID
	EnvironmentID *uuid.UUID
}

type ServiceIdentity struct {
	ZitiIdentityID string
	ServiceType    ServiceType
	LeaseExpiresAt time.Time
	CreatedAt      time.Time
}

type ListFilter struct {
	IdentityType *IdentityType
}

type PageCursor struct {
	AfterID string
}

type ListResult struct {
	Identities []ManagedIdentity
	NextCursor *PageCursor
}
