package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/controlplane"
	"github.com/behaviorengineering/wonderfeed/internal/provider"
)

type okProvider struct{}

type noopAllowlistSync struct{}

func (noopAllowlistSync) AddMembership(ctx context.Context, providerProfileID string, channel provider.ScopedChannel) error {
	return nil
}
func (noopAllowlistSync) RemoveMembership(ctx context.Context, providerProfileID string, channel provider.ScopedChannel) error {
	return nil
}
func (noopAllowlistSync) ReconcileAll(ctx context.Context, providerProfileID string, channels []provider.ScopedChannel) error {
	return nil
}

func (okProvider) ListChildProfiles(ctx context.Context) ([]provider.Profile, error) {
	return nil, nil
}
func (okProvider) CreateChildProfile(ctx context.Context, name, avatarColor string) (provider.Profile, error) {
	return provider.Profile{ID: "1", Name: name, AvatarColor: avatarColor, IsChild: true}, nil
}
func (okProvider) ApplyPolicy(ctx context.Context, providerProfileID string, policy provider.PolicyPayload) (provider.ApplyResult, error) {
	return provider.ApplyResult{ProviderProfileID: providerProfileID}, nil
}

func testHandler(t *testing.T, authKey string) http.Handler {
	t.Helper()
	now := time.Date(2026, 9, 25, 3, 0, 0, 0, time.UTC)
	svc := controlplane.NewService(controlplane.ServiceConfig{
		Store:         controlplane.NewMemoryStore(func() time.Time { return now }),
		Provider:      okProvider{},
		AllowlistSync: &noopAllowlistSync{},
		Clock:         func() time.Time { return now },
		Logger:        slog.Default(),
	})
	h := NewHandler(HandlerConfig{Service: svc, ParentAuthKey: authKey})
	mux := http.NewServeMux()
	h.Mount(mux)
	return mux
}

func TestHealthUnauthenticated(t *testing.T) {
	t.Parallel()
	mux := testHandler(t, "secret")
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestParentRoutesRequireAuth(t *testing.T) {
	t.Parallel()
	mux := testHandler(t, "secret")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/parent/children", nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("code = %d body=%s", rr.Code, rr.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/parent/children", nil)
	req.Header.Set("Authorization", "Bearer secret")
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("code = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestCreateChildInvalidJSON(t *testing.T) {
	t.Parallel()
	mux := testHandler(t, "")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/parent/children", bytes.NewBufferString("{"))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("code = %d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["code"] != "invalid" {
		t.Fatalf("body = %#v", body)
	}
}

func TestCreateChildOK(t *testing.T) {
	t.Parallel()
	mux := testHandler(t, "")
	payload := `{"name":"Ada","avatar_color":"#123"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/parent/children", bytes.NewBufferString(payload))
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("code = %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAllowlistRoutes(t *testing.T) {
	t.Parallel()
	mux := testHandler(t, "")

	create := httptest.NewRequest(http.MethodPost, "/api/v1/parent/children", bytes.NewBufferString(`{"name":"Ada"}`))
	createRR := httptest.NewRecorder()
	mux.ServeHTTP(createRR, create)
	if createRR.Code != http.StatusCreated {
		t.Fatalf("create code = %d body=%s", createRR.Code, createRR.Body.String())
	}
	var child controlplane.ChildProfile
	if err := json.Unmarshal(createRR.Body.Bytes(), &child); err != nil {
		t.Fatal(err)
	}

	const channelID = "UCaaaaaaaaaaaaaaaaaaaaaa"
	add := httptest.NewRequest(http.MethodPost, "/api/v1/parent/children/"+child.ID+"/allowlist",
		bytes.NewBufferString(`{"expected_version":1,"provider":"youtube","external_id":"`+channelID+`","title":"Ada"}`))
	addRR := httptest.NewRecorder()
	mux.ServeHTTP(addRR, add)
	if addRR.Code != http.StatusCreated {
		t.Fatalf("add code = %d body=%s", addRR.Code, addRR.Body.String())
	}

	list := httptest.NewRequest(http.MethodGet, "/api/v1/parent/children/"+child.ID+"/allowlist", nil)
	listRR := httptest.NewRecorder()
	mux.ServeHTTP(listRR, list)
	if listRR.Code != http.StatusOK {
		t.Fatalf("list code = %d body=%s", listRR.Code, listRR.Body.String())
	}
	var body struct {
		Allowlist []controlplane.AllowlistChannel `json:"allowlist"`
		Version   int64                           `json:"version"`
	}
	if err := json.Unmarshal(listRR.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Version != 2 || len(body.Allowlist) != 1 || body.Allowlist[0].ExternalID != channelID {
		t.Fatalf("body = %+v", body)
	}
}
