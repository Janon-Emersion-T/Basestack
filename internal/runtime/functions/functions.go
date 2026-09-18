// Package functions registers trusted, compiled application functions.
package functions

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Janon-Emersion-T/Basestack/internal/runtime/auth"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/rbac"
)

type Metadata struct {
	Name       string `json:"name"`
	Public     bool   `json:"public"`
	Permission string `json:"permission,omitempty"`
}
type Request struct {
	Body        json.RawMessage
	User        auth.PublicUser
	Environment func(string) (string, bool)
}
type Handler func(context.Context, Request) (any, error)
type Definition struct {
	Metadata
	Handle Handler
}
type Registry struct{ definitions map[string]Definition }

var namePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,49}$`)

func New(definitions []Definition) (*Registry, error) {
	r := &Registry{map[string]Definition{}}
	for _, d := range definitions {
		if !namePattern.MatchString(d.Name) || d.Handle == nil || (!d.Public && !rbac.ValidName(d.Permission)) || (d.Public && d.Permission != "") {
			return nil, errors.New("invalid function definition; private functions require a permission")
		}
		if _, ok := r.definitions[d.Name]; ok {
			return nil, errors.New("duplicate function definition")
		}
		r.definitions[d.Name] = d
	}
	return r, nil
}
func (r *Registry) List() []Metadata {
	result := []Metadata{}
	for _, d := range r.definitions {
		result = append(result, d.Metadata)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}
func (r *Registry) Inspect(name string) (Metadata, bool) {
	d, ok := r.definitions[name]
	return d.Metadata, ok
}

func problem(w http.ResponseWriter, status int, code string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": "Function request could not be completed."}})
}
func (r *Registry) HTTP(sessions auth.Sessions, permissions rbac.Authorizer, environment map[string]string) http.Handler {
	// Copy at construction; concurrent handlers only read immutable configuration.
	values := map[string]string{}
	for k, v := range environment {
		values[k] = v
	}
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		defer func() {
			if recover() != nil {
				problem(w, 500, "function_failed")
			}
		}()
		d, ok := r.definitions[strings.TrimPrefix(req.URL.Path, "/api/functions/")]
		if !ok {
			problem(w, 404, "not_found")
			return
		}
		if req.Method != "POST" {
			w.Header().Set("Allow", "POST, OPTIONS")
			problem(w, 405, "method_not_allowed")
			return
		}
		ctx, cancel := context.WithTimeout(req.Context(), 8*time.Second)
		defer cancel()
		var user auth.PublicUser
		if !d.Public {
			if sessions == nil || permissions == nil {
				problem(w, 503, "authorization_unavailable")
				return
			}
			var err error
			user, err = sessions.Current(ctx, auth.Bearer(req))
			if err != nil {
				problem(w, 401, "unauthorized")
				return
			}
			if err = permissions.Check(ctx, user.ID, d.Permission); err != nil {
				if errors.Is(err, rbac.ErrDenied) {
					problem(w, 403, "forbidden")
				} else {
					problem(w, 503, "authorization_unavailable")
				}
				return
			}
		}
		body, err := io.ReadAll(http.MaxBytesReader(w, req.Body, 1<<20))
		if err != nil {
			problem(w, 413, "request_too_large")
			return
		}
		if len(body) == 0 {
			body = []byte(`{}`)
		}
		if !json.Valid(body) {
			problem(w, 400, "invalid_json")
			return
		}
		value, err := d.Handle(ctx, Request{body, user, func(key string) (string, bool) { v, ok := values[key]; return v, ok }})
		if err != nil || ctx.Err() != nil {
			problem(w, 500, "function_failed")
			return
		}
		encoded, err := json.Marshal(map[string]any{"data": value})
		if err != nil || len(encoded) > 1<<20 {
			problem(w, 500, "function_failed")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(encoded)
	})
}
