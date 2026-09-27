package secret

import (
	"errors"
	"os"
	"strings"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

// IsNotFound reports whether err means the secret is unset in env and store.
func IsNotFound(err error) bool {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae.Code == apperr.CodeNotFound
	}
	return false
}

// Resolve returns a secret by env var name: process env first, then the platform store.
func Resolve(name string) (string, error) {
	return ResolveWith(DefaultStore(), name)
}

// ResolveWith is Resolve with an injectable store (tests and alternate backends).
func ResolveWith(store Store, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", apperr.New(apperr.CodeInvalid, "secret.Resolve", "empty name")
	}
	if v := strings.TrimSpace(os.Getenv(name)); v != "" {
		return v, nil
	}
	if store != nil && store.Available() {
		v, err := store.Load(name)
		if err == nil && strings.TrimSpace(v) != "" {
			return v, nil
		}
		if err != nil && !IsNotFound(err) {
			var ae *apperr.Error
			if errors.As(err, &ae) && ae.Code == apperr.CodeFailed {
				// Store unavailable: treat as not found so env-only hosts still work.
			} else if err != nil {
				return "", apperr.Wrap(err, apperr.CodeUnavailable, "secret.Resolve", "load from store").With("name", name)
			}
		}
	}
	return "", apperr.New(apperr.CodeNotFound, "secret.Resolve", "secret not set in environment or store").With("name", name)
}

// Set stores a secret in the default platform store.
func Set(name, value string, force bool) error {
	return SetWith(DefaultStore(), name, value, force)
}

// SetWith stores a secret using the given store.
func SetWith(store Store, name, value string, force bool) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return apperr.New(apperr.CodeInvalid, "secret.Set", "empty name")
	}
	if strings.TrimSpace(value) == "" {
		return apperr.New(apperr.CodeInvalid, "secret.Set", "empty value")
	}
	if store == nil || !store.Available() {
		return apperr.New(apperr.CodeFailed, "secret.Set", "secret store unavailable on this platform")
	}
	return store.Store(name, value, force)
}

// Delete removes a secret from the default platform store.
func Delete(name string) error {
	return DeleteWith(DefaultStore(), name)
}

// DeleteWith removes a secret using the given store.
func DeleteWith(store Store, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return apperr.New(apperr.CodeInvalid, "secret.Delete", "empty name")
	}
	if store == nil || !store.Available() {
		return apperr.New(apperr.CodeFailed, "secret.Delete", "secret store unavailable on this platform")
	}
	return store.Delete(name)
}
