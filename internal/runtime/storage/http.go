package storage

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Janon-Emersion-T/Basestack/internal/runtime/auth"
	"github.com/Janon-Emersion-T/Basestack/internal/runtime/rbac"
)

// HTTP never serves filesystem paths directly. All objects require a session
// and an exact storage.<bucket>.read or storage.<bucket>.write permission.
func HTTP(store Store, sessions auth.Sessions, permissions rbac.Authorizer, maximum int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fail := func(status int) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": "storage_request_failed", "message": "Storage request could not be completed."}})
		}
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/storage/"), "/")
		if len(parts) < 1 || len(parts) > 2 || !bucketName.MatchString(parts[0]) || (len(parts) == 2 && !objectID.MatchString(parts[1])) {
			fail(400)
			return
		}
		if (r.Method != "GET" && r.Method != "POST" && r.Method != "DELETE") || (r.Method == "POST" && len(parts) != 1) || (r.Method == "DELETE" && len(parts) != 2) {
			w.Header().Set("Allow", "GET, POST, DELETE, OPTIONS")
			fail(405)
			return
		}
		if sessions == nil || permissions == nil {
			fail(503)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		user, err := sessions.Current(ctx, auth.Bearer(r))
		if err != nil {
			fail(401)
			return
		}
		action := "read"
		if r.Method != "GET" {
			action = "write"
		}
		if err := permissions.Check(ctx, user.ID, "storage."+parts[0]+"."+action); err != nil {
			if errors.Is(err, rbac.ErrDenied) {
				fail(403)
			} else {
				fail(503)
			}
			return
		}
		var data any
		switch r.Method {
		case "POST":
			data, err = store.Put(ctx, parts[0], http.MaxBytesReader(w, r.Body, maximum+1))
		case "DELETE":
			err = store.Delete(ctx, parts[0], parts[1])
			data = map[string]bool{"deleted": err == nil}
		case "GET":
			if len(parts) == 1 {
				data, err = store.List(ctx, parts[0])
				break
			}
			var f io.ReadCloser
			f, _, err = store.Open(ctx, parts[0], parts[1])
			if err != nil {
				break
			}
			defer f.Close()
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Disposition", `attachment; filename="`+parts[1]+`"`)
			_, _ = io.Copy(w, f)
			return
		}
		if err != nil {
			switch {
			case errors.Is(err, ErrNotFound):
				fail(404)
			case errors.Is(err, ErrTooLarge):
				fail(413)
			default:
				fail(503)
			}
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
	})
}
