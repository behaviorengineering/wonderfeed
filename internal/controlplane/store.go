package controlplane

import "context"

// Store persists child profiles and desired policies.
type Store interface {
	Create(ctx context.Context, name, avatarColor string, policy ChildPolicy) (ChildProfile, error)
	List(ctx context.Context) ([]ChildProfile, error)
	Get(ctx context.Context, id string) (ChildProfile, error)
	ListAllowlist(ctx context.Context, id string) ([]AllowlistChannel, error)
	ReplaceAllowlist(ctx context.Context, id string, expectedVersion int64, channels []AllowlistChannel) (ChildProfile, error)
	UpdatePolicy(ctx context.Context, id string, expectedVersion int64, policy ChildPolicy) (ChildProfile, error)
	SetProviderProfileID(ctx context.Context, id, providerProfileID string) (ChildProfile, error)
	RecordSyncStatus(ctx context.Context, id string, status SyncStatus, syncErr string) (ChildProfile, error)
}
