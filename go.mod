module github.com/amh11706/qws

go 1.26.0

replace (
	github.com/amh11706/logger => ../logger
	github.com/amh11706/qdb => ../qdb
	github.com/amh11706/qsql => ../qsql
)

require (
	github.com/amh11706/logger v0.0.0-20250302202755-66c44f6576ab
	github.com/amh11706/qdb v0.0.0-20240229163817-a0ec794d9771
	github.com/amh11706/qsql v0.0.0-20261125193255-79b51aa20aae
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
)

require (
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/go-sql-driver/mysql v1.10.2 // indirect
	github.com/jmoiron/sqlx v1.4.0 // indirect
)
