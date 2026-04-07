package store

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Store struct {
	pool dbPool
}

type dbPool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func NewStore(pool dbPool) *Store {
	return &Store{pool: pool}
}

func (s *Store) InsertManagedIdentity(ctx context.Context, identity ManagedIdentity) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO managed_identities (ziti_identity_id, identity_id, identity_type, ziti_service_id) VALUES ($1, $2, $3, $4)`,
		identity.ZitiIdentityID,
		identity.IdentityID,
		identity.IdentityType,
		identity.ZitiServiceID,
	)
	if err != nil {
		return fmt.Errorf("insert managed identity: %w", err)
	}
	return nil
}

func (s *Store) DeleteManagedIdentity(ctx context.Context, zitiIdentityID string) error {
	cmd, err := s.pool.Exec(ctx, `DELETE FROM managed_identities WHERE ziti_identity_id = $1`, zitiIdentityID)
	if err != nil {
		return fmt.Errorf("delete managed identity: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrManagedIdentityNotFound
	}
	return nil
}

func (s *Store) ResolveIdentity(ctx context.Context, zitiIdentityID string) (ManagedIdentity, error) {
	row := s.pool.QueryRow(ctx, `SELECT identity_id, identity_type, ziti_service_id, created_at FROM managed_identities WHERE ziti_identity_id = $1`, zitiIdentityID)
	identity := ManagedIdentity{ZitiIdentityID: zitiIdentityID}
	if err := row.Scan(&identity.IdentityID, &identity.IdentityType, &identity.ZitiServiceID, &identity.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ManagedIdentity{}, ErrManagedIdentityNotFound
		}
		return ManagedIdentity{}, fmt.Errorf("resolve identity: %w", err)
	}
	return identity, nil
}

func (s *Store) ResolveIdentityByIdentityID(ctx context.Context, identityID uuid.UUID) (ManagedIdentity, error) {
	row := s.pool.QueryRow(ctx, `SELECT ziti_identity_id, identity_type, ziti_service_id, created_at FROM managed_identities WHERE identity_id = $1`, identityID)
	identity := ManagedIdentity{IdentityID: identityID}
	if err := row.Scan(&identity.ZitiIdentityID, &identity.IdentityType, &identity.ZitiServiceID, &identity.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ManagedIdentity{}, ErrManagedIdentityNotFound
		}
		return ManagedIdentity{}, fmt.Errorf("resolve identity by identity id: %w", err)
	}
	return identity, nil
}

func (s *Store) ListManagedIdentities(ctx context.Context, filter ListFilter, pageSize int32, cursor *PageCursor) (ListResult, error) {
	limit := normalizePageSize(pageSize)
	args := make([]any, 0, 3)
	clauses := make([]string, 0, 2)

	if filter.IdentityType != nil {
		args = append(args, *filter.IdentityType)
		clauses = append(clauses, fmt.Sprintf("identity_type = $%d", len(args)))
	}
	if cursor != nil {
		args = append(args, cursor.AfterID)
		clauses = append(clauses, fmt.Sprintf("ziti_identity_id > $%d", len(args)))
	}

	query := "SELECT ziti_identity_id, identity_id, identity_type, ziti_service_id, created_at FROM managed_identities"
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	args = append(args, limit+1)
	query += fmt.Sprintf(" ORDER BY ziti_identity_id ASC LIMIT $%d", len(args))

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return ListResult{}, fmt.Errorf("list managed identities: %w", err)
	}
	defer rows.Close()

	identities := make([]ManagedIdentity, 0)
	for rows.Next() {
		var identity ManagedIdentity
		if err := rows.Scan(&identity.ZitiIdentityID, &identity.IdentityID, &identity.IdentityType, &identity.ZitiServiceID, &identity.CreatedAt); err != nil {
			return ListResult{}, fmt.Errorf("scan managed identity: %w", err)
		}
		identities = append(identities, identity)
	}
	if err := rows.Err(); err != nil {
		return ListResult{}, fmt.Errorf("list managed identities: %w", err)
	}

	result := ListResult{Identities: identities}
	if int32(len(result.Identities)) > limit {
		nextID := result.Identities[limit-1].ZitiIdentityID
		result.Identities = result.Identities[:limit]
		result.NextCursor = &PageCursor{AfterID: nextID}
	}
	return result, nil
}

func (s *Store) InsertServiceIdentity(ctx context.Context, zitiIdentityID string, serviceType ServiceType, leaseExpiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO service_identities (ziti_identity_id, service_type, lease_expires_at) VALUES ($1, $2, $3)`,
		zitiIdentityID,
		serviceType,
		leaseExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("insert service identity: %w", err)
	}
	return nil
}

func (s *Store) ExtendServiceIdentityLease(ctx context.Context, zitiIdentityID string, leaseExpiresAt time.Time) error {
	cmd, err := s.pool.Exec(ctx, `UPDATE service_identities SET lease_expires_at = $2 WHERE ziti_identity_id = $1`,
		zitiIdentityID,
		leaseExpiresAt,
	)
	if err != nil {
		return fmt.Errorf("extend service identity lease: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrServiceIdentityNotFound
	}
	return nil
}

func (s *Store) ListExpiredServiceIdentities(ctx context.Context, gracePeriod time.Duration) ([]ServiceIdentity, error) {
	rows, err := s.pool.Query(ctx, `SELECT ziti_identity_id, service_type, lease_expires_at, created_at FROM service_identities WHERE lease_expires_at < NOW() - $1::interval ORDER BY lease_expires_at ASC`, gracePeriod)
	if err != nil {
		return nil, fmt.Errorf("list expired service identities: %w", err)
	}
	defer rows.Close()

	identities := make([]ServiceIdentity, 0)
	for rows.Next() {
		var identity ServiceIdentity
		if err := rows.Scan(&identity.ZitiIdentityID, &identity.ServiceType, &identity.LeaseExpiresAt, &identity.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan expired service identity: %w", err)
		}
		identities = append(identities, identity)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list expired service identities: %w", err)
	}
	return identities, nil
}

func (s *Store) DeleteServiceIdentity(ctx context.Context, zitiIdentityID string) error {
	cmd, err := s.pool.Exec(ctx, `DELETE FROM service_identities WHERE ziti_identity_id = $1`, zitiIdentityID)
	if err != nil {
		return fmt.Errorf("delete service identity: %w", err)
	}
	if cmd.RowsAffected() == 0 {
		return ErrServiceIdentityNotFound
	}
	return nil
}
