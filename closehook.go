package qws

import (
	"context"
)

type CloseHandler *func(ctx context.Context, c *UserConn)

func NewCloseHandler(f func(ctx context.Context, c *UserConn)) CloseHandler {
	if f == nil {
		return nil
	}
	return CloseHandler(&f)
}
