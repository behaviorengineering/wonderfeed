package backup

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

// EnsureAgeIdentity creates an age identity file when missing and returns its recipient.
func EnsureAgeIdentity(path string) (recipient string, created bool, err error) {
	const op = "backup.EnsureAgeIdentity"
	if path == "" {
		return "", false, apperr.New(apperr.CodeInvalid, op, "age identity path is empty")
	}
	if _, err := os.Stat(path); err == nil {
		rec, err := recipientFromIdentityFile(path)
		return rec, false, err
	} else if !os.IsNotExist(err) {
		return "", false, wrap(err, op, "stat age identity").With("path", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", false, wrap(err, op, "create age identity directory").With("path", path)
	}
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return "", false, wrap(err, op, "generate age identity")
	}
	body := identity.String() + "\n# recipient: " + identity.Recipient().String() + "\n"
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		return "", false, wrap(err, op, "write age identity").With("path", path)
	}
	return identity.Recipient().String(), true, nil
}

func recipientFromIdentityFile(path string) (string, error) {
	const op = "backup.recipientFromIdentityFile"
	identities, err := parseAgeIdentities(path)
	if err != nil {
		return "", err
	}
	if len(identities) == 0 {
		return "", apperr.New(apperr.CodeInvalid, op, "age identity file has no identities").With("path", path)
	}
	if x, ok := identities[0].(*age.X25519Identity); ok {
		return x.Recipient().String(), nil
	}
	return "", apperr.New(apperr.CodeInvalid, op, "unsupported age identity type").With("path", path)
}

func parseAgeIdentities(path string) ([]age.Identity, error) {
	const op = "backup.parseAgeIdentities"
	f, err := os.Open(path)
	if err != nil {
		return nil, apperr.Wrap(err, apperr.CodeNotFound, op, "open age identity").With("path", path)
	}
	defer f.Close()
	identities, err := age.ParseIdentities(f)
	if err != nil {
		return nil, wrap(err, op, "parse age identity").With("path", path)
	}
	return identities, nil
}

// Encrypt encrypts plaintext to the recipient derived from the identity file.
func Encrypt(identityPath string, plaintext []byte) ([]byte, error) {
	const op = "backup.Encrypt"
	recipient, err := recipientFromIdentityFile(identityPath)
	if err != nil {
		return nil, err
	}
	r, err := age.ParseX25519Recipient(recipient)
	if err != nil {
		return nil, wrap(err, op, "parse age recipient")
	}
	var buf bytes.Buffer
	w, err := age.Encrypt(&buf, r)
	if err != nil {
		return nil, wrap(err, op, "start age encryption")
	}
	if _, err := w.Write(plaintext); err != nil {
		return nil, wrap(err, op, "write age ciphertext")
	}
	if err := w.Close(); err != nil {
		return nil, wrap(err, op, "close age encryption")
	}
	return buf.Bytes(), nil
}

// Decrypt decrypts ciphertext with the identity file.
func Decrypt(identityPath string, ciphertext []byte) ([]byte, error) {
	const op = "backup.Decrypt"
	identities, err := parseAgeIdentities(identityPath)
	if err != nil {
		return nil, err
	}
	r, err := age.Decrypt(bytes.NewReader(ciphertext), identities...)
	if err != nil {
		return nil, wrap(err, op, "decrypt age ciphertext")
	}
	plain, err := io.ReadAll(r)
	if err != nil {
		return nil, wrap(err, op, "read decrypted payload")
	}
	return plain, nil
}

func identityLooksPresent(path string) bool {
	st, err := os.Stat(path)
	return err == nil && st.Size() > 0 && !strings.HasSuffix(path, "/")
}
