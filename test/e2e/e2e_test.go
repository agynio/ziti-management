//go:build e2e

package e2e

import (
	"context"
	"testing"
	"time"

	zitimanagementv1 "github.com/agynio/ziti-management/.gen/go/agynio/api/ziti_management/v1"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func TestZitiManagementServiceE2E(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	conn, err := grpc.NewClient(zitiManagementAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	client := zitimanagementv1.NewZitiManagementServiceClient(conn)

	agentID := uuid.NewString()

	createResp, err := client.CreateAgentIdentity(ctx, &zitimanagementv1.CreateAgentIdentityRequest{
		AgentId: agentID,
	})
	require.NoError(t, err)
	require.NotNil(t, createResp)
	require.NotEmpty(t, createResp.GetZitiIdentityId())
	require.NotEmpty(t, createResp.GetEnrollmentJwt())

	deleted := false
	t.Cleanup(func() {
		if deleted {
			return
		}
		cleanupCtx, cancelCleanup := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancelCleanup()
		_, err := client.DeleteIdentity(cleanupCtx, &zitimanagementv1.DeleteIdentityRequest{
			ZitiIdentityId: createResp.GetZitiIdentityId(),
		})
		if err != nil {
			t.Logf("cleanup delete identity: %v", err)
		}
	})

	resolveResp, err := client.ResolveIdentity(ctx, &zitimanagementv1.ResolveIdentityRequest{
		ZitiIdentityId: createResp.GetZitiIdentityId(),
	})
	require.NoError(t, err)
	require.Equal(t, agentID, resolveResp.GetIdentityId())
	require.Equal(t, zitimanagementv1.IdentityType_IDENTITY_TYPE_AGENT, resolveResp.GetIdentityType())

	pageToken := ""
	found := false
	for i := 0; i < 10; i++ {
		listResp, err := client.ListManagedIdentities(ctx, &zitimanagementv1.ListManagedIdentitiesRequest{
			IdentityType: zitimanagementv1.IdentityType_IDENTITY_TYPE_AGENT,
			PageSize:     50,
			PageToken:    pageToken,
		})
		require.NoError(t, err)

		for _, identity := range listResp.GetIdentities() {
			if identity.GetZitiIdentityId() == createResp.GetZitiIdentityId() {
				found = true
				require.Equal(t, agentID, identity.GetIdentityId())
				require.Equal(t, zitimanagementv1.IdentityType_IDENTITY_TYPE_AGENT, identity.GetIdentityType())
				require.NotNil(t, identity.GetCreatedAt())
				break
			}
		}

		if found || listResp.GetNextPageToken() == "" {
			break
		}
		pageToken = listResp.GetNextPageToken()
	}
	require.True(t, found)

	_, err = client.DeleteIdentity(ctx, &zitimanagementv1.DeleteIdentityRequest{
		ZitiIdentityId: createResp.GetZitiIdentityId(),
	})
	require.NoError(t, err)
	deleted = true

	_, err = client.ResolveIdentity(ctx, &zitimanagementv1.ResolveIdentityRequest{
		ZitiIdentityId: createResp.GetZitiIdentityId(),
	})
	require.Error(t, err)
	statusErr, ok := status.FromError(err)
	require.True(t, ok)
	require.Equal(t, codes.NotFound, statusErr.Code())
}
