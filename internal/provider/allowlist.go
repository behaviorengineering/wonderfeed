package provider

import "context"

// ProviderYouTube is the first supported video provider key.
const ProviderYouTube = "youtube"

// ScopedChannel is a provider-scoped feed source entry.
type ScopedChannel struct {
	Provider   string
	ExternalID string
	Title      string
	URL        string
}

// AllowlistSynchronizer pushes host allowlist membership into provider storage.
type AllowlistSynchronizer interface {
	AddMembership(ctx context.Context, providerProfileID string, channel ScopedChannel) error
	RemoveMembership(ctx context.Context, providerProfileID string, channel ScopedChannel) error
	ReconcileAll(ctx context.Context, providerProfileID string, channels []ScopedChannel) error
}
