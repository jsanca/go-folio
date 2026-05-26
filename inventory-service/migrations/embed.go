// Package migrations exposes the SQL migration files as embedded strings.
package migrations

import _ "embed"

// SQL001 is the initial stock and reservations tables migration.
//
//go:embed 001_create_tables.sql
var SQL001 string
