package plugin

import (
	"context"
	"log/slog"

	"github.com/linuxsuren/atest-ext-ai/pkg/logging"
)

func loggerFromContext(ctx context.Context) *slog.Logger {
	return logging.FromContext(ctx)
}
