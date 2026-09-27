package secret

import (
	"errors"
	"strings"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
	"github.com/zalando/go-keyring"
)

func platformAvailable() bool {
	// go-keyring reports unsupported backends via errors on Get; probe with a sentinel account.
	_, err := keyring.Get(Service, "__wonderfeed_probe__")
	if err == nil {
		return true
	}
	return !errors.Is(err, keyring.ErrUnsupportedPlatform)
}

func platformLoad(account string) (string, error) {
	account = strings.TrimSpace(account)
	if account == "" {
		return "", apperr.New(apperr.CodeInvalid, "secret.Load", "empty account")
	}
	value, err := keyring.Get(Service, account)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", apperr.New(apperr.CodeNotFound, "secret.Load", "secret not found in store").With("name", account)
		}
		if errors.Is(err, keyring.ErrUnsupportedPlatform) {
			return "", apperr.New(apperr.CodeFailed, "secret.Load", "secret store unavailable on this platform")
		}
		return "", apperr.Wrap(err, apperr.CodeUnavailable, "secret.Load", "load from store").With("name", account)
	}
	if strings.TrimSpace(value) == "" {
		return "", apperr.New(apperr.CodeNotFound, "secret.Load", "secret not found in store").With("name", account)
	}
	return value, nil
}

func platformStoreSecret(account, value string, force bool) error {
	account = strings.TrimSpace(account)
	if account == "" {
		return apperr.New(apperr.CodeInvalid, "secret.Store", "empty account")
	}
	if strings.TrimSpace(value) == "" {
		return apperr.New(apperr.CodeInvalid, "secret.Store", "empty value")
	}
	if !platformAvailable() {
		return apperr.New(apperr.CodeFailed, "secret.Store", "secret store unavailable on this platform")
	}
	if !force {
		_, loadErr := platformLoad(account)
		if loadErr == nil {
			return apperr.New(apperr.CodeConflict, "secret.Store", "secret exists (use force to replace)").With("name", account)
		}
		var ae *apperr.Error
		if errors.As(loadErr, &ae) && ae.Code != apperr.CodeNotFound {
			return loadErr
		}
	}
	if err := keyring.Set(Service, account, value); err != nil {
		return apperr.Wrap(err, apperr.CodeUnavailable, "secret.Store", "store secret").With("name", account)
	}
	return nil
}

func platformDelete(account string) error {
	account = strings.TrimSpace(account)
	if account == "" {
		return apperr.New(apperr.CodeInvalid, "secret.Delete", "empty account")
	}
	if !platformAvailable() {
		return apperr.New(apperr.CodeFailed, "secret.Delete", "secret store unavailable on this platform")
	}
	if err := keyring.Delete(Service, account); err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil
		}
		return apperr.Wrap(err, apperr.CodeUnavailable, "secret.Delete", "delete secret").With("name", account)
	}
	return nil
}
