package auth

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/argon2"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Parameters struct {
	Memory      uint32
	Iterations  uint32
	Parallelism uint8
}

func DefaultParameters() Parameters { return Parameters{64 * 1024, 3, 1} }
func (p Parameters) Validate() error {
	if p.Memory < 19*1024 || p.Memory > 256*1024 || p.Iterations < 2 || p.Iterations > 6 || p.Parallelism < 1 || p.Parallelism > 4 {
		return fmt.Errorf("Argon2id parameters outside supported security/resource bounds")
	}
	return nil
}
func ValidatePassword(password string) error {
	if !utf8.ValidString(password) || utf8.RuneCountInString(password) < 15 || len(password) > 1024 {
		return ErrInvalidRequest
	}
	return nil
}

type Passwords struct {
	params Parameters
	slots  chan struct{}
}

func NewPasswords(params Parameters) (*Passwords, error) {
	if err := params.Validate(); err != nil {
		return nil, err
	}
	return &Passwords{params: params, slots: make(chan struct{}, 2)}, nil
}
func (p *Passwords) enter(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return ErrUnavailable
	}
	select {
	case p.slots <- struct{}{}:
		return nil
	default:
		return ErrBusy
	}
}
func (p *Passwords) Hash(ctx context.Context, password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	if err := p.enter(ctx); err != nil {
		return "", err
	}
	defer func() { <-p.slots }()
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", ErrUnavailable
	}
	hash := argon2.IDKey([]byte(password), salt, p.params.Iterations, p.params.Memory, p.params.Parallelism, 32)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s", p.params.Memory, p.params.Iterations, p.params.Parallelism, base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(hash)), nil
}
func decodeHash(encoded string) (Parameters, []byte, []byte, error) {
	var p Parameters
	if len(encoded) > 256 {
		return p, nil, nil, ErrInvalidCredentials
	}
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		return p, nil, nil, ErrInvalidCredentials
	}
	fields := strings.Split(parts[3], ",")
	if len(fields) != 3 {
		return p, nil, nil, ErrInvalidCredentials
	}
	nums := make([]uint64, 3)
	for i, key := range []string{"m=", "t=", "p="} {
		if !strings.HasPrefix(fields[i], key) {
			return p, nil, nil, ErrInvalidCredentials
		}
		n, err := strconv.ParseUint(strings.TrimPrefix(fields[i], key), 10, 32)
		if err != nil {
			return p, nil, nil, ErrInvalidCredentials
		}
		nums[i] = n
	}
	if nums[2] > 255 {
		return p, nil, nil, ErrInvalidCredentials
	}
	p = Parameters{uint32(nums[0]), uint32(nums[1]), uint8(nums[2])}
	if p.Validate() != nil {
		return p, nil, nil, ErrInvalidCredentials
	}
	salt, e1 := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	hash, e2 := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if e1 != nil || e2 != nil || len(salt) != 16 || len(hash) != 32 {
		return p, nil, nil, ErrInvalidCredentials
	}
	return p, salt, hash, nil
}
func (p *Passwords) Verify(ctx context.Context, password, encoded string) (bool, bool, error) {
	if len(password) > 1024 {
		return false, false, ErrInvalidCredentials
	}
	params, salt, expected, err := decodeHash(encoded)
	if err != nil {
		return false, false, err
	}
	if err := p.enter(ctx); err != nil {
		return false, false, err
	}
	defer func() { <-p.slots }()
	actual := argon2.IDKey([]byte(password), salt, params.Iterations, params.Memory, params.Parallelism, uint32(len(expected)))
	ok := subtle.ConstantTimeCompare(actual, expected) == 1
	return ok, ok && params != p.params, nil
}
