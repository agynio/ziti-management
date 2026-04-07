package gc

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/agynio/ziti-management/internal/store"
	"github.com/agynio/ziti-management/internal/ziti"
)

func RunServiceIdentityGC(ctx context.Context, storeClient *store.Store, zitiClient *ziti.Client, interval time.Duration, gracePeriod time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		if err := sweepServiceIdentities(ctx, storeClient, zitiClient, gracePeriod); err != nil {
			log.Printf("service identity GC sweep failed: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func sweepServiceIdentities(ctx context.Context, storeClient *store.Store, zitiClient *ziti.Client, gracePeriod time.Duration) error {
	identities, err := storeClient.ListExpiredServiceIdentities(ctx, gracePeriod)
	if err != nil {
		return err
	}

	for _, identity := range identities {
		if err := zitiClient.DeleteIdentity(ctx, identity.ZitiIdentityID); err != nil {
			if !errors.Is(err, ziti.ErrIdentityNotFound) {
				log.Printf("failed to delete service identity %s from ziti: %v", identity.ZitiIdentityID, err)
				continue
			}
		}
		if err := storeClient.DeleteServiceIdentity(ctx, identity.ZitiIdentityID); err != nil {
			if !errors.Is(err, store.ErrServiceIdentityNotFound) {
				log.Printf("failed to delete service identity %s from database: %v", identity.ZitiIdentityID, err)
			}
			continue
		}
		log.Printf("garbage collected service identity %s", identity.ZitiIdentityID)
	}
	return nil
}
