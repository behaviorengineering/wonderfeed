// Package provider defines provider-neutral seams for Wonderfeed adapters.
package provider

import "context"

// Profile is a child profile as seen by a media provider.
type Profile struct {
	ID          string
	Name        string
	AvatarColor string
	IsChild     bool
}

// PolicyPayload is the provider-neutral policy subset adapters understand.
type PolicyPayload struct {
	DailyMinutes  int
	LocalOnly     bool
	HideShorts    bool
	HideLive      bool
	DownloadsOnly bool
	BedtimeStart  string
	BedtimeEnd    string
}

// ApplyResult is the outcome of pushing policy to a provider.
type ApplyResult struct {
	ProviderProfileID string
	Unsupported       []string
}

// ChildProfileProvider syncs child profiles and policies to an interchangeable media provider.
type ChildProfileProvider interface {
	ListChildProfiles(ctx context.Context) ([]Profile, error)
	CreateChildProfile(ctx context.Context, name, avatarColor string) (Profile, error)
	ApplyPolicy(ctx context.Context, providerProfileID string, policy PolicyPayload) (ApplyResult, error)
}
