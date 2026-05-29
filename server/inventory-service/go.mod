module github.com/jsanca/go-folio/inventory-service

go 1.25.0

require (
	github.com/jackc/pgx/v5 v5.9.2
	github.com/jsanca/go-folio/gen v0.0.0
	github.com/mattn/go-sqlite3 v1.14.44
	google.golang.org/grpc v1.72.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sync v0.17.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.29.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/protobuf v1.36.8 // indirect
)

replace github.com/jsanca/go-folio/gen => ../gen
