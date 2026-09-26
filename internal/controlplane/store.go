package controlplane

import "context"

// Store persists child profiles and desired policies.
type Store interface {
	Create(ctx context.Context, name, avatarColor string, policy ChildPolicy) (ChildProfile, error)
	List(ctx context.Context) ([]ChildProfile, error)
	Get(ctx context.Context, id string) (ChildProfile, error)
	ListAllowlist(ctx context.Context, id string) ([]AllowlistChannel, error)
	GetAllowlistEntry(ctx context.Context, id, providerKey, externalID string) (AllowlistChannel, error)
	AddAllowlistEntry(ctx context.Context, id string, expectedVersion int64, entry AllowlistChannel) (ChildProfile, error)
	UpdateAllowlistEntry(ctx context.Context, id, providerKey, externalID string, title, channelURL string) (AllowlistChannel, error)
	DeleteAllowlistEntry(ctx context.Context, id string, expectedVersion int64, providerKey, externalID string) (ChildProfile, error)
	UpdatePolicy(ctx context.Context, id string, expectedVersion int64, policy ChildPolicy) (ChildProfile, error)
	SetProviderProfileID(ctx context.Context, id, providerProfileID string) (ChildProfile, error)
	RecordSyncStatus(ctx context.Context, id string, status SyncStatus, syncErr string) (ChildProfile, error)
}
