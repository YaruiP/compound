package core

import (
	"context"
	"time"
)

// IBlockService block service interface
type IBlockService interface {
	GetBlock(ctx context.Context, t time.Time) (int64, error)
	CurrentBlock(ctx context.Context) (int64, error)
	FormatBlockMemo(ctx context.Context, memo Action) (string, error)
	ParseBlockMemo(ctx context.Context, memoStr string) (Action, error)
}
