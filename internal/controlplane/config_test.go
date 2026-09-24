package controlplane

import (
	"testing"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

func TestConfigValidateLoopbackWithoutAuth(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Bind:            "127.0.0.1:8080",
		DatabaseURL:     "postgres://ytzero:x@127.0.0.1:5432/ytzero?sslmode=disable",
		ProviderBaseURL: DefaultProviderBaseURL,
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("loopback without auth should be allowed: %v", err)
	}
}

func TestConfigValidateNonLoopbackRequiresAuth(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Bind:            "0.0.0.0:8080",
		DatabaseURL:     "postgres://ytzero:x@127.0.0.1:5432/ytzero?sslmode=disable",
		ProviderBaseURL: DefaultProviderBaseURL,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected fail-closed error for non-loopback without auth")
	}
	ae, ok := err.(*apperr.Error)
	if !ok || ae.Code != apperr.CodeInvalid {
		t.Fatalf("got %#v", err)
	}

	cfg.ParentAuthKey = "secret"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("non-loopback with auth should pass: %v", err)
	}
}

func TestConfigValidateRequiresDatabaseURL(t *testing.T) {
	t.Parallel()
	cfg := Config{
		Bind:            DefaultBind,
		ProviderBaseURL: DefaultProviderBaseURL,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected DATABASE_URL required")
	}
}

func TestIsLoopbackBind(t *testing.T) {
	t.Parallel()
	cases := []struct {
		addr string
		want bool
	}{
		{"127.0.0.1:8080", true},
		{"localhost:8080", true},
		{"[::1]:8080", true},
		{"0.0.0.0:8080", false},
		{"192.168.1.10:8080", false},
	}
	for _, tc := range cases {
		got, err := isLoopbackBind(tc.addr)
		if err != nil {
			t.Fatalf("%s: %v", tc.addr, err)
		}
		if got != tc.want {
			t.Fatalf("%s: got %v want %v", tc.addr, got, tc.want)
		}
	}
}
