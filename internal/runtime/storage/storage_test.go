package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestLocalLifecycleLimitsAndPaths(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	s, err := NewLocal(dir, 4)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.CreateBucket(ctx, "files"); err != nil {
		t.Fatal(err)
	}
	m, err := s.Put(ctx, "files", strings.NewReader("data"))
	if err != nil || m.Size != 4 {
		t.Fatal("upload", err)
	}
	f, got, err := s.Open(ctx, "files", m.ID)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(f)
	f.Close()
	if string(data) != "data" || got.ID != m.ID {
		t.Fatal("wrong object")
	}
	if _, err := s.Put(ctx, "files", strings.NewReader("large")); err != ErrTooLarge {
		t.Fatal("limit ignored")
	}
	list, err := s.List(ctx, "files")
	if err != nil || len(list) != 1 {
		t.Fatal("incomplete upload published")
	}
	for _, bad := range []string{"../escape", "/absolute", "a/b", "a\\b", "..", "", "%2e%2e"} {
		if s.CreateBucket(ctx, bad) == nil {
			t.Fatal("unsafe bucket accepted")
		}
		if _, _, err := s.Open(ctx, "files", bad); err == nil {
			t.Fatal("unsafe key accepted")
		}
	}
	outside := filepath.Join(t.TempDir(), "private")
	os.WriteFile(outside, []byte("secret"), 0600)
	linkID := strings.Repeat("a", 64)
	os.Symlink(outside, filepath.Join(s.root, "files", linkID))
	if _, _, err := s.Open(ctx, "files", linkID); err == nil {
		t.Fatal("symlink read")
	}
	if s.Delete(ctx, "files", linkID) == nil {
		t.Fatal("symlink deletion accepted")
	}
	os.Symlink(t.TempDir(), filepath.Join(s.root, "linked"))
	if s.CreateBucket(ctx, "linked") == nil {
		t.Fatal("symlink bucket accepted")
	}
	if err := s.Delete(ctx, "files", m.ID); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Open(ctx, "files", m.ID); err != ErrNotFound {
		t.Fatal("delete failed")
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := s.Put(canceled, "files", strings.NewReader("data")); err == nil {
		t.Fatal("cancellation ignored")
	}
}
func TestConcurrentUploads(t *testing.T) {
	s, err := NewLocal(t.TempDir(), 100)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	s.CreateBucket(ctx, "files")
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Put(ctx, "files", strings.NewReader("data")); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	list, err := s.List(ctx, "files")
	if err != nil || len(list) != 20 {
		t.Fatal("concurrent writes lost")
	}
}
