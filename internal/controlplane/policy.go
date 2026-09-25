// Package controlplane owns Wonderfeed parent child-profile policy.
package controlplane

import (
	"net/url"
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

// AllowlistChannel is a parent-approved channel in a child's feed source.
type AllowlistChannel struct {
	ChannelID string    `json:"channel_id"`
	Title     string    `json:"title,omitempty"`
	URL       string    `json:"url,omitempty"`
	AddedAt   time.Time `json:"added_at"`
}

// NormalizeAllowlist validates and canonicalizes a replacement allowlist.
func NormalizeAllowlist(in []AllowlistChannel) ([]AllowlistChannel, error) {
	const op = "controlplane.NormalizeAllowlist"
	if len(in) > 10000 {
		return nil, apperr.New(apperr.CodeInvalid, op, "allowlist cannot contain more than 10000 channels")
	}
	out := make([]AllowlistChannel, 0, len(in))
	seen := make(map[string]struct{}, len(in))
	for _, entry := range in {
		channelID := strings.TrimSpace(entry.ChannelID)
		if channelID == "" {
			return nil, apperr.New(apperr.CodeInvalid, op, "channel_id is required")
		}
		if len(channelID) > 128 || strings.ContainsAny(channelID, " \t\r\n/") {
			return nil, apperr.New(apperr.CodeInvalid, op, "channel_id is invalid").With("channel_id", channelID)
		}
		if _, ok := seen[channelID]; ok {
			return nil, apperr.New(apperr.CodeInvalid, op, "allowlist contains duplicate channel_id").
				With("channel_id", channelID)
		}
		seen[channelID] = struct{}{}
		channelURL := strings.TrimSpace(entry.URL)
		if channelURL == "" {
			channelURL = "https://www.youtube.com/channel/" + url.PathEscape(channelID)
		} else {
			parsed, err := url.Parse(channelURL)
			if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
				return nil, apperr.New(apperr.CodeInvalid, op, "url must be an HTTP or HTTPS URL").
					With("channel_id", channelID)
			}
		}
		out = append(out, AllowlistChannel{
			ChannelID: channelID,
			Title:     strings.TrimSpace(entry.Title),
			URL:       channelURL,
		})
	}
	return out, nil
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
