// Package runtime embeds the tested application runtime for source-owned generated projects.
package runtime

import "embed"

// Source includes runtime tests; generated applications can run go test ./... independently.
//
//go:embed config database migrations server app auth command privatefs rbac storage functions module.txt sums.txt
var Source embed.FS
