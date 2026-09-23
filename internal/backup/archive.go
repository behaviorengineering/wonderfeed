package backup

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

// Manifest describes one backup archive.
type Manifest struct {
	Version   int       `json:"version"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Hostname  string    `json:"hostname,omitempty"`
	Postgres  struct {
		Format   string `json:"format"`
		Database string `json:"database"`
		User     string `json:"user"`
	} `json:"postgres"`
	Includes []string `json:"includes"`
	Excludes []string `json:"excludes"`
}

// DefaultExcludes are top-level data/ytzero entries skipped in state packaging.
var DefaultExcludes = []string{
	"downloads",
	"imgcache",
	"logs",
	"download-cookies",
	"bin",
	"db",
	"restore-sessions",
	"sqlite-pre-postgres-backup",
}

// NewBackupID returns a UTC timestamp id suitable for object names.
func NewBackupID(now time.Time) string {
	return now.UTC().Format("20060102T150405Z")
}

// BuildArchiveTar builds a plaintext tar with manifest, dump, and allowed state files.
// It returns the tar bytes and the final include list written into the manifest.
func BuildArchiveTar(manifest Manifest, dump []byte, providerData string) ([]byte, []string, error) {
	const op = "backup.BuildArchiveTar"
	exclude := excludeSet(manifest.Excludes)
	state, order, err := collectStateFiles(providerData, exclude)
	if err != nil {
		return nil, nil, err
	}

	includes := []string{ManifestName, DumpRelPath}
	includes = append(includes, order...)
	manifest.Includes = includes

	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, nil, wrap(err, op, "encode manifest")
	}

	var out bytes.Buffer
	tw := tar.NewWriter(&out)
	if err := writeTarFile(tw, ManifestName, manifestJSON); err != nil {
		return nil, nil, err
	}
	if err := writeTarFile(tw, DumpRelPath, dump); err != nil {
		return nil, nil, err
	}
	for _, name := range order {
		if err := writeTarFile(tw, name, state[name]); err != nil {
			return nil, nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, nil, wrap(err, op, "close tar")
	}
	return out.Bytes(), includes, nil
}

// ReadArchiveTar parses a plaintext tar archive into manifest, dump, and state files.
func ReadArchiveTar(raw []byte) (Manifest, []byte, map[string][]byte, error) {
	const op = "backup.ReadArchiveTar"
	tr := tar.NewReader(bytes.NewReader(raw))
	var manifest Manifest
	var dump []byte
	state := map[string][]byte{}
	sawManifest := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return Manifest{}, nil, nil, wrap(err, op, "read tar entry")
		}
		if hdr.Typeflag != tar.TypeReg && hdr.Typeflag != tar.TypeRegA {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return Manifest{}, nil, nil, wrap(err, op, "read tar body").With("name", hdr.Name)
		}
		switch {
		case hdr.Name == ManifestName:
			if err := json.Unmarshal(data, &manifest); err != nil {
				return Manifest{}, nil, nil, wrap(err, op, "parse manifest")
			}
			sawManifest = true
		case hdr.Name == DumpRelPath:
			dump = data
		case strings.HasPrefix(hdr.Name, "state/"):
			state[hdr.Name] = data
		}
	}
	if !sawManifest || manifest.Version == 0 {
		return Manifest{}, nil, nil, apperr.New(apperr.CodeInvalid, op, "manifest missing or invalid")
	}
	if dump == nil {
		return Manifest{}, nil, nil, apperr.New(apperr.CodeInvalid, op, "postgres dump missing from archive")
	}
	return manifest, dump, state, nil
}

func collectStateFiles(providerData string, exclude map[string]bool) (map[string][]byte, []string, error) {
	const op = "backup.collectStateFiles"
	state := map[string][]byte{}
	var order []string
	if providerData == "" {
		return state, order, nil
	}
	err := filepath.WalkDir(providerData, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(providerData, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		top := strings.SplitN(rel, "/", 2)[0]
		if exclude[top] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		base := filepath.Base(path)
		if strings.HasSuffix(base, ".age") || base == "age-identity.txt" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		name := "state/" + rel
		state[name] = data
		order = append(order, name)
		return nil
	})
	if err != nil {
		return nil, nil, wrap(err, op, "walk provider data")
	}
	return state, order, nil
}

func writeTarFile(tw *tar.Writer, name string, data []byte) error {
	const op = "backup.writeTarFile"
	hdr := &tar.Header{
		Name:    name,
		Mode:    0o600,
		Size:    int64(len(data)),
		ModTime: time.Now().UTC(),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		return wrap(err, op, "write tar header").With("name", name)
	}
	if _, err := tw.Write(data); err != nil {
		return wrap(err, op, "write tar body").With("name", name)
	}
	return nil
}

func excludeSet(names []string) map[string]bool {
	out := map[string]bool{}
	for _, n := range names {
		out[n] = true
	}
	return out
}
