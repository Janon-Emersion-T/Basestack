// Package storage supplies private bucket storage with replaceable providers.
package storage

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/Janon-Emersion-T/Basestack/internal/runtime/privatefs"
)

var ErrInvalid = errors.New("invalid bucket or object identifier")
var ErrUnavailable = errors.New("storage unavailable")
var ErrNotFound = errors.New("storage object not found")
var ErrTooLarge = errors.New("storage object exceeds configured limit")
var bucketName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,49}$`)
var objectID = regexp.MustCompile(`^[0-9a-f]{64}$`)

type Metadata struct {
	ID         string    `json:"id"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modifiedAt"`
}
type Store interface {
	CreateBucket(context.Context, string) error
	Buckets(context.Context) ([]string, error)
	List(context.Context, string) ([]Metadata, error)
	Put(context.Context, string, io.Reader) (Metadata, error)
	Open(context.Context, string, string) (io.ReadCloser, Metadata, error)
	Delete(context.Context, string, string) error
}
type Local struct {
	root    string
	maximum int64
}

func NewLocal(project string, maximum int64) (*Local, error) {
	if maximum < 1 || maximum > 1<<30 {
		return nil, ErrInvalid
	}
	root, err := privatefs.Directory(project, true, ".basestack", "storage")
	if err != nil {
		return nil, ErrUnavailable
	}
	return &Local{root, maximum}, nil
}
func (s *Local) bucket(ctx context.Context, bucket string, create bool) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}
	if !bucketName.MatchString(bucket) {
		return "", ErrInvalid
	}
	path, err := privatefs.Directory(s.root, create, bucket)
	if os.IsNotExist(err) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", ErrUnavailable
	}
	return path, nil
}
func (s *Local) CreateBucket(ctx context.Context, bucket string) error {
	_, err := s.bucket(ctx, bucket, true)
	return err
}
func (s *Local) Buckets(ctx context.Context) ([]string, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return nil, ErrUnavailable
	}
	result := []string{}
	for _, e := range entries {
		if _, err := s.bucket(ctx, e.Name(), false); err != nil {
			return nil, err
		}
		result = append(result, e.Name())
	}
	return result, nil
}
func (s *Local) List(ctx context.Context, bucket string) ([]Metadata, error) {
	dir, err := s.bucket(ctx, bucket, false)
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, ErrUnavailable
	}
	result := []Metadata{}
	for _, e := range entries {
		if !objectID.MatchString(e.Name()) {
			continue
		} // In-progress temporary uploads are private.
		f, m, err := s.Open(ctx, bucket, e.Name())
		if err != nil {
			return nil, err
		}
		f.Close()
		result = append(result, m)
	}
	return result, nil
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r contextReader) Read(b []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(b)
}
func (s *Local) Put(ctx context.Context, bucket string, reader io.Reader) (Metadata, error) {
	dir, err := s.bucket(ctx, bucket, false)
	if err != nil {
		return Metadata{}, err
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Metadata{}, ErrUnavailable
	}
	id := hex.EncodeToString(raw)
	f, err := os.CreateTemp(dir, ".upload-*")
	if err != nil {
		return Metadata{}, ErrUnavailable
	}
	defer os.Remove(f.Name())
	defer f.Close()
	n, err := io.Copy(f, io.LimitReader(contextReader{ctx, reader}, s.maximum+1))
	if err != nil {
		return Metadata{}, ErrUnavailable
	}
	if n > s.maximum {
		return Metadata{}, ErrTooLarge
	}
	if ctx.Err() != nil {
		return Metadata{}, ctx.Err()
	}
	if f.Sync() != nil {
		return Metadata{}, ErrUnavailable
	}
	info, err := f.Stat()
	if err != nil {
		return Metadata{}, ErrUnavailable
	}
	if f.Close() != nil {
		return Metadata{}, ErrUnavailable
	}
	// Link publishes complete bytes exclusively, so even a random collision cannot overwrite.
	if os.Link(f.Name(), filepath.Join(dir, id)) != nil {
		return Metadata{}, ErrUnavailable
	}
	return Metadata{id, n, info.ModTime().UTC()}, nil
}
func (s *Local) Open(ctx context.Context, bucket, id string) (io.ReadCloser, Metadata, error) {
	dir, err := s.bucket(ctx, bucket, false)
	if err != nil {
		return nil, Metadata{}, err
	}
	if !objectID.MatchString(id) {
		return nil, Metadata{}, ErrInvalid
	}
	path := filepath.Join(dir, id)
	if err := privatefs.Regular(path); os.IsNotExist(err) {
		return nil, Metadata{}, ErrNotFound
	} else if err != nil {
		return nil, Metadata{}, ErrUnavailable
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, Metadata{}, ErrUnavailable
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, Metadata{}, ErrUnavailable
	}
	return f, Metadata{id, info.Size(), info.ModTime().UTC()}, nil
}
func (s *Local) Delete(ctx context.Context, bucket, id string) error {
	f, _, err := s.Open(ctx, bucket, id)
	if err != nil {
		return err
	}
	f.Close()
	if os.Remove(filepath.Join(s.root, bucket, id)) != nil {
		return ErrUnavailable
	}
	return nil
}
