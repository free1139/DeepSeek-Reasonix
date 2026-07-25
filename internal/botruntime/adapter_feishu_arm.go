//go:build arm

package botruntime

import (
	"log/slog"

	"reasonix/internal/bot"
	"reasonix/internal/config"
)

// newFeishuAdapter returns nil on 32-bit ARM where the larksuite SDK is
// incompatible (math.MaxInt64 overflow in int). Feishu bot channels are
// unavailable on this architecture.
func newFeishuAdapter(_ config.FeishuBotConfig, _ *slog.Logger) bot.Adapter {
	return nil
}
