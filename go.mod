module github.com/amh11706/qws

go 1.14

replace (
	github.com/amh11706/logger => ../logger
	github.com/amh11706/qdb => ../qdb
	github.com/amh11706/qsql => ../qsql
)

require (
	github.com/amh11706/logger v0.0.0-00010101000000-000000000000
	github.com/amh11706/qdb v0.0.0-00010101000000-000000000000
	github.com/amh11706/qsql v0.0.0-00010101000000-000000000000
	nhooyr.io/websocket v1.8.7
)
