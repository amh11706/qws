module github.com/amh11706/qws

go 1.22.0

// replace (
// 	github.com/amh11706/logger => ../logger
// 	github.com/amh11706/qdb => ../qdb
// 	github.com/amh11706/qsql => ../qsql
// )

require (
	github.com/amh11706/logger v0.0.0-20240228210936-9df6d23b8ea9
	github.com/amh11706/qdb v0.0.0-20201108153937-e79024dfa7f6
	github.com/amh11706/qsql v0.0.0-20220123094420-b9b581d9642f
	github.com/gorilla/websocket v1.5.1
)

require (
	github.com/go-sql-driver/mysql v1.7.1 // indirect
	github.com/jmoiron/sqlx v1.3.5 // indirect
	golang.org/x/net v0.17.0 // indirect
)
