package server

import (
	"context"
	"errors"
	"testing"
	"time"

	zitimanagementv1 "github.com/agynio/ziti-management/.gen/go/agynio/api/ziti_management/v1"
	"github.com/agynio/ziti-management/internal/store"
	"github.com/agynio/ziti-management/internal/ziti"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestDeleteIdentityRemovesBothSides(t *testing.T) {
	storeClient := &deleteAppIdentityStore{}
	zitiClient := &fakeZitiClient{}
	server := New(storeClient, zitiClient, time.Minute, false)

	if _, err := server.DeleteIdentity(context.Background(), &zitimanagementv1.DeleteIdentityRequest{
		ZitiIdentityId: "ziti-1",
	}); err != nil {
		t.Fatalf("delete identity: %v", err)
	}
	if len(storeClient.deleteCalls) != 1 || storeClient.deleteCalls[0] != "ziti-1" {
		t.Fatalf("managed identity deletes = %v", storeClient.deleteCalls)
	}
	if len(zitiClient.deleteIdentityIDs) != 1 {
		t.Fatalf("ziti identity deletes = %v", zitiClient.deleteIdentityIDs)
	}
}

// The caller is a reconciler that retries until the identity is gone. Reporting
// an already-removed record as NotFound leaves it retrying forever — which is
// how a sandbox start stalled behind a compensating delete.
func TestDeleteIdentityIsIdempotent(t *testing.T) {
	storeClient := &deleteAppIdentityStore{deleteErr: store.ErrManagedIdentityNotFound}
	zitiClient := &fakeZitiClient{}
	server := New(storeClient, zitiClient, time.Minute, false)

	if _, err := server.DeleteIdentity(context.Background(), &zitimanagementv1.DeleteIdentityRequest{
		ZitiIdentityId: "ziti-gone",
	}); err != nil {
		t.Fatalf("delete identity with no record: %v", err)
	}
	// The record being gone says nothing about the controller, so the Ziti side
	// is still cleaned up.
	if len(zitiClient.deleteIdentityIDs) != 1 {
		t.Fatalf("ziti identity deletes = %v", zitiClient.deleteIdentityIDs)
	}
}

func TestDeleteIdentityToleratesAnAbsentZitiIdentity(t *testing.T) {
	storeClient := &deleteAppIdentityStore{}
	zitiClient := &fakeZitiClient{deleteIdentityErr: ziti.ErrIdentityNotFound}
	server := New(storeClient, zitiClient, time.Minute, false)

	if _, err := server.DeleteIdentity(context.Background(), &zitimanagementv1.DeleteIdentityRequest{
		ZitiIdentityId: "ziti-2",
	}); err != nil {
		t.Fatalf("delete identity: %v", err)
	}
}

// A store failure that is not "already gone" still has to surface, or a real
// outage would look like successful cleanup.
func TestDeleteIdentitySurfacesOtherStoreFailures(t *testing.T) {
	storeClient := &deleteAppIdentityStore{deleteErr: errors.New("connection refused")}
	server := New(storeClient, &fakeZitiClient{}, time.Minute, false)

	_, err := server.DeleteIdentity(context.Background(), &zitimanagementv1.DeleteIdentityRequest{
		ZitiIdentityId: "ziti-3",
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("code = %v, want Internal", status.Code(err))
	}
}

func TestDeleteIdentityRequiresAnID(t *testing.T) {
	server := New(&deleteAppIdentityStore{}, &fakeZitiClient{}, time.Minute, false)

	_, err := server.DeleteIdentity(context.Background(), &zitimanagementv1.DeleteIdentityRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", status.Code(err))
	}
}
