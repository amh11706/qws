package qws

import (
	"context"
	"time"

	"github.com/amh11706/logger"
	"github.com/amh11706/qdb"
	"github.com/amh11706/qsql"
)

type CommandLog interface {
	Status(result string)
	End(ctx context.Context)
}

type CommandLogger struct {
	table   qsql.Table
	columns []string
}

func NewCommandLogger(tableName string, sample any) *CommandLogger {
	return &CommandLogger{table: qsql.NewTable(&qdb.DB, tableName), columns: qsql.GetColumns(sample, true)}
}

func (l *CommandLogger) Start(c UserInfoer, cmd string, params string) CommandLog {
	return &commandLog{UserId: c.UserId(), LobbyId: c.InLobby(), logger: l, Command: cmd, Params: params, startTime: time.Now()}
}

type commandLog struct {
	logger    *CommandLogger
	startTime time.Time
	UserId    int64         `db:"user_id"`
	LobbyId   int64         `db:"lobby_id"`
	Duration  time.Duration `db:"duration"`
	Command   string        `db:"command"`
	Params    string        `db:"params"`
	Result    string        `db:"result"`
}

func (l *commandLog) Status(result string) {
	l.Result = result
}

func (l *commandLog) End(ctx context.Context) {
	l.Duration = time.Since(l.startTime)
	if l.Result == "" {
		l.Result = "Error"
	}
	_, err := l.logger.table.Create(ctx, l, l.logger.columns...)
	logger.Check(err)
}
