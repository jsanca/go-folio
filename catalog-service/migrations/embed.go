// Package migrations exposes the SQL migration files as embedded strings.
package migrations

import _ "embed"

// SQL001 is the initial products table migration.
//
//go:embed 001_create_products_table.sql
var SQL001 string

// SQL002 is the catalog tables migration.
//
//go:embed 002_create_catalog_tables.sql
var SQL002 string
