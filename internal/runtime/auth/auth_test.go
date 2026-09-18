package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func smallParameters() Parameters { return Parameters{19 * 1024, 2, 1} }
func TestPasswords(t *testing.T) {
	p, err := NewPasswords(smallParameters())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	password := "correct horse battery staple"
	hash, err := p.Hash(ctx, password)
	if err != nil {
		t.Fatal(err)
	}
	other, err := p.Hash(ctx, password)
	if err != nil || hash == other {
		t.Fatal("salts not randomized")
	}
	ok, rehash, err := p.Verify(ctx, password, hash)
	if err != nil || !ok || rehash {
		t.Fatal("password verification failed")
	}
	ok, _, err = p.Verify(ctx, "incorrect passphrase", hash)
	if err != nil || ok {
		t.Fatal("wrong password accepted")
	}
	stronger, _ := NewPasswords(Parameters{20 * 1024, 2, 1})
	ok, rehash, err = stronger.Verify(ctx, password, hash)
	if err != nil || !ok || !rehash {
		t.Fatal("rehash not detected")
	}
	for _, bad := range []string{"", strings.Repeat("a", 14), strings.Repeat("a", 1025)} {
		if _, err := p.Hash(ctx, bad); err == nil {
			t.Fatal("bad password accepted")
		}
	}
	if _, err := p.Hash(ctx, strings.Repeat("界", 15)); err != nil {
		t.Fatal("Unicode passphrase rejected")
	}
	for _, bad := range []string{"bad", strings.Replace(hash, "m=19456", "m=4294967295", 1), strings.Replace(hash, "p=1", "p=256", 1), strings.Replace(hash, "v=19", "v=0", 1), hash + "="} {
		if ok, _, _ := p.Verify(ctx, password, bad); ok {
			t.Fatal("malformed hash accepted")
		}
	}
	p.slots <- struct{}{}
	p.slots <- struct{}{}
	if _, err := p.Hash(ctx, password); err != ErrBusy {
		t.Fatal("unbounded hash concurrency")
	}
	<-p.slots
	<-p.slots
}
func TestEmailNormalization(t *testing.T) {
	display, normalized, err := NormalizeEmail("  User+tag@EXAMPLE.COM ")
	if err != nil || display != "User+tag@EXAMPLE.COM" || normalized != "user+tag@example.com" {
		t.Fatal("email normalization failed")
	}
	for _, email := range []string{"bad", "Name <user@example.com>", "a..b@example.com", ".a@example.com", "a@-bad.com", "a@example.com\nBcc: x@test.com", strings.Repeat("a", 65) + "@example.com", "用户@example.com"} {
		if _, _, err := NormalizeEmail(email); err == nil {
			t.Fatal("invalid email accepted")
		}
	}
}
func TestChallengesAndPublicModels(t *testing.T) {
	token, c, err := NewChallenge(VerifyEmail, time.Hour)
	if err != nil || len(token) != 43 || len(c.Digest) != 32 {
		t.Fatal("bad challenge")
	}
	digest, err := TokenDigest(token)
	if err != nil || !bytes.Equal(digest, c.Digest) || bytes.Equal([]byte(token), digest) {
		t.Fatal("token not digested")
	}
	other, _, _ := NewChallenge(VerifyEmail, time.Hour)
	if token == other {
		t.Fatal("predictable challenge")
	}
	if _, err := TokenDigest("invalid"); err == nil {
		t.Fatal("invalid token accepted")
	}
	id, _ := UUID()
	if !uuidPattern.MatchString(id) {
		t.Fatal("bad UUID")
	}
	user := User{ID: id, Email: "public@example.com", PasswordHash: "SECRET_HASH", Status: "active"}
	for _, value := range []any{user, user.Public()} {
		data, _ := json.Marshal(value)
		if strings.Contains(string(data), "SECRET") || strings.Contains(string(data), "password") {
			t.Fatal("user secrets serialized")
		}
	}
}
func TestLimiter(t *testing.T) {
	l := NewLimiter()
	now := time.Now()
	l.now = func() time.Time { return now }
	for i := 0; i < 5; i++ {
		if ok, _ := l.Allow("client", 5); !ok {
			t.Fatal("early rate limit")
		}
	}
	if ok, retry := l.Allow("client", 5); ok || retry <= 0 {
		t.Fatal("limit not enforced")
	}
	now = now.Add(time.Minute)
	if ok, _ := l.Allow("client", 5); !ok {
		t.Fatal("window did not expire")
	}
	l.maximum = 1
	if ok, _ := l.Allow("different", 5); ok {
		t.Fatal("limiter memory not bounded")
	}
	concurrent := NewLimiter()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); concurrent.Allow("same", 10) }()
	}
	wg.Wait()
	if concurrent.buckets["same"].count != 10 {
		t.Fatal("rate-limit race")
	}
}
func TestLocalDeliverySafety(t *testing.T) {
	if _, err := NewLocalDelivery(t.TempDir(), "production"); err == nil {
		t.Fatal("production outbox allowed")
	}
	dir := t.TempDir()
	d, err := NewLocalDelivery(dir, "development")
	if err != nil {
		t.Fatal(err)
	}
	message := Message{Email: "test@example.com", Purpose: VerifyEmail, Token: "TEST_SECRET", ExpiresAt: time.Now().Add(time.Hour)}
	if err := d.Deliver(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	entries, _ := os.ReadDir(filepath.Join(dir, ".basestack", "auth-outbox"))
	if len(entries) != 1 {
		t.Fatal("delivery missing")
	}
	info, _ := entries[0].Info()
	if info.Mode().Perm() != 0600 {
		t.Fatal("outbox permissions")
	}
	if !d.DevelopmentOnly() {
		t.Fatal("outbox not flagged as development")
	}
	if _, err := NewService(&Repository{}, Options{Environment: "production", Parameters: smallParameters(), VerificationTTL: time.Hour, ResetTTL: time.Minute}, d); err == nil {
		t.Fatal("production accepted dev provider")
	}
}
