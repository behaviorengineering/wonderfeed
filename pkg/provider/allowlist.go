package provider

import "context"

// ScopedChannel is a provider-scoped feed source entry (provider + external_id).
type ScopedChannel struct {
	Provider   string
	ExternalID string
	Title      string
	URL        string
}

// AllowlistSynchronizer pushes host allowlist membership into provider storage.
// The first implementation (YT Zero PgAllowlistSync) applies only to ProviderYouTube
// channel ids; other provider keys are rejected at sync time.
type AllowlistSynchronizer interface {
	AddMembership(ctx context.Context, providerProfileID string, channel ScopedChannel) error
	RemoveMembership(ctx context.Context, providerProfileID string, channel ScopedChannel) error
	ReconcileAll(ctx context.Context, providerProfileID string, channels []ScopedChannel) error
}
