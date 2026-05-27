module github.com/jsanca/go-folio/inventory-service

go 1.24

require (
	github.com/jsanca/go-folio/gen v0.0.0
	github.com/mattn/go-sqlite3 v1.14.44
	google.golang.org/grpc v1.72.0
)

require (
	golang.org/x/net v0.43.0 // indirect
	golang.org/x/sys v0.35.0 // indirect
	golang.org/x/text v0.28.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250218202821-56aae31c358a // indirect
	google.golang.org/protobuf v1.36.8 // indirect
)

replace github.com/jsanca/go-folio/gen => ../gen
