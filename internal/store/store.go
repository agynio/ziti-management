package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

func (s *Store) InsertManagedIdentity(ctx context.Context, identity ManagedIdentity) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO managed_identities (ziti_identity_id, identity_id, identity_type, tenant_id) VALUES ($1, $2, $3, $4)`,
		identity.ZitiIdentityID,
		identity.IdentityID,
		identity.IdentityType,
		identity.TenantID,
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
	row := s.pool.QueryRow(ctx, `SELECT identity_id, identity_type, tenant_id, created_at FROM managed_identities WHERE ziti_identity_id = $1`, zitiIdentityID)
	identity := ManagedIdentity{ZitiIdentityID: zitiIdentityID}
	if err := row.Scan(&identity.IdentityID, &identity.IdentityType, &identity.TenantID, &identity.CreatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ManagedIdentity{}, ErrManagedIdentityNotFound
		}
		return ManagedIdentity{}, fmt.Errorf("resolve identity: %w", err)
	}
	return identity, nil
}

func (s *Store) ListManagedIdentities(ctx context.Context, filter ListFilter, pageSize int32, cursor *PageCursor) (ListResult, error) {
	limit := normalizePageSize(pageSize)
	args := make([]any, 0, 4)
	clauses := make([]string, 0, 3)

	if filter.IdentityType != nil {
		args = append(args, *filter.IdentityType)
		clauses = append(clauses, fmt.Sprintf("identity_type = $%d", len(args)))
	}
	if filter.TenantID != nil {
		args = append(args, *filter.TenantID)
		clauses = append(clauses, fmt.Sprintf("tenant_id = $%d", len(args)))
	}
	if cursor != nil {
		args = append(args, cursor.AfterID)
		clauses = append(clauses, fmt.Sprintf("ziti_identity_id > $%d", len(args)))
	}

	query := "SELECT ziti_identity_id, identity_id, identity_type, tenant_id, created_at FROM managed_identities"
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
		if err := rows.Scan(&identity.ZitiIdentityID, &identity.IdentityID, &identity.IdentityType, &identity.TenantID, &identity.CreatedAt); err != nil {
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
