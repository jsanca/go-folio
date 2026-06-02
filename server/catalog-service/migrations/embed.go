// Package migrations exposes the SQL migration files as embedded strings.
package migrations

import _ "embed"

// SQL002 is the catalog tables migration.
//
//go:embed 002_create_catalog_tables.sql
var SQL002 string

// SQL003 adds primary_image_url to catalog_products.
//
//go:embed 003_add_primary_image_url.sql
var SQL003 string
