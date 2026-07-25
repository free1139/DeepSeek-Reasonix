//go:build !arm

package botruntime

import (
	"log/slog"

	"reasonix/internal/bot"
	"reasonix/internal/bot/feishu"
	"reasonix/internal/config"
)

// newFeishuAdapter creates a Feishu/Lark bot adapter.
func newFeishuAdapter(cfg config.FeishuBotConfig, logger *slog.Logger) bot.Adapter {
	return feishu.New(cfg, logger)
}
