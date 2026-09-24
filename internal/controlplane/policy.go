// Package controlplane owns Wonderfeed parent child-profile policy.
package controlplane

import (
	"strconv"
	"strings"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

// SyncStatus is the provider synchronization state for a desired policy.
type SyncStatus string

const (
	// SyncPending means the host has desired policy but provider apply has not finished.
	SyncPending SyncStatus = "sync_pending"
	// SyncSynced means the provider accepted the last desired policy.
	SyncSynced SyncStatus = "synced"
	// SyncFailed means the last provider apply failed; host desired policy remains.
	SyncFailed SyncStatus = "sync_failed"
)

// MaxDailyMinutes is the upper bound for a daily watch limit (24 hours).
const MaxDailyMinutes = 24 * 60

// ChildPolicy is the provider-neutral desired policy for one child profile.
type ChildPolicy struct {
	DailyMinutes  int       `json:"daily_minutes"`
	LocalOnly     bool      `json:"local_only"`
	HideShorts    bool      `json:"hide_shorts"`
	HideLive      bool      `json:"hide_live"`
	DownloadsOnly bool      `json:"downloads_only"`
	BedtimeStart  string    `json:"bedtime_start,omitempty"`
	BedtimeEnd    string    `json:"bedtime_end,omitempty"`
	Version       int64     `json:"version"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// ChildProfile is a host-owned child identity with desired policy and sync state.
type ChildProfile struct {
	ID                string      `json:"id"`
	Name              string      `json:"name"`
	AvatarColor       string      `json:"avatar_color"`
	ProviderProfileID string      `json:"provider_profile_id,omitempty"`
	Policy            ChildPolicy `json:"policy"`
	SyncStatus        SyncStatus  `json:"sync_status"`
	SyncError         string      `json:"sync_error,omitempty"`
	Version           int64       `json:"version"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

// DefaultChildPolicy returns fail-closed defaults for a new child profile.
func DefaultChildPolicy() ChildPolicy {
	return ChildPolicy{
		DailyMinutes:  0,
		LocalOnly:     true,
		HideShorts:    true,
		HideLive:      true,
		DownloadsOnly: false,
	}
}

// ValidatePolicy checks policy bounds and bedtime format.
func ValidatePolicy(p ChildPolicy) error {
	const op = "controlplane.ValidatePolicy"
	if p.DailyMinutes < 0 || p.DailyMinutes > MaxDailyMinutes {
		return apperr.New(apperr.CodeInvalid, op, "daily_minutes must be between 0 and 1440").
			With("daily_minutes", strconv.Itoa(p.DailyMinutes))
	}
	if err := validateBedtime(p.BedtimeStart, "bedtime_start"); err != nil {
		return err
	}
	if err := validateBedtime(p.BedtimeEnd, "bedtime_end"); err != nil {
		return err
	}
	if (p.BedtimeStart == "") != (p.BedtimeEnd == "") {
		return apperr.New(apperr.CodeInvalid, op, "bedtime_start and bedtime_end must both be set or both empty")
	}
	return nil
}

// ValidateCreateName checks a non-empty child display name.
func ValidateCreateName(name string) error {
	const op = "controlplane.ValidateCreateName"
	if strings.TrimSpace(name) == "" {
		return apperr.New(apperr.CodeInvalid, op, "name is required")
	}
	return nil
}

func validateBedtime(value, field string) error {
	const op = "controlplane.ValidatePolicy"
	if value == "" {
		return nil
	}
	if _, err := time.Parse("15:04", value); err != nil {
		return apperr.New(apperr.CodeInvalid, op, field+" must be HH:MM").With(field, value)
	}
	return nil
}
