package backup

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/behaviorengineering/wonderfeed/internal/apperr"
)

// ObjectStore persists and lists encrypted backup blobs.
type ObjectStore interface {
	Put(ctx context.Context, id string, ciphertext []byte) (location string, err error)
	Get(ctx context.Context, id string) (ciphertext []byte, err error)
	List(ctx context.Context) ([]ObjectInfo, error)
}

// ObjectInfo is a listed backup object.
type ObjectInfo struct {
	ID        string    `json:"id"`
	Location  string    `json:"location"`
	SizeBytes int64     `json:"sizeBytes"`
	ModTime   time.Time `json:"modTime"`
	Backend   string    `json:"backend"`
}

// LocalStore keeps objects under Paths.ObjectsDir as <id>.age.
type LocalStore struct {
	Dir string
}

// Put writes ciphertext to Dir/<id>.age.
func (s LocalStore) Put(_ context.Context, id string, ciphertext []byte) (string, error) {
	const op = "backup.LocalStore.Put"
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return "", wrap(err, op, "mkdir objects")
	}
	path := filepath.Join(s.Dir, id+".age")
	if err := os.WriteFile(path, ciphertext, 0o600); err != nil {
		return "", wrap(err, op, "write object").With("path", path)
	}
	return path, nil
}

// Get reads Dir/<id>.age.
func (s LocalStore) Get(_ context.Context, id string) ([]byte, error) {
	const op = "backup.LocalStore.Get"
	path := filepath.Join(s.Dir, id+".age")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, apperr.Wrap(err, apperr.CodeNotFound, op, "local backup not found").With("id", id)
		}
		return nil, wrap(err, op, "read object").With("path", path)
	}
	return data, nil
}

// List returns local .age objects newest first.
func (s LocalStore) List(_ context.Context) ([]ObjectInfo, error) {
	const op = "backup.LocalStore.List"
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, wrap(err, op, "read objects dir")
	}
	var out []ObjectInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".age") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".age")
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, ObjectInfo{
			ID:        id,
			Location:  filepath.Join(s.Dir, e.Name()),
			SizeBytes: info.Size(),
			ModTime:   info.ModTime().UTC(),
			Backend:   "local",
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ModTime.After(out[j].ModTime) })
	return out, nil
}

// S3Store is an optional S3-compatible backend.
type S3Store struct {
	Client *s3.Client
	Bucket string
	Prefix string
}

// NewS3Store builds a client when S3 config is complete.
func NewS3Store(cfg S3Config) (*S3Store, error) {
	const op = "backup.NewS3Store"
	if cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, apperr.New(apperr.CodeInvalid, op, "incomplete S3 config (need bucket, access key, secret key)")
	}
	region := cfg.Region
	if region == "" {
		region = "auto"
	}
	opts := s3.Options{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
	}
	if cfg.Endpoint != "" {
		endpoint := cfg.Endpoint
		if _, err := url.Parse(endpoint); err != nil {
			return nil, wrap(err, op, "parse S3 endpoint")
		}
		opts.BaseEndpoint = aws.String(endpoint)
		opts.UsePathStyle = true
	}
	return &S3Store{
		Client: s3.New(opts),
		Bucket: cfg.Bucket,
		Prefix: cfg.Prefix,
	}, nil
}

func (s *S3Store) objectKey(id string) string {
	if s.Prefix == "" {
		return id + ".age"
	}
	return s.Prefix + "/" + id + ".age"
}

// Put uploads ciphertext.
func (s *S3Store) Put(ctx context.Context, id string, ciphertext []byte) (string, error) {
	const op = "backup.S3Store.Put"
	key := s.objectKey(id)
	_, err := s.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(ciphertext),
	})
	if err != nil {
		return "", wrap(err, op, "put s3 object").With("key", key)
	}
	return fmt.Sprintf("s3://%s/%s", s.Bucket, key), nil
}

// Get downloads ciphertext.
func (s *S3Store) Get(ctx context.Context, id string) ([]byte, error) {
	const op = "backup.S3Store.Get"
	key := s.objectKey(id)
	out, err := s.Client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, wrap(err, op, "get s3 object").With("key", key)
	}
	defer out.Body.Close()
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(out.Body); err != nil {
		return nil, wrap(err, op, "read s3 body").With("key", key)
	}
	return buf.Bytes(), nil
}

// List returns remote objects newest first.
func (s *S3Store) List(ctx context.Context) ([]ObjectInfo, error) {
	const op = "backup.S3Store.List"
	prefix := s.Prefix
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	out, err := s.Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.Bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, wrap(err, op, "list s3 objects")
	}
	var items []ObjectInfo
	for _, obj := range out.Contents {
		key := aws.ToString(obj.Key)
		if !strings.HasSuffix(key, ".age") {
			continue
		}
		base := filepath.Base(key)
		id := strings.TrimSuffix(base, ".age")
		mod := time.Time{}
		if obj.LastModified != nil {
			mod = obj.LastModified.UTC()
		}
		items = append(items, ObjectInfo{
			ID:        id,
			Location:  fmt.Sprintf("s3://%s/%s", s.Bucket, key),
			SizeBytes: aws.ToInt64(obj.Size),
			ModTime:   mod,
			Backend:   "s3",
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ModTime.After(items[j].ModTime) })
	return items, nil
}

// S3Configured reports whether remote upload can run.
func S3Configured(cfg S3Config) bool {
	return cfg.Bucket != "" && cfg.AccessKey != "" && cfg.SecretKey != ""
}
