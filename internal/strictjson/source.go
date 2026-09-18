package strictjson

import "embed"

// Source is copied into generated applications; source.go itself is excluded.
//
//go:embed json.go
var Source embed.FS
