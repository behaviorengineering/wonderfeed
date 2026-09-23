package backup_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/backup"
)

func TestArchiveRoundTripExcludesMedia(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	mustWrite(t, filepath.Join(dir, "database-state.json"), `{"version":1}`)
	mustWrite(t, filepath.Join(dir, "avatars", "a.png"), "avatar")
	mustWrite(t, filepath.Join(dir, "downloads", "big.mp4"), "VIDEO")
	mustWrite(t, filepath.Join(dir, "imgcache", "x.jpg"), "IMG")
	mustWrite(t, filepath.Join(dir, "logs", "ytzero.log"), "LOG")
	mustWrite(t, filepath.Join(dir, "download-cookies", "c.txt"), "SECRET")

	manifest := backup.Manifest{
		Version:   backup.ArchiveVersion,
		ID:        "test1",
		CreatedAt: time.Date(2026, 9, 23, 1, 0, 0, 0, time.UTC),
		Excludes:  append([]string{}, backup.DefaultExcludes...),
	}
	manifest.Postgres.Format = "plain"
	manifest.Postgres.Database = "ytzero"
	manifest.Postgres.User = "ytzero"

	dump := []byte("SELECT 1;\n")
	raw, includes, err := backup.BuildArchiveTar(manifest, dump, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(includes) < 2 {
		t.Fatalf("includes = %#v", includes)
	}
	got, gotDump, state, err := backup.ReadArchiveTar(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "test1" {
		t.Fatalf("id = %q", got.ID)
	}
	if string(gotDump) != string(dump) {
		t.Fatalf("dump mismatch")
	}
	if _, ok := state["state/database-state.json"]; !ok {
		t.Fatal("expected database-state.json")
	}
	if _, ok := state["state/avatars/a.png"]; !ok {
		t.Fatal("expected avatars")
	}
	for _, forbidden := range []string{
		"state/downloads/big.mp4",
		"state/imgcache/x.jpg",
		"state/logs/ytzero.log",
		"state/download-cookies/c.txt",
	} {
		if _, ok := state[forbidden]; ok {
			t.Fatalf("excluded path present: %s", forbidden)
		}
	}
}

func TestAgeEncryptDecrypt(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	identity := filepath.Join(dir, "age-identity.txt")
	if _, _, err := backup.EnsureAgeIdentity(identity); err != nil {
		t.Fatal(err)
	}
	plain := []byte("hello wonderfeed")
	cipher, err := backup.Encrypt(identity, plain)
	if err != nil {
		t.Fatal(err)
	}
	if string(cipher) == string(plain) {
		t.Fatal("ciphertext should differ from plaintext")
	}
	out, err := backup.Decrypt(identity, cipher)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != string(plain) {
		t.Fatalf("got %q", out)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}
