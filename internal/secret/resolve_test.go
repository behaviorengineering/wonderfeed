package secret

import (
	"os"
	"testing"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

type mapStore map[string]string

func (m mapStore) Available() bool { return true }

func (m mapStore) Load(account string) (string, error) {
	if v, ok := m[account]; ok && v != "" {
		return v, nil
	}
	return "", apperr.New(apperr.CodeNotFound, "mapStore.Load", "missing")
}

func (m mapStore) Store(account, value string, force bool) error { m[account] = value; return nil }

func (m mapStore) Delete(account string) error { delete(m, account); return nil }

func TestResolvePrefersEnvironment(t *testing.T) {
	const key = "WONDERFEED_TEST_SECRET_RESOLVE"
	t.Setenv(key, "from-env")
	store := mapStore{key: "from-store"}
	got, err := ResolveWith(store, key)
	if err != nil {
		t.Fatalf("ResolveWith: %v", err)
	}
	if got != "from-env" {
		t.Fatalf("got %q want from-env", got)
	}
}

func TestResolveUsesStoreWhenEnvEmpty(t *testing.T) {
	const key = "WONDERFEED_TEST_SECRET_STORE_ONLY"
	os.Unsetenv(key)
	store := mapStore{key: "from-store"}
	got, err := ResolveWith(store, key)
	if err != nil {
		t.Fatalf("ResolveWith: %v", err)
	}
	if got != "from-store" {
		t.Fatalf("got %q want from-store", got)
	}
}

func TestResolveFailClosedWhenMissing(t *testing.T) {
	const key = "WONDERFEED_TEST_SECRET_MISSING"
	os.Unsetenv(key)
	_, err := ResolveWith(mapStore{}, key)
	if err == nil || !IsNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}
