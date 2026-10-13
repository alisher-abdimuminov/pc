package migrations

import "embed"

// FS holds the goose SQL migrations (non-SQL files are ignored by goose).
//
//go:embed all:*
var FS embed.FS
