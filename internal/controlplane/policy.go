// Package controlplane owns Wonderfeed parent child-profile policy orchestration.
package controlplane

import (
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
	cp "github.com/behaviorengineering/wonderfeed/pkg/controlplane"
	"github.com/behaviorengineering/wonderfeed/pkg/provider"
)

var youtubeChannelID = regexp.MustCompile(`^UC[\w-]{22}$`)

// NormalizeAllowlistEntry validates one provider-scoped allowlist entry.
func NormalizeAllowlistEntry(entry cp.AllowlistChannel) (cp.AllowlistChannel, error) {
	const op = "controlplane.NormalizeAllowlistEntry"
	providerKey := strings.ToLower(strings.TrimSpace(entry.Provider))
	if providerKey == "" {
		providerKey = provider.ProviderYouTube
	}
	externalID := strings.TrimSpace(entry.ExternalID)
	if externalID == "" {
		return cp.AllowlistChannel{}, apperr.New(apperr.CodeInvalid, op, "external_id is required")
	}
	if len(externalID) > 128 || strings.ContainsAny(externalID, " \t\r\n/") {
		return cp.AllowlistChannel{}, apperr.New(apperr.CodeInvalid, op, "external_id is invalid").
			With("external_id", externalID)
	}
	if err := validateProviderExternalID(providerKey, externalID); err != nil {
		return cp.AllowlistChannel{}, err
	}
	channelURL := strings.TrimSpace(entry.URL)
	if channelURL == "" {
		channelURL = defaultChannelURL(providerKey, externalID)
	} else {
		parsed, err := url.Parse(channelURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return cp.AllowlistChannel{}, apperr.New(apperr.CodeInvalid, op, "url must be an HTTP or HTTPS URL").
				With("external_id", externalID)
		}
	}
	return cp.AllowlistChannel{
		Provider:   providerKey,
		ExternalID: externalID,
		Title:      strings.TrimSpace(entry.Title),
		URL:        channelURL,
	}, nil
}

func validateProviderExternalID(providerKey, externalID string) error {
	const op = "controlplane.NormalizeAllowlistEntry"
	switch providerKey {
	case provider.ProviderYouTube:
		if !youtubeChannelID.MatchString(externalID) {
			return apperr.New(apperr.CodeInvalid, op, "youtube external_id must be a UC channel id").
				With("external_id", externalID)
		}
		return nil
	default:
		// Future providers (netflix, pbs, and similar) use opaque stable ids until a provider adapter adds stricter rules.
		if len(externalID) < 2 {
			return apperr.New(apperr.CodeInvalid, op, "external_id is too short").With("provider", providerKey)
		}
		return nil
	}
}

func defaultChannelURL(providerKey, externalID string) string {
	switch providerKey {
	case provider.ProviderYouTube:
		return "https://www.youtube.com/channel/" + url.PathEscape(externalID)
	default:
		return ""
	}
}

// ValidatePolicy checks policy bounds and bedtime format.
func ValidatePolicy(p cp.ChildPolicy) error {
	const op = "controlplane.ValidatePolicy"
	if p.DailyMinutes < 0 || p.DailyMinutes > cp.MaxDailyMinutes {
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
