// Package biz contains business logic around StrategyVersion lifecycle.
//
// The Registry is the only place that mutates strategy_versions rows. Callers
// invoke ActivateOrCreate at startup with a snapshot of the current runtime
// config; the Registry figures out whether the snapshot matches the currently
// active row, a historical row (reactivate), or is entirely new (bump version).
package biz

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"

	"j_ai_trade/modules/strategy_version/model"
	"j_ai_trade/trading/engine"
)

const defaultStrategyName = "j_ai_trade"

// Registry persists StrategyVersion rows.
type Registry struct {
	db   *gorm.DB
	name string
}

func NewRegistry(db *gorm.DB) *Registry {
	return &Registry{db: db, name: defaultStrategyName}
}

// ActivateOrCreate ensures the given snapshot is reflected in strategy_versions.
//
// Semantics:
//  1. Same fingerprint as currently-active row   → return that row (no-op).
//  2. Fingerprint found but inactive (EndedAt!=nil) → clear EndedAt, close the
//     currently-active row, return the reactivated row.
//  3. Otherwise                                   → close active row, insert a
//     new row with auto-generated version label + auto-diff notes.
//
// All three paths are safe to call repeatedly at startup.
func (r *Registry) ActivateOrCreate(ctx context.Context, snapshot engine.ConfigSnapshot) (*model.StrategyVersion, error) {
	fingerprint, canon, err := snapshot.Fingerprint()
	if err != nil {
		return nil, fmt.Errorf("fingerprint snapshot: %w", err)
	}

	active, err := r.getActive(ctx)
	if err != nil {
		return nil, err
	}

	if active != nil && active.Fingerprint == fingerprint {
		return active, nil
	}

	// Look for an inactive row with the same fingerprint (reactivation).
	var reactivate model.StrategyVersion
	err = r.db.WithContext(ctx).
		Where("fingerprint = ? AND name = ?", fingerprint, r.name).
		First(&reactivate).Error
	if err == nil {
		if err := r.closeActive(ctx, active); err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		reactivate.EndedAt = nil
		reactivate.StartedAt = now
		reactivate.Notes = fmt.Sprintf("reactivated at %s (was %s)", now.Format(time.RFC3339), reactivate.Version)
		if err := r.db.WithContext(ctx).Save(&reactivate).Error; err != nil {
			return nil, fmt.Errorf("reactivate version: %w", err)
		}
		return &reactivate, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("lookup existing fingerprint: %w", err)
	}

	// Brand new snapshot → insert.
	if err := r.closeActive(ctx, active); err != nil {
		return nil, err
	}

	notes := "initial version"
	if active != nil {
		prevCanon, _ := active.Params.MarshalJSON()
		notes = engine.DiffSnapshots(prevCanon, canon)
	}

	version, err := r.nextVersionLabel(ctx)
	if err != nil {
		return nil, err
	}

	sv := &model.StrategyVersion{
		Name:        r.name,
		Version:     version,
		Fingerprint: fingerprint,
		StartedAt:   time.Now().UTC(),
		Params:      datatypes.JSON(canon),
		Notes:       notes,
	}
	if err := r.db.WithContext(ctx).Create(sv).Error; err != nil {
		return nil, fmt.Errorf("create version: %w", err)
	}
	return sv, nil
}

func (r *Registry) getActive(ctx context.Context) (*model.StrategyVersion, error) {
	var sv model.StrategyVersion
	err := r.db.WithContext(ctx).
		Where("name = ? AND ended_at IS NULL", r.name).
		Order("started_at DESC").
		First(&sv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get active version: %w", err)
	}
	return &sv, nil
}

func (r *Registry) closeActive(ctx context.Context, active *model.StrategyVersion) error {
	if active == nil {
		return nil
	}
	now := time.Now().UTC()
	active.EndedAt = &now
	if err := r.db.WithContext(ctx).Save(active).Error; err != nil {
		return fmt.Errorf("close active version: %w", err)
	}
	return nil
}

// nextVersionLabel returns YYYY-MM-DD.N where N is the 1-based count of rows
// created today. Allows inspection at a glance and is immutable once issued.
func (r *Registry) nextVersionLabel(ctx context.Context) (string, error) {
	today := time.Now().UTC().Format("2006-01-02")
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.StrategyVersion{}).
		Where("name = ? AND version LIKE ?", r.name, today+".%").
		Count(&count).Error
	if err != nil {
		return "", fmt.Errorf("count today versions: %w", err)
	}
	return fmt.Sprintf("%s.%d", today, count+1), nil
}
