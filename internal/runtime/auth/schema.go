package auth

import _ "embed"

// SchemaSQL is copied verbatim to the generated project's 000002_auth_core.sql.
//
//go:embed schema.sql
var SchemaSQL string
