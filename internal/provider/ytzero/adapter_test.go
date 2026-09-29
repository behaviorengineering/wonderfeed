package ytzero

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/controlplane"
	"github.com/behaviorengineering/wonderfeed/pkg/provider"
)

func TestAdapterCreateAndApplyPolicy(t *testing.T) {
	t.Parallel()
	var gotPatch map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/profiles":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"profile": map[string]any{
					"id":           9,
					"name":         "Ada",
					"avatar_color": "#112233",
					"is_child":     true,
				},
			})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/profiles/9":
			if err := json.NewDecoder(r.Body).Decode(&gotPatch); err != nil {
				t.Errorf("decode patch: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"profile": map[string]any{"id": 9, "name": "Ada", "is_child": true},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/profiles":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"profiles": []map[string]any{
					{"id": 1, "name": "Parent", "is_child": false},
					{"id": 9, "name": "Ada", "avatar_color": "#112233", "is_child": true},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	a := NewAdapter(AdapterConfig{BaseURL: srv.URL, HTTP: srv.Client()})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	created, err := a.CreateChildProfile(ctx, "Ada", "#112233")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID != "9" {
		t.Fatalf("id = %q", created.ID)
	}

	policy := controlplane.DefaultChildPolicy()
	policy.DailyMinutes = 30
	policy.BedtimeStart = "21:00"
	policy.BedtimeEnd = "07:00"
	result, err := a.ApplyPolicy(ctx, created.ID, provider.PolicyPayload{
		DailyMinutes:  policy.DailyMinutes,
		LocalOnly:     policy.LocalOnly,
		HideShorts:    policy.HideShorts,
		HideLive:      policy.HideLive,
		DownloadsOnly: policy.DownloadsOnly,
		BedtimeStart:  policy.BedtimeStart,
		BedtimeEnd:    policy.BedtimeEnd,
	})
	if err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(result.Unsupported) != 1 || result.Unsupported[0] != "bedtime" {
		t.Fatalf("unsupported = %#v", result.Unsupported)
	}
	cc, ok := gotPatch["child_config"].(map[string]any)
	if !ok {
		t.Fatalf("patch body = %#v", gotPatch)
	}
	if int(cc["limit_minutes"].(float64)) != 30 {
		t.Fatalf("limit_minutes = %#v", cc["limit_minutes"])
	}

	listed, err := a.ListChildProfiles(ctx)
	if err != nil || len(listed) != 1 || listed[0].ID != "9" {
		t.Fatalf("list: %v %#v", err, listed)
	}
}

func TestAdapterApplyYouTubeFollowOwnership(t *testing.T) {
	t.Parallel()
	var gotPut map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/access-control":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"default_group_id": 1,
				"groups": []map[string]any{
					{"id": 2, "name": "Restricted"},
				},
				"profiles": []map[string]any{
					{"id": 9, "access": map[string]any{"group_id": 1}},
				},
			})
		case r.Method == http.MethodPut && r.URL.Path == "/api/access-control/profiles/9":
			if err := json.NewDecoder(r.Body).Decode(&gotPut); err != nil {
				t.Errorf("decode put: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{
				"access": map[string]any{"group_id": 2},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	a := NewAdapter(AdapterConfig{BaseURL: srv.URL, HTTP: srv.Client(), SessionCookie: "ytzero_session=test"})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.ApplyYouTubeFollowOwnership(ctx, "9"); err != nil {
		t.Fatalf("apply YouTube follow ownership: %v", err)
	}
	if int(gotPut["group_id"].(float64)) != 2 {
		t.Fatalf("group_id = %#v", gotPut["group_id"])
	}
	overrides, ok := gotPut["overrides"].(map[string]any)
	if !ok || overrides["channels"] != "deny" || overrides["imports"] != "deny" {
		t.Fatalf("overrides = %#v", gotPut["overrides"])
	}
}

func TestAdapterApplyYouTubeFollowOwnershipRequiresSessionCookie(t *testing.T) {
	t.Parallel()
	a := NewAdapter(AdapterConfig{BaseURL: "http://example.invalid", HTTP: http.DefaultClient})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := a.ApplyYouTubeFollowOwnership(ctx, "9"); err == nil {
		t.Fatal("expected missing session cookie error")
	}
}

func TestResilientClientRequiresDeadline(t *testing.T) {
	t.Parallel()
	c := NewResilientClient("http://example.invalid", http.DefaultClient, "")
	_, _, err := c.DoJSON(context.Background(), http.MethodGet, "/api/profiles", nil)
	if err == nil {
		t.Fatal("expected missing deadline error")
	}
}

func TestResilientClientTransientRetry(t *testing.T) {
	t.Parallel()
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("busy"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"profiles":[]}`))
	}))
	defer srv.Close()

	c := NewResilientClient(srv.URL, srv.Client(), "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	raw, _, err := c.DoJSON(ctx, http.MethodGet, "/api/profiles", nil)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	if attempts < 2 {
		t.Fatalf("attempts = %d", attempts)
	}
	if string(raw) == "" {
		t.Fatal("empty body")
	}
}
