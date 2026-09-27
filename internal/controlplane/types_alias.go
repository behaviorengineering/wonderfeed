package controlplane

import cp "github.com/behaviorengineering/wonderfeed/pkg/controlplane"

// Domain types re-exported from the public package for host implementation code.
type (
	ChildProfile     = cp.ChildProfile
	ChildPolicy      = cp.ChildPolicy
	AllowlistChannel = cp.AllowlistChannel
	SyncStatus       = cp.SyncStatus
)

const (
	SyncPending     = cp.SyncPending
	SyncSynced      = cp.SyncSynced
	SyncFailed      = cp.SyncFailed
	MaxDailyMinutes = cp.MaxDailyMinutes
)

// DefaultChildPolicy returns fail-closed defaults for a new child profile.
func DefaultChildPolicy() ChildPolicy {
	return cp.DefaultChildPolicy()
}
