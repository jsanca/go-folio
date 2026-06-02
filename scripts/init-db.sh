#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "postgres" <<-EOSQL
    SELECT 'CREATE DATABASE folio_catalog'
      WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'folio_catalog')\gexec
    SELECT 'CREATE DATABASE folio_inventory'
      WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'folio_inventory')\gexec
EOSQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "folio_catalog" <<-EOSQL
    ALTER TABLE catalog_products ADD COLUMN IF NOT EXISTS primary_image_url TEXT NOT NULL DEFAULT '';
EOSQL
