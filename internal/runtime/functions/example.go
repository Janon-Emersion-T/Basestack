package functions

import "context"

// Examples is owner-editable generated Go source. Registration is deterministic;
// BaseStack never scans and executes arbitrary scripts from the filesystem.
func Examples() []Definition {
	return []Definition{{Metadata: Metadata{Name: "hello", Public: true}, Handle: func(ctx context.Context, r Request) (any, error) {
		return map[string]string{"message": "Hello from BaseStack"}, nil
	}}, {Metadata: Metadata{Name: "whoami", Permission: "profile.read"}, Handle: func(ctx context.Context, r Request) (any, error) { return r.User, nil }}}
}
