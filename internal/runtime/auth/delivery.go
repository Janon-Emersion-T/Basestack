package auth

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Message contains a raw secret solely for delivery. Never pass it to logs or API responses.
type Message struct {
	Email     string    `json:"email"`
	Purpose   string    `json:"purpose"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
}
type Delivery interface {
	Deliver(context.Context, Message) error
	DevelopmentOnly() bool
}
type LocalDelivery struct{ dir string }

func NewLocalDelivery(projectDir, environment string) (*LocalDelivery, error) {
	if environment != "development" {
		return nil, ErrUnavailable
	}
	parent := filepath.Join(projectDir, ".basestack")
	dir := filepath.Join(parent, "auth-outbox")
	for _, path := range []string{parent, dir} {
		if err := os.Mkdir(path, 0700); err != nil && !os.IsExist(err) {
			return nil, ErrUnavailable
		}
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
			return nil, ErrUnavailable
		}
	}
	return &LocalDelivery{dir: dir}, nil
}
func (d *LocalDelivery) DevelopmentOnly() bool { return true }
func (d *LocalDelivery) Deliver(ctx context.Context, message Message) error {
	if ctx.Err() != nil {
		return ErrUnavailable
	}
	id, err := UUID()
	if err != nil {
		return err
	}
	b, err := json.Marshal(message)
	if err != nil {
		return ErrUnavailable
	}
	f, err := os.OpenFile(filepath.Join(d.dir, id+".json"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return ErrUnavailable
	}
	ok := false
	defer func() {
		f.Close()
		if !ok {
			os.Remove(f.Name())
		}
	}()
	if _, err = f.Write(append(b, '\n')); err != nil {
		return ErrUnavailable
	}
	if err = f.Sync(); err != nil {
		return ErrUnavailable
	}
	if err = f.Close(); err != nil {
		return ErrUnavailable
	}
	ok = true
	return nil
}
