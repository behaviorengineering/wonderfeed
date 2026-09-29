package provider

import "context"

// ProviderYouTube is the allowlist provider key for YouTube channel ids (UC…).
const ProviderYouTube = "youtube"

// YouTubeFollowOwnership locks a YT Zero child profile so the child cannot widen
// YouTube channel membership from the provider UI. Other providers are out of scope.
type YouTubeFollowOwnership interface {
	ApplyYouTubeFollowOwnership(ctx context.Context, providerProfileID string) error
}
