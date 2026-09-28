// Package controlplane exposes provider-neutral host domain types for Wonderfeed clients and adapters.
package controlplane

import (
	"time"

	"github.com/behaviorengineering/wonderfeed/pkg/provider"
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
	ID                string             `json:"id"`
	Name              string             `json:"name"`
	AvatarColor       string             `json:"avatar_color"`
	ProviderProfileID string             `json:"provider_profile_id,omitempty"`
	AllowlistVersion  int64              `json:"allowlist_version"`
	Allowlist         []AllowlistChannel `json:"allowlist,omitempty"`
	Policy            ChildPolicy        `json:"policy"`
	SyncStatus        SyncStatus         `json:"sync_status"`
	SyncError         string             `json:"sync_error,omitempty"`
	Version           int64              `json:"version"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

// AllowlistChannel is a parent-approved provider feed source for a child.
type AllowlistChannel struct {
	Provider   string    `json:"provider"`
	ExternalID string    `json:"external_id"`
	Title      string    `json:"title,omitempty"`
	URL        string    `json:"url,omitempty"`
	AddedAt    time.Time `json:"added_at"`
}

// ToProviderScoped converts a host allowlist entry for provider sync.
func (c AllowlistChannel) ToProviderScoped() provider.ScopedChannel {
	return provider.ScopedChannel{
		Provider:   c.Provider,
		ExternalID: c.ExternalID,
		Title:      c.Title,
		URL:        c.URL,
	}
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
