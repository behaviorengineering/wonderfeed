package ytzero

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/controlplane"
	"github.com/behaviorengineering/wonderfeed/internal/provider"
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

func TestResilientClientRequiresDeadline(t *testing.T) {
	t.Parallel()
	c := NewResilientClient("http://example.invalid", http.DefaultClient, "")
	_, _, err := c.DoJSON(context.Background(), http.MethodGet, "/api/profiles", nil)
	if err == nil {
		t.Fatal("expected missing deadline error")
	}
}

func TestAdapterApplyAllowlistUsesAdminReconcile(t *testing.T) {
	t.Parallel()
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/channels/reconcile" {
			http.NotFound(w, r)
			return
		}
		if strings.Contains(r.Header.Get("Cookie"), "ytzero_profile=") {
			t.Errorf("unexpected child profile cookie: %q", r.Header.Get("Cookie"))
		}
		if !strings.Contains(r.Header.Get("Cookie"), "ytzero_session=secret") {
			t.Errorf("cookie = %q", r.Header.Get("Cookie"))
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("decode body: %v", err)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "followed": 1, "unfollowed": 0})
	}))
	defer srv.Close()

	a := NewAdapter(AdapterConfig{
		BaseURL:       srv.URL,
		SessionCookie: "ytzero_session=secret",
		HTTP:          srv.Client(),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := a.ApplyAllowlist(ctx, "9", []provider.Channel{{ID: "UC-new", Title: "New"}}); err != nil {
		t.Fatalf("apply allowlist: %v", err)
	}
	if int(gotBody["user_id"].(float64)) != 9 {
		t.Fatalf("user_id = %#v", gotBody["user_id"])
	}
	ids, ok := gotBody["channel_ids"].([]any)
	if !ok || len(ids) != 1 || ids[0] != "UC-new" {
		t.Fatalf("channel_ids = %#v", gotBody["channel_ids"])
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
