package stores

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/zoobz-io/astql"
	"github.com/zoobz-io/sum"

	"github.com/zoobz-io/barbara/database/models"
	"github.com/zoobz-io/barbara/events"
	"github.com/zoobz-io/barbara/internal/auth"
)

// ErrAppNameTaken is returned when creating or renaming an app to a name another
// app in the tenant already holds (the per-tenant unique index).
var ErrAppNameTaken = errors.New("an app with that name already exists")

// ErrAppHasReleases is returned when deleting an app that has any release.
// Releases are permanent, so the app that owns them is too.
var ErrAppHasReleases = errors.New("app has releases; releases are never deleted")

// Apps is the data-access layer for the app — the release unit. Every method is
// scoped to the tenant carried in the request context. releases is a read-only
// handle used only to guard the delete; the release primitives live in their own
// store.
type Apps struct {
	*sum.Database[models.App]
	releases *sum.Database[models.Release]
}

// NewApps creates an apps store.
func NewApps(db *sqlx.DB, renderer astql.Renderer) *Apps {
	return &Apps{
		Database: sum.NewDatabase[models.App](db, "apps", renderer),
		releases: sum.NewDatabase[models.Release](db, "releases", renderer),
	}
}

// Create inserts a new app with the given name for the request's tenant. The
// name must be unique per tenant; a duplicate returns ErrAppNameTaken.
func (s *Apps) Create(ctx context.Context, name string) (*models.App, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	// Timestamps are set here, not left to the column default: the insert carries
	// every non-primary-key field, so a zero time.Time would override the default.
	// The id is DB-generated (primary key, omitted); current_release_id starts null.
	now := time.Now()
	app := &models.App{
		TenantID:  tenantID,
		Name:      name,
		CreatedAt: now,
		UpdatedAt: now,
	}
	created, err := s.Insert().Exec(ctx, app)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAppNameTaken
		}
		return nil, fmt.Errorf("creating app: %w", err)
	}
	events.App.Created.Emit(ctx, events.AppCreatedEvent{
		AppID: created.ID, TenantID: tenantID, Name: created.Name,
	})
	return created, nil
}

// Get retrieves an app by ID, scoped to the request's tenant.
func (s *Apps) Get(ctx context.Context, id string) (*models.App, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	app, err := s.Select().
		Where("id", "=", "id").
		Where("tenant_id", "=", "tenant_id").
		Exec(ctx, map[string]any{"id": id, "tenant_id": tenantID})
	if err != nil {
		return nil, err
	}
	return app, nil
}

// List returns the tenant's apps, oldest first, paginated.
func (s *Apps) List(ctx context.Context, limit, offset int) ([]*models.App, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	apps, err := s.Query().
		Where("tenant_id", "=", "tenant_id").
		OrderBy("created_at", "asc").
		Limit(limit).
		Offset(offset).
		Exec(ctx, map[string]any{"tenant_id": tenantID})
	if err != nil {
		return nil, fmt.Errorf("listing apps: %w", err)
	}
	return apps, nil
}

// Rename changes an app's name. The new name must be unique per tenant; a
// duplicate returns ErrAppNameTaken. Returns ErrNotFound if the app does not
// exist for the tenant.
func (s *Apps) Rename(ctx context.Context, id, newName string) (*models.App, error) {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return nil, err
	}
	app, err := s.Modify().
		Set("name", "name").
		Set("updated_at", "updated_at").
		Where("id", "=", "id").
		Where("tenant_id", "=", "tenant_id").
		Exec(ctx, map[string]any{
			"name":       newName,
			"updated_at": time.Now(),
			"id":         id,
			"tenant_id":  tenantID,
		})
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrAppNameTaken
		}
		return nil, fmt.Errorf("renaming app: %w", err)
	}
	events.App.Renamed.Emit(ctx, events.AppRenamedEvent{
		AppID: app.ID, TenantID: tenantID, Name: app.Name,
	})
	return app, nil
}

// Delete removes an app that has no release. An app with any release is refused
// with ErrAppHasReleases; a missing one with ErrNotFound. The release count is
// the friendly guard; the releases.app_id foreign key is the backstop if a cut
// races this delete.
func (s *Apps) Delete(ctx context.Context, id string) error {
	tenantID, err := auth.RequireTenant(ctx)
	if err != nil {
		return err
	}
	count, err := s.releases.Count().
		Where("app_id", "=", "app_id").
		Where("tenant_id", "=", "tenant_id").
		Exec(ctx, map[string]any{"app_id": id, "tenant_id": tenantID})
	if err != nil {
		return fmt.Errorf("counting releases: %w", err)
	}
	if count > 0 {
		return ErrAppHasReleases
	}
	n, err := s.Remove().
		Where("id", "=", "id").
		Where("tenant_id", "=", "tenant_id").
		Exec(ctx, map[string]any{"id": id, "tenant_id": tenantID})
	if err != nil {
		if isForeignKeyViolation(err) {
			return ErrAppHasReleases // a release was cut between the count and the delete
		}
		return fmt.Errorf("deleting app: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	events.App.Deleted.Emit(ctx, events.AppDeletedEvent{AppID: id, TenantID: tenantID})
	return nil
}

// isUniqueViolation reports whether err is a Postgres unique-constraint
// violation (a duplicate key).
func isUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

// isForeignKeyViolation reports whether err is a Postgres foreign-key-constraint
// violation (a referenced row still exists).
func isForeignKeyViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23503"
}
